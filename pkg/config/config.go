package config

import (
	"errors"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/jakthom/hercules/pkg/db"
	"github.com/jakthom/hercules/pkg/labels"
	"github.com/jakthom/hercules/pkg/metric"
	herculespackage "github.com/jakthom/hercules/pkg/package"
	"github.com/jakthom/hercules/pkg/source"
	"sigs.k8s.io/yaml"
)

var (
	memoryLimitPattern   = regexp.MustCompile(`(?i)^(\d+(?:\.\d+)?)\s?(?:B|KB|MB|GB|TB|KiB|MiB|GiB|TiB)$`)
	extensionNamePattern = regexp.MustCompile(`^[a-zA-Z][a-zA-Z0-9_]*$`)
)

const (
	HerculesConfigPath        = "HERCULES_CONFIG_PATH"
	DefaultHerculesConfigPath = "hercules.yml"
	DebugEnvVar               = "DEBUG"
	TraceEnvVar               = "TRACE"
	HerculesNameLabel         = "hercules"
	CurrentConfigVersion      = 1
	DefaultPort               = 9999
	DefaultDB                 = ":memory:"
	DefaultCollectionInterval = 15 * time.Second
	DefaultCollectionTimeout  = 10 * time.Second
	DefaultMaxRowsPerMetric   = 100_000
	DefaultMaxSeries          = 100_000
	DefaultRemotePackageBytes = int64(2 << 20)
	DefaultDuckDBMemoryLimit  = "1GB"
	DefaultDuckDBThreads      = 4
	MaxCollectionInterval     = 7 * 24 * time.Hour
	MaxCollectionTimeout      = time.Hour
)

type DuckDB struct {
	MemoryLimit   string `json:"memoryLimit"`
	Threads       int    `json:"threads"`
	TempDirectory string `json:"tempDirectory,omitempty"`
}

type HTTP struct {
	TLSCertificateFile string `json:"tlsCertificateFile,omitempty"`
	TLSKeyFile         string `json:"tlsKeyFile,omitempty"`
	BearerTokenFile    string `json:"bearerTokenFile,omitempty"`
}

type Security struct {
	AllowRemotePackages        bool     `json:"allowRemotePackages"`
	RequireRemotePackageSHA256 bool     `json:"requireRemotePackageSHA256"`
	AllowedPackageHosts        []string `json:"allowedPackageHosts"`
	PackageCacheDirectory      string   `json:"packageCacheDirectory,omitempty"`
	Offline                    bool     `json:"offline"`
	AllowCommunityExtensions   bool     `json:"allowCommunityExtensions"`
	AllowedExtensions          []string `json:"allowedExtensions"`
	MaxRemotePackageBytes      int64    `json:"maxRemotePackageBytes"`
}

type Config struct {
	Version                   int                             `json:"version"`
	Name                      string                          `json:"name"`
	Debug                     bool                            `json:"debug"`
	ListenAddress             string                          `json:"listenAddress"`
	Port                      int                             `json:"port"`
	DB                        string                          `json:"db"`
	EphemeralDB               bool                            `json:"ephemeralDB"`
	DuckDB                    DuckDB                          `json:"duckdb"`
	HTTP                      HTTP                            `json:"http"`
	CollectionIntervalSeconds int                             `json:"collectionIntervalSeconds"`
	CollectionTimeoutSeconds  int                             `json:"collectionTimeoutSeconds"`
	MaxRowsPerMetric          int                             `json:"maxRowsPerMetric"`
	MaxSeries                 int                             `json:"maxSeries"`
	GlobalLabels              labels.Labels                   `json:"globalLabels"`
	Packages                  []herculespackage.PackageConfig `json:"packages"`
	Extensions                db.Extensions                   `json:"extensions"`
	Macros                    []db.Macro                      `json:"macros"`
	Sources                   []source.Source                 `json:"sources"`
	Metrics                   metric.Definitions              `json:"metrics"`
	Security                  Security                        `json:"security"`
}

type Loaded struct {
	Config  Config
	Path    string
	BaseDir string
}

