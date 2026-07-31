package config_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jakthom/hercules/pkg/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadAppliesDefaultsAndLegacyLabels(t *testing.T) {
	t.Setenv("HERCULES_TEST_ENV", "test")
	path := writeConfig(t, `
version: 1
name: test
globalLabels:
  - env: $HERCULES_TEST_ENV
`)
	loaded, err := config.Load(path)
	require.NoError(t, err)
	assert.Equal(t, config.DefaultPort, loaded.Config.Port)
	assert.Equal(t, config.DefaultDB, loaded.Config.DB)
	labels, err := loaded.Config.InstanceLabels()
	require.NoError(t, err)
	assert.Equal(t, "test", labels["env"])
	assert.Equal(t, "test", labels[config.HerculesNameLabel])
}

func TestLoadRejectsUnknownField(t *testing.T) {
	path := writeConfig(t, "version: 1\nname: test\ntyop: true\n")
	_, err := config.Load(path)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unknown field")
}

func TestLoadRejectsInvalidValues(t *testing.T) {
	path := writeConfig(t, "version: 2\nname: ''\nport: 99999\n")
	_, err := config.Load(path)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported config version")
	assert.Contains(t, err.Error(), "name must not be empty")
	assert.Contains(t, err.Error(), "port must be between")
}

func TestLoadRequiresExplicitVersion(t *testing.T) {
	path := writeConfig(t, "name: test\n")
	_, err := config.Load(path)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "version is required")
}

func TestHTTPRequiresCertificateAndKeyTogether(t *testing.T) {
	path := writeConfig(t, "version: 1\nhttp:\n  tlsCertificateFile: server.crt\n")
	_, err := config.Load(path)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "must be configured together")
}

func TestDuckDBMemoryLimitMustBePositive(t *testing.T) {
	conf := config.Defaults()
	conf.DuckDB.MemoryLimit = "0.5GB"
	require.NoError(t, conf.Validate())

	for _, memoryLimit := range []string{"0B", "0.0GB", strings.Repeat("9", 400) + "GB"} {
		conf = config.Defaults()
		conf.DuckDB.MemoryLimit = memoryLimit
		err := conf.Validate()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "memoryLimit must be a positive size")
	}
}

func TestInstanceLabelsRequiresEnvironment(t *testing.T) {
	conf := config.Defaults()
	conf.GlobalLabels["env"] = "$HERCULES_MISSING_ENV"
	_, err := conf.InstanceLabels()
	require.Error(t, err)
}

func TestRepositorySchemasAreValidJSON(t *testing.T) {
	paths, err := filepath.Glob(filepath.Join("..", "..", "schemas", "*.json"))
	require.NoError(t, err)
	require.NotEmpty(t, paths)
	for _, path := range paths {
		payload, readErr := os.ReadFile(path)
		require.NoError(t, readErr)
		assert.True(t, json.Valid(payload), path)
	}
}

func writeConfig(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "hercules.yml")
	require.NoError(t, os.WriteFile(path, []byte(content), 0o600))
	return path
}
