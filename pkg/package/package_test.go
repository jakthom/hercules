package herculespackage_test

import (
	"crypto/sha256"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"

	herculespackage "github.com/jakthom/hercules/pkg/package"
	"github.com/jakthom/hercules/pkg/source"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const packageYAML = `
apiVersion: hercules.dev/v1alpha1
kind: Package
name: example
version: "1.0"
variables:
  table: default_table
sources:
  - name: data
    type: csv
    source: "{{table}}.csv"
    materialize: false
metrics:
  gauge:
    - name: value
      help: Example value.
      sql: select 1 as value
`

func TestLoadStrictPackageWithVariablesPathsAndDigest(t *testing.T) {
	directory := t.TempDir()
	path := filepath.Join(directory, "package.yml")
	require.NoError(t, os.WriteFile(path, []byte(packageYAML), 0o600))
	digest := fmt.Sprintf("sha256:%x", sha256.Sum256([]byte(packageYAML)))
	configuredPackage, err := (herculespackage.PackageConfig{
		Package:   path,
		Variables: herculespackage.Variables{"table": "events"},
		SHA256:    digest,
	}).Load(t.Context(), directory, nil, herculespackage.LoadPolicy{}, "0.9.0")
	require.NoError(t, err)
	assert.Equal(t, "example", configuredPackage.Name)
	assert.Equal(t, filepath.Join(directory, "events.csv"), configuredPackage.Sources[0].Source)
}

func TestLoadRejectsUnknownFieldsAndDigestMismatch(t *testing.T) {
	directory := t.TempDir()
	path := filepath.Join(directory, "package.yml")
	require.NoError(t, os.WriteFile(path, []byte(packageYAML+"typo: true\n"), 0o600))
	_, err := (herculespackage.PackageConfig{Package: path}).Load(t.Context(), directory, nil, herculespackage.LoadPolicy{}, "")
	require.Error(t, err)
	_, err = (herculespackage.PackageConfig{Package: path, SHA256: "deadbeef"}).Load(t.Context(), directory, nil, herculespackage.LoadPolicy{}, "")
	require.Error(t, err)
}

func TestLoadRequiresPackageAPIVersionAndKind(t *testing.T) {
	directory := t.TempDir()
	path := filepath.Join(directory, "package.yml")
	require.NoError(t, os.WriteFile(path, []byte("name: example\nversion: '1'\n"), 0o600))
	_, err := (herculespackage.PackageConfig{Package: path}).Load(
		t.Context(),
		directory,
		nil,
		herculespackage.LoadPolicy{},
		"",
	)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported apiVersion")
	assert.Contains(t, err.Error(), "unsupported kind")
}

func TestRemotePackagesAreOptInAndHostLimited(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, _ *http.Request) {
		_, _ = response.Write([]byte(packageYAML))
	}))
	defer server.Close()
	reference := herculespackage.PackageConfig{Package: server.URL}
	_, err := reference.Load(t.Context(), t.TempDir(), server.Client(), herculespackage.LoadPolicy{}, "")
	require.Error(t, err)
	_, err = reference.Load(t.Context(), t.TempDir(), server.Client(), herculespackage.LoadPolicy{
		AllowRemote:  true,
		AllowHTTP:    true,
		AllowedHosts: []string{"127.0.0.1"},
		MaxBytes:     1024 * 1024,
	}, "")
	require.NoError(t, err)

	_, err = reference.Load(t.Context(), t.TempDir(), server.Client(), herculespackage.LoadPolicy{
		AllowRemote:   true,
		AllowHTTP:     true,
		RequireSHA256: true,
		AllowedHosts:  []string{"127.0.0.1"},
		MaxBytes:      1024 * 1024,
	}, "")
	require.Error(t, err)
	digest := fmt.Sprintf("sha256:%x", sha256.Sum256([]byte(packageYAML)))
	reference.SHA256 = digest
	_, err = reference.Load(t.Context(), t.TempDir(), server.Client(), herculespackage.LoadPolicy{
		AllowRemote:   true,
		AllowHTTP:     true,
		RequireSHA256: true,
		AllowedHosts:  []string{"127.0.0.1"},
		MaxBytes:      1024 * 1024,
	}, "")
	require.NoError(t, err)
}