func Defaults() Config {
	return Config{
		Version:       CurrentConfigVersion,
		Name:          "hercules",
		ListenAddress: "127.0.0.1",
		Port:          DefaultPort,
		DB:            DefaultDB,
		EphemeralDB:   false,
		DuckDB: DuckDB{
			MemoryLimit: DefaultDuckDBMemoryLimit,
			Threads:     DefaultDuckDBThreads,
		},
		CollectionIntervalSeconds: int(DefaultCollectionInterval / time.Second),
		CollectionTimeoutSeconds:  int(DefaultCollectionTimeout / time.Second),
		MaxRowsPerMetric:          DefaultMaxRowsPerMetric,
		MaxSeries:                 DefaultMaxSeries,
		GlobalLabels:              labels.Labels{},
		Security: Security{
			RequireRemotePackageSHA256: true,
			MaxRemotePackageBytes:      DefaultRemotePackageBytes,
		},
	}
}

func (c Config) Address() string {
	return net.JoinHostPort(c.ListenAddress, fmt.Sprintf("%d", c.Port))
}

func (c Config) CollectionInterval() time.Duration {
	return time.Duration(c.CollectionIntervalSeconds) * time.Second
}

func (c Config) CollectionTimeout() time.Duration {
	return time.Duration(c.CollectionTimeoutSeconds) * time.Second
}

func (c Config) InstanceLabels() (labels.Labels, error) {
	resolved, err := c.GlobalLabels.ResolveEnvironment()
	if err != nil {
		return nil, err
	}
	if _, exists := resolved[HerculesNameLabel]; exists {
		return nil, fmt.Errorf("global label %q is reserved", HerculesNameLabel)
	}
	resolved[HerculesNameLabel] = c.Name
	return resolved, nil
}

func (c Config) Validate() error {
	var errs []error
	if c.Version != CurrentConfigVersion {
		errs = append(errs, fmt.Errorf("unsupported config version %d; expected %d", c.Version, CurrentConfigVersion))
	}
	if strings.TrimSpace(c.Name) == "" {
		errs = append(errs, errors.New("name must not be empty"))
	}
	if c.Port < 1 || c.Port > 65535 {
		errs = append(errs, fmt.Errorf("port must be between 1 and 65535, got %d", c.Port))
	}
	if net.ParseIP(c.ListenAddress) == nil && c.ListenAddress != "localhost" {
		errs = append(errs, fmt.Errorf("listenAddress must be an IP address or localhost, got %q", c.ListenAddress))
	}
	if strings.TrimSpace(c.DB) == "" {
		errs = append(errs, errors.New("db must not be empty; use :memory: for an in-memory database"))
	}
	memoryLimit := memoryLimitPattern.FindStringSubmatch(c.DuckDB.MemoryLimit)
	validMemoryLimit := false
	if len(memoryLimit) == 2 {
		memoryValue, memoryErr := strconv.ParseFloat(memoryLimit[1], 64)
		validMemoryLimit = memoryErr == nil && memoryValue > 0
	}
	if !validMemoryLimit {
		errs = append(errs, fmt.Errorf(
			"duckdb.memoryLimit must be a positive size such as 512MB or 1GB, got %q",
			c.DuckDB.MemoryLimit,
		))
	}
	if c.DuckDB.Threads < 1 {
		errs = append(errs, errors.New("duckdb.threads must be positive"))
	}
	if (c.HTTP.TLSCertificateFile == "") != (c.HTTP.TLSKeyFile == "") {
		errs = append(errs, errors.New("http.tlsCertificateFile and http.tlsKeyFile must be configured together"))
	}
	if c.CollectionIntervalSeconds < 1 {
		errs = append(errs, errors.New("collectionIntervalSeconds must be positive"))
	} else if c.CollectionInterval() > MaxCollectionInterval {
		errs = append(errs, fmt.Errorf("collectionIntervalSeconds must not exceed %v", MaxCollectionInterval))
	}
	if c.CollectionTimeoutSeconds < 1 {
		errs = append(errs, errors.New("collectionTimeoutSeconds must be positive"))
	} else if c.CollectionTimeout() > MaxCollectionTimeout {
		errs = append(errs, fmt.Errorf("collectionTimeoutSeconds must not exceed %v", MaxCollectionTimeout))
	}
	if c.MaxRowsPerMetric < 1 {
		errs = append(errs, errors.New("maxRowsPerMetric must be positive"))
	}
	if c.MaxSeries < 1 {
		errs = append(errs, errors.New("maxSeries must be positive"))
	}
	if c.Security.MaxRemotePackageBytes < 1 {
		errs = append(errs, errors.New("security.maxRemotePackageBytes must be positive"))
	}
	for i, pkg := range c.Packages {
		if strings.TrimSpace(pkg.Package) == "" {
			errs = append(errs, fmt.Errorf("packages[%d].package must not be empty", i))
		}
	}
	if err := c.Security.validate(); err != nil {
		errs = append(errs, err)
	}
	if err := c.GlobalLabels.ValidateNames(); err != nil {
		errs = append(errs, err)
	}
	return errors.Join(errs...)
}

