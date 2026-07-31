package herculespackage

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"time"

	"github.com/jakthom/hercules/pkg/db"
	"github.com/jakthom/hercules/pkg/metric"
	"github.com/jakthom/hercules/pkg/source"
	"golang.org/x/mod/semver"
	"sigs.k8s.io/yaml"
)

const (
	CurrentAPIVersion      = "hercules.dev/v1alpha1"
	PackageKind            = "Package"
	DefaultHTTPTimeout     = 15 * time.Second
	DefaultMaxPackageBytes = int64(2 << 20)
	SHA256Prefix           = "sha256:"
)

var (
	packageNamePattern  = regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_-]*$`)
	variableNamePattern = regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_]*$`)
	variablePattern     = regexp.MustCompile(`\{\{\s*([a-zA-Z_][a-zA-Z0-9_]*)\s*\}\}`)
)

type Variables map[string]string

type Package struct {
	APIVersion         string             `json:"apiVersion"`
	Kind               string             `json:"kind"`
	Name               string             `json:"name"`
	Version            string             `json:"version"`
	MinHerculesVersion string             `json:"minHerculesVersion,omitempty"`
	Dependencies       []string           `json:"dependencies,omitempty"`
	Variables          Variables          `json:"variables,omitempty"`
	Extensions         db.Extensions      `json:"extensions,omitempty"`
	Macros             []db.Macro         `json:"macros,omitempty"`
	Sources            []source.Source    `json:"sources,omitempty"`
	Metrics            metric.Definitions `json:"metrics,omitempty"`
	Metadata           metric.Metadata    `json:"-"`
}

type PackageConfig struct {
	Package      string        `json:"package"`
	Variables    Variables     `json:"variables,omitempty"`
	MetricPrefix metric.Prefix `json:"metricPrefix,omitempty"`
	SHA256       string        `json:"sha256,omitempty"`
}

type LoadPolicy struct {
	AllowRemote   bool
	AllowHTTP     bool
	RequireSHA256 bool
	AllowedHosts  []string
	MaxBytes      int64
	CacheDir      string
	Offline       bool
}

func (p PackageConfig) Load(
	ctx context.Context,
	baseDir string,
	client *http.Client,
	policy LoadPolicy,
	herculesVersion string,
) (Package, error) {
	if strings.TrimSpace(p.Package) == "" {
		return Package{}, errors.New("package location must not be empty")
	}
	if policy.MaxBytes <= 0 {
		policy.MaxBytes = DefaultMaxPackageBytes
	}
	if strings.TrimSpace(p.SHA256) != "" {
		if _, err := normalizeDigest(p.SHA256); err != nil {
			return Package{}, fmt.Errorf("package %q has invalid sha256: %w", p.Package, err)
		}
	}
	if isRemoteLocation(p.Package) && policy.RequireSHA256 && strings.TrimSpace(p.SHA256) == "" {
		return Package{}, fmt.Errorf("remote package %q requires a sha256 digest", p.Package)
	}
	client = packageHTTPClient(client, policy)

	payload, packageBase, err := loadPackageBytes(ctx, p.Package, p.SHA256, baseDir, client, policy)
	if err != nil {
		return Package{}, err
	}
	if err := verifyDigest(payload, p.SHA256); err != nil {
		return Package{}, fmt.Errorf("verify package %q: %w", p.Package, err)
	}
	if isRemoteLocation(p.Package) && !policy.Offline {
		if err := cachePackage(payload, p.SHA256, policy.CacheDir); err != nil {
			return Package{}, fmt.Errorf("cache package %q: %w", p.Package, err)
		}
	}

	configuredPackage := Package{}
	if err := yaml.UnmarshalStrict(payload, &configuredPackage); err != nil {
		return Package{}, fmt.Errorf("decode package %q: %w", p.Package, err)
	}
	configuredPackage.Variables, err = mergeVariables(configuredPackage.Variables, p.Variables)
	if err != nil {
		return Package{}, fmt.Errorf("resolve variables for package %q: %w", configuredPackage.Name, err)
	}
	if err := expandPackage(&configuredPackage); err != nil {
		return Package{}, err
	}
	resolveSourcePaths(&configuredPackage, packageBase)
	if err := configuredPackage.Validate(herculesVersion); err != nil {
		return Package{}, err
	}
	configuredPackage.Metadata.Prefix = p.MetricPrefix
	return configuredPackage, nil
}