func TestCustomHTTPClientCannotBypassRedirectPolicy(t *testing.T) {
	var forbiddenRequests atomic.Int32
	forbidden := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, _ *http.Request) {
		forbiddenRequests.Add(1)
		_, _ = response.Write([]byte(packageYAML))
	}))
	defer forbidden.Close()

	allowed := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		http.Redirect(response, request, forbidden.URL, http.StatusFound)
	}))
	defer allowed.Close()
	allowedURL := strings.Replace(allowed.URL, "127.0.0.1", "localhost", 1)

	_, err := (herculespackage.PackageConfig{Package: allowedURL}).Load(
		t.Context(),
		t.TempDir(),
		allowed.Client(),
		herculespackage.LoadPolicy{
			AllowRemote:  true,
			AllowHTTP:    true,
			AllowedHosts: []string{"localhost"},
		},
		"",
	)
	require.Error(t, err)
	assert.Contains(t, err.Error(), `redirected package host "127.0.0.1" is not allowed`)
	assert.Zero(t, forbiddenRequests.Load())
}

func TestDigestPinnedRemotePackageSupportsOfflineCache(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, _ *http.Request) {
		_, _ = response.Write([]byte(packageYAML))
	}))
	digest := fmt.Sprintf("sha256:%x", sha256.Sum256([]byte(packageYAML)))
	cacheDirectory := t.TempDir()
	reference := herculespackage.PackageConfig{Package: server.URL, SHA256: digest}
	policy := herculespackage.LoadPolicy{
		AllowRemote:   true,
		AllowHTTP:     true,
		RequireSHA256: true,
		AllowedHosts:  []string{"127.0.0.1"},
		MaxBytes:      1024 * 1024,
		CacheDir:      cacheDirectory,
	}
	_, err := reference.Load(t.Context(), t.TempDir(), server.Client(), policy, "")
	require.NoError(t, err)
	server.Close()

	policy.Offline = true
	configuredPackage, err := reference.Load(t.Context(), t.TempDir(), server.Client(), policy, "")
	require.NoError(t, err)
	assert.Equal(t, "example", configuredPackage.Name)
}

func TestOrderDependenciesAndRejectCycles(t *testing.T) {
	packages := []herculespackage.Package{
		{APIVersion: herculespackage.CurrentAPIVersion, Kind: herculespackage.PackageKind, Name: "consumer", Version: "1", Dependencies: []string{"base"}},
		{APIVersion: herculespackage.CurrentAPIVersion, Kind: herculespackage.PackageKind, Name: "base", Version: "1"},
	}
	ordered, err := herculespackage.Order(packages)
	require.NoError(t, err)
	assert.Equal(t, "base", string(ordered[0].Name))
	packages[1].Dependencies = []string{"consumer"}
	_, err = herculespackage.Order(packages)
	require.Error(t, err)
}

func TestOrderRejectsCaseInsensitiveCatalogCollisions(t *testing.T) {
	packages := []herculespackage.Package{
		{
			APIVersion: herculespackage.CurrentAPIVersion,
			Kind:       herculespackage.PackageKind,
			Name:       "first",
			Version:    "1",
			Sources:    []source.Source{{Name: "Events"}},
		},
		{
			APIVersion: herculespackage.CurrentAPIVersion,
			Kind:       herculespackage.PackageKind,
			Name:       "second",
			Version:    "1",
			Sources:    []source.Source{{Name: "events"}},
		},
	}
	_, err := herculespackage.Order(packages)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "declared by packages")
}

func TestRepositoryExamplePackagesCompileTogether(t *testing.T) {
	repositoryRoot, err := filepath.Abs(filepath.Join("..", ".."))
	require.NoError(t, err)
	paths, err := filepath.Glob(filepath.Join(repositoryRoot, "hercules-packages", "*", "*.yml"))
	require.NoError(t, err)
	require.NotEmpty(t, paths)
	packages := make([]herculespackage.Package, 0, len(paths))
	for _, path := range paths {
		configuredPackage, loadErr := (herculespackage.PackageConfig{Package: path}).Load(
			t.Context(),
			repositoryRoot,
			nil,
			herculespackage.LoadPolicy{},
			"0.9.0",
		)
		require.NoError(t, loadErr, path)
		packages = append(packages, configuredPackage)
	}
	_, err = herculespackage.Order(packages)
	require.NoError(t, err)
}