func (s Security) validate() error {
	var errs []error
	if s.AllowRemotePackages && len(s.AllowedPackageHosts) == 0 {
		errs = append(errs, errors.New("security.allowedPackageHosts must not be empty when remote packages are enabled"))
	}
	if s.Offline && strings.TrimSpace(s.PackageCacheDirectory) == "" {
		errs = append(errs, errors.New("security.packageCacheDirectory is required in offline mode"))
	}
	if s.Offline && !s.AllowRemotePackages {
		errs = append(errs, errors.New("security.allowRemotePackages must be true in offline mode"))
	}
	seenHosts := make(map[string]struct{}, len(s.AllowedPackageHosts))
	for _, host := range s.AllowedPackageHosts {
		if strings.TrimSpace(host) == "" || strings.ContainsAny(host, "/@") {
			errs = append(errs, fmt.Errorf("security.allowedPackageHosts contains invalid hostname %q", host))
		}
		if host != strings.ToLower(host) {
			errs = append(errs, fmt.Errorf("security.allowedPackageHosts hostname %q must be lowercase", host))
		}
		if _, exists := seenHosts[host]; exists {
			errs = append(errs, fmt.Errorf("security.allowedPackageHosts repeats %q", host))
		}
		seenHosts[host] = struct{}{}
	}
	seenExtensions := make(map[string]struct{}, len(s.AllowedExtensions))
	for _, extension := range s.AllowedExtensions {
		if !extensionNamePattern.MatchString(extension) {
			errs = append(errs, fmt.Errorf("security.allowedExtensions contains invalid name %q", extension))
		}
		if _, exists := seenExtensions[extension]; exists {
			errs = append(errs, fmt.Errorf("security.allowedExtensions repeats %q", extension))
		}
		seenExtensions[extension] = struct{}{}
	}
	return errors.Join(errs...)
}

func Load(path string) (Loaded, error) {
	if strings.TrimSpace(path) == "" {
		return Loaded{}, errors.New("configuration path must not be empty")
	}
	absPath, err := filepath.Abs(path)
	if err != nil {
		return Loaded{}, fmt.Errorf("resolve configuration path: %w", err)
	}
	payload, err := os.ReadFile(absPath)
	if err != nil {
		return Loaded{}, fmt.Errorf("read configuration %q: %w", absPath, err)
	}
	var header struct {
		Version *int `json:"version"`
	}
	if err := yaml.Unmarshal(payload, &header); err != nil {
		return Loaded{}, fmt.Errorf("decode configuration %q: %w", absPath, err)
	}
	if header.Version == nil {
		return Loaded{}, fmt.Errorf("decode configuration %q: version is required", absPath)
	}

	conf := Defaults()
	if err := yaml.UnmarshalStrict(payload, &conf); err != nil {
		return Loaded{}, fmt.Errorf("decode configuration %q: %w", absPath, err)
	}
	if err := conf.Validate(); err != nil {
		return Loaded{}, fmt.Errorf("validate configuration %q: %w", absPath, err)
	}
	return Loaded{Config: conf, Path: absPath, BaseDir: filepath.Dir(absPath)}, nil
}

func GetConfig() (Config, error) {
	path := os.Getenv(HerculesConfigPath)
	if path == "" {
		path = DefaultHerculesConfigPath
	}
	loaded, err := Load(path)
	if err != nil {
		return Config{}, err
	}
	return loaded.Config, nil
}