func packageHTTPClient(client *http.Client, policy LoadPolicy) *http.Client {
	if client == nil {
		client = &http.Client{Timeout: DefaultHTTPTimeout}
	}
	configuredClient := *client
	configuredRedirect := configuredClient.CheckRedirect
	configuredClient.CheckRedirect = func(request *http.Request, via []*http.Request) error {
		if err := validatePackageRedirect(request.URL, policy); err != nil {
			return err
		}
		if configuredRedirect != nil {
			return configuredRedirect(request, via)
		}
		if len(via) >= 10 {
			return errors.New("stopped after 10 package redirects")
		}
		return nil
	}
	return &configuredClient
}

func validatePackageRedirect(location *url.URL, policy LoadPolicy) error {
	if location.Scheme != "https" && (location.Scheme != "http" || !policy.AllowHTTP) {
		return errors.New("redirected package URL must use HTTPS")
	}
	if !slices.Contains(policy.AllowedHosts, location.Hostname()) {
		return fmt.Errorf("redirected package host %q is not allowed", location.Hostname())
	}
	return nil
}

func loadPackageBytes(
	ctx context.Context,
	location string,
	expectedDigest string,
	baseDir string,
	client *http.Client,
	policy LoadPolicy,
) ([]byte, string, error) {
	if !isRemoteLocation(location) {
		return loadBytes(ctx, location, baseDir, client, policy)
	}
	if err := validateRemoteLocation(location, policy); err != nil {
		return nil, "", err
	}
	packageBase, err := remotePackageBase(location)
	if err != nil {
		return nil, "", err
	}
	if policy.Offline {
		payload, cacheErr := readCachedPackage(expectedDigest, policy.CacheDir, policy.MaxBytes)
		if cacheErr != nil {
			return nil, "", fmt.Errorf("load remote package %q in offline mode: %w", location, cacheErr)
		}
		return payload, packageBase, nil
	}
	payload, loadedBase, loadErr := loadBytes(ctx, location, baseDir, client, policy)
	if loadErr == nil {
		return payload, loadedBase, nil
	}
	if expectedDigest == "" || policy.CacheDir == "" {
		return nil, "", loadErr
	}
	payload, cacheErr := readCachedPackage(expectedDigest, policy.CacheDir, policy.MaxBytes)
	if cacheErr != nil {
		return nil, "", errors.Join(loadErr, fmt.Errorf("read package cache: %w", cacheErr))
	}
	return payload, packageBase, nil
}

func remotePackageBase(location string) (string, error) {
	parsed, err := url.Parse(location)
	if err != nil {
		return "", fmt.Errorf("parse package location %q: %w", location, err)
	}
	return parsed.ResolveReference(&url.URL{Path: "."}).String(), nil
}

func (p Package) Validate(herculesVersion string) error {
	var errs []error
	if p.APIVersion != CurrentAPIVersion {
		errs = append(errs, fmt.Errorf("package %q uses unsupported apiVersion %q", p.Name, p.APIVersion))
	}
	if p.Kind != PackageKind {
		errs = append(errs, fmt.Errorf("package %q uses unsupported kind %q", p.Name, p.Kind))
	}
	if !packageNamePattern.MatchString(p.Name) {
		errs = append(errs, fmt.Errorf("invalid package name %q", p.Name))
	}
	if strings.TrimSpace(p.Version) == "" {
		errs = append(errs, fmt.Errorf("package %q version must not be empty", p.Name))
	}
	seenDependencies := make(map[string]struct{}, len(p.Dependencies))
	for _, dependency := range p.Dependencies {
		if !packageNamePattern.MatchString(dependency) {
			errs = append(errs, fmt.Errorf("package %q has invalid dependency name %q", p.Name, dependency))
		}
		if _, exists := seenDependencies[dependency]; exists {
			errs = append(errs, fmt.Errorf("package %q repeats dependency %q", p.Name, dependency))
		}
		seenDependencies[dependency] = struct{}{}
	}
	for name := range p.Variables {
		if !variableNamePattern.MatchString(name) {
			errs = append(errs, fmt.Errorf("package %q has invalid variable name %q", p.Name, name))
		}
	}
	if p.MinHerculesVersion != "" && herculesVersion != "" {
		required := normalizeVersion(p.MinHerculesVersion)
		current := normalizeVersion(herculesVersion)
		if !semver.IsValid(required) {
			errs = append(errs, fmt.Errorf("package %q has invalid minHerculesVersion %q", p.Name, p.MinHerculesVersion))
		} else if !semver.IsValid(current) {
			errs = append(errs, fmt.Errorf("hercules has invalid version %q", herculesVersion))
		} else if semver.Compare(current, required) < 0 {
			errs = append(errs, fmt.Errorf("package %q requires Hercules %s or newer", p.Name, p.MinHerculesVersion))
		}
	}
	for _, configuredSource := range p.Sources {
		if err := configuredSource.Validate(); err != nil {
			errs = append(errs, fmt.Errorf("package %q: %w", p.Name, err))
		}
	}
	for _, configuredMacro := range p.Macros {
		if err := configuredMacro.Validate(); err != nil {
			errs = append(errs, fmt.Errorf("package %q: %w", p.Name, err))
		}
	}
	if err := p.Metrics.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("package %q: %w", p.Name, err))
	}
	return errors.Join(errs...)
}

func Order(packages []Package) ([]Package, error) {
	byName := make(map[string]Package, len(packages))
	for _, configuredPackage := range packages {
		name := configuredPackage.Name
		if _, exists := byName[name]; exists {
			return nil, fmt.Errorf("duplicate package name %q", name)
		}
		byName[name] = configuredPackage
	}

	ordered := make([]Package, 0, len(packages))
	state := make(map[string]uint8, len(packages))
	var visit func(string) error
	visit = func(name string) error {
		switch state[name] {
		case 1:
			return fmt.Errorf("package dependency cycle contains %q", name)
		case 2:
			return nil
		}
		configuredPackage, exists := byName[name]
		if !exists {
			return fmt.Errorf("package dependency %q is not configured", name)
		}
		state[name] = 1
		for _, dependency := range configuredPackage.Dependencies {
			if err := visit(dependency); err != nil {
				return fmt.Errorf("package %q: %w", name, err)
			}
		}
		state[name] = 2
		ordered = append(ordered, configuredPackage)
		return nil
	}
	for _, configuredPackage := range packages {
		if err := visit(configuredPackage.Name); err != nil {
			return nil, err
		}
	}
	return ordered, validateCatalogNames(ordered)
}

func loadBytes(
	ctx context.Context,
	location string,
	baseDir string,
	client *http.Client,
	policy LoadPolicy,
) ([]byte, string, error) {
	parsed, err := url.Parse(location)
	if err != nil {
		return nil, "", fmt.Errorf("parse package location %q: %w", location, err)
	}
	if parsed.Scheme == "http" || parsed.Scheme == "https" {
		if !policy.AllowRemote {
			return nil, "", fmt.Errorf("remote package %q is disabled by security policy", location)
		}
		if parsed.Scheme != "https" && !policy.AllowHTTP {
			return nil, "", fmt.Errorf("remote package %q must use HTTPS", location)
		}
		if len(policy.AllowedHosts) == 0 || !slices.Contains(policy.AllowedHosts, parsed.Hostname()) {
			return nil, "", fmt.Errorf("remote package host %q is not allowed", parsed.Hostname())
		}
		request, err := http.NewRequestWithContext(ctx, http.MethodGet, parsed.String(), nil)
		if err != nil {
			return nil, "", fmt.Errorf("create package request: %w", err)
		}
		response, err := client.Do(request)
		if err != nil {
			return nil, "", fmt.Errorf("fetch package %q: %w", location, err)
		}
		if response.Request.URL.Scheme != "https" && !policy.AllowHTTP {
			closeErr := response.Body.Close()
			return nil, "", errors.Join(
				fmt.Errorf("remote package final URL must use HTTPS"),
				closeErr,
			)
		}
		if !slices.Contains(policy.AllowedHosts, response.Request.URL.Hostname()) {
			closeErr := response.Body.Close()
			return nil, "", errors.Join(
				fmt.Errorf("remote package redirect host %q is not allowed", response.Request.URL.Hostname()),
				closeErr,
			)
		}
		if response.StatusCode < 200 || response.StatusCode >= 300 {
			closeErr := response.Body.Close()
			return nil, "", errors.Join(
				fmt.Errorf("fetch package %q: unexpected HTTP status %s", location, response.Status),
				closeErr,
			)
		}
		payload, readErr := readLimited(response.Body, policy.MaxBytes)
		closeErr := response.Body.Close()
		if combinedErr := errors.Join(readErr, closeErr); combinedErr != nil {
			return nil, "", fmt.Errorf("read package %q: %w", location, combinedErr)
		}
		return payload, response.Request.URL.ResolveReference(&url.URL{Path: "."}).String(), nil
	}
	if parsed.Scheme != "" && parsed.Scheme != "file" {
		return nil, "", fmt.Errorf("unsupported package scheme %q", parsed.Scheme)
	}
	path := location
	if parsed.Scheme == "file" {
		path = parsed.Path
	}
	if !filepath.IsAbs(path) {
		path = filepath.Join(baseDir, path)
	}
	absPath, err := filepath.Abs(path)
	if err != nil {
		return nil, "", fmt.Errorf("resolve package path %q: %w", path, err)
	}
	payload, err := os.ReadFile(absPath)
	if err != nil {
		return nil, "", fmt.Errorf("read package %q: %w", absPath, err)
	}
	if int64(len(payload)) > policy.MaxBytes {
		return nil, "", fmt.Errorf("package %q exceeds maximum of %d bytes", absPath, policy.MaxBytes)
	}
	return payload, filepath.Dir(absPath), nil
}

func readLimited(reader io.Reader, maximum int64) ([]byte, error) {
	payload, err := io.ReadAll(io.LimitReader(reader, maximum+1))
	if err != nil {
		return nil, err
	}
	if int64(len(payload)) > maximum {
		return nil, fmt.Errorf("response exceeds maximum of %d bytes", maximum)
	}
	return payload, nil
}

func verifyDigest(payload []byte, expected string) error {
	if expected == "" {
		return nil
	}
	normalized, err := normalizeDigest(expected)
	if err != nil {
		return err
	}
	actual := fmt.Sprintf("%x", sha256.Sum256(payload))
	if normalized != actual {
		return fmt.Errorf("SHA-256 mismatch: expected %s, got %s", normalized, actual)
	}
	return nil
}

func normalizeDigest(expected string) (string, error) {
	normalized := strings.TrimPrefix(strings.ToLower(strings.TrimSpace(expected)), SHA256Prefix)
	if len(normalized) != sha256.Size*2 {
		return "", fmt.Errorf("SHA-256 digest must contain %d hexadecimal characters", sha256.Size*2)
	}
	if _, err := hex.DecodeString(normalized); err != nil {
		return "", fmt.Errorf("decode SHA-256 digest: %w", err)
	}
	return normalized, nil
}

func readCachedPackage(expected string, cacheDir string, maximum int64) ([]byte, error) {
	cachePath, err := packageCachePath(expected, cacheDir)
	if err != nil {
		return nil, err
	}
	payload, err := os.ReadFile(cachePath)
	if err != nil {
		return nil, err
	}
	if int64(len(payload)) > maximum {
		return nil, fmt.Errorf("cached package exceeds maximum of %d bytes", maximum)
	}
	return payload, nil
}

func cachePackage(payload []byte, expected string, cacheDir string) error {
	if expected == "" || cacheDir == "" {
		return nil
	}
	cachePath, err := packageCachePath(expected, cacheDir)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(cacheDir, 0o700); err != nil {
		return err
	}
	temporary, err := os.CreateTemp(cacheDir, ".hercules-package-*")
	if err != nil {
		return err
	}
	temporaryPath := temporary.Name()
	defer func() { _ = os.Remove(temporaryPath) }()
	if err := temporary.Chmod(0o600); err != nil {
		_ = temporary.Close()
		return err
	}
	_, writeErr := temporary.Write(payload)
	closeErr := temporary.Close()
	if err := errors.Join(writeErr, closeErr); err != nil {
		return err
	}
	return os.Rename(temporaryPath, cachePath)
}

func packageCachePath(expected string, cacheDir string) (string, error) {
	if strings.TrimSpace(cacheDir) == "" {
		return "", errors.New("package cache directory is not configured")
	}
	normalized, err := normalizeDigest(expected)
	if err != nil {
		return "", fmt.Errorf("package cache requires a valid digest: %w", err)
	}
	return filepath.Join(cacheDir, normalized+".yml"), nil
}

func mergeVariables(defaults Variables, overrides Variables) (Variables, error) {
	merged := make(Variables, len(defaults)+len(overrides))
	for name, value := range defaults {
		merged[name] = value
	}
	for name, value := range overrides {
		if strings.HasPrefix(value, "$") {
			environmentName := strings.TrimPrefix(value, "$")
			resolved, exists := os.LookupEnv(environmentName)
			if !exists {
				return nil, fmt.Errorf("environment variable %q required by variable %q is not set", environmentName, name)
			}
			value = resolved
		}
		merged[name] = value
	}
	return merged, nil
}

func expandPackage(configuredPackage *Package) error {
	expand := func(value string) (string, error) {
		var expansionError error
		result := variablePattern.ReplaceAllStringFunc(value, func(match string) string {
			parts := variablePattern.FindStringSubmatch(match)
			replacement, exists := configuredPackage.Variables[parts[1]]
			if !exists {
				expansionError = fmt.Errorf("package %q requires undefined variable %q", configuredPackage.Name, parts[1])
				return match
			}
			return replacement
		})
		return result, expansionError
	}
	for i := range configuredPackage.Macros {
		value, err := expand(string(configuredPackage.Macros[i].SQL))
		if err != nil {
			return err
		}
		configuredPackage.Macros[i].SQL = db.SQL(value)
	}
	for i := range configuredPackage.Sources {
		value, err := expand(configuredPackage.Sources[i].Source)
		if err != nil {
			return err
		}
		configuredPackage.Sources[i].Source = value
	}
	return configuredPackage.Metrics.ExpandVariables(expand)
}

func resolveSourcePaths(configuredPackage *Package, packageBase string) {
	if strings.HasPrefix(packageBase, "http://") || strings.HasPrefix(packageBase, "https://") {
		baseURL, err := url.Parse(packageBase)
		if err != nil {
			return
		}
		for i := range configuredPackage.Sources {
			configuredSource := &configuredPackage.Sources[i]
			if configuredSource.Type == source.SQLSourceType {
				continue
			}
			reference, err := url.Parse(configuredSource.Source)
			if err == nil && reference.Scheme == "" {
				configuredSource.Source = baseURL.ResolveReference(reference).String()
			}
		}
		return
	}
	for i := range configuredPackage.Sources {
		configuredSource := &configuredPackage.Sources[i]
		if configuredSource.Type == source.SQLSourceType || filepath.IsAbs(configuredSource.Source) {
			continue
		}
		parsed, err := url.Parse(configuredSource.Source)
		if err == nil && parsed.Scheme != "" {
			continue
		}
		configuredSource.Source = filepath.Join(packageBase, configuredSource.Source)
	}
}

func validateCatalogNames(packages []Package) error {
	objects := make(map[string]string)
	macros := make(map[string]string)
	metrics := make(map[string]string)
	for _, configuredPackage := range packages {
		owner := configuredPackage.Name
		for _, configuredSource := range configuredPackage.Sources {
			lookupName := strings.ToLower(configuredSource.Name)
			if previous, exists := objects[lookupName]; exists {
				return fmt.Errorf("source %q is declared by packages %q and %q", configuredSource.Name, previous, owner)
			}
			objects[lookupName] = owner
		}
		for _, configuredMacro := range configuredPackage.Macros {
			lookupName := strings.ToLower(configuredMacro.Name)
			if previous, exists := macros[lookupName]; exists {
				return fmt.Errorf("macro %q is declared by packages %q and %q", configuredMacro.Name, previous, owner)
			}
			macros[lookupName] = owner
		}
		for _, name := range configuredPackage.Metrics.OutputNames(configuredPackage.Metadata) {
			if previous, exists := metrics[name]; exists {
				return fmt.Errorf("metric %q is declared by packages %q and %q", name, previous, owner)
			}
			metrics[name] = owner
		}
	}
	return nil
}

func normalizeVersion(version string) string {
	if strings.HasPrefix(version, "v") {
		return version
	}
	return "v" + version
}

func isRemoteLocation(location string) bool {
	parsed, err := url.Parse(location)
	return err == nil && (parsed.Scheme == "http" || parsed.Scheme == "https")
}

func validateRemoteLocation(location string, policy LoadPolicy) error {
	parsed, err := url.Parse(location)
	if err != nil {
		return fmt.Errorf("parse package location %q: %w", location, err)
	}
	if !policy.AllowRemote {
		return fmt.Errorf("remote package %q is disabled by security policy", location)
	}
	if parsed.Scheme != "https" && !policy.AllowHTTP {
		return fmt.Errorf("remote package %q must use HTTPS", location)
	}
	if len(policy.AllowedHosts) == 0 || !slices.Contains(policy.AllowedHosts, parsed.Hostname()) {
		return fmt.Errorf("remote package host %q is not allowed", parsed.Hostname())
	}
	return nil
}
