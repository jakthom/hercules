package main

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestVersionCommand(t *testing.T) {
	previous := version
	version = "1.2.3"
	t.Cleanup(func() { version = previous })
	var stdout bytes.Buffer
	assert.Equal(t, 0, run([]string{"version"}, &stdout, &bytes.Buffer{}))
	assert.Equal(t, "1.2.3\n", stdout.String())
}

func TestCheckConfigCommand(t *testing.T) {
	path := filepath.Join(t.TempDir(), "hercules.yml")
	require.NoError(t, os.WriteFile(path, []byte("version: 1\nname: test\n"), 0o600))
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	assert.Equal(t, 0, run([]string{"check-config", "--config", path}, &stdout, &stderr))
	assert.Contains(t, stdout.String(), "configuration and packages are valid")

	require.NoError(t, os.WriteFile(path, []byte("version: 1\ntyop: true\n"), 0o600))
	stdout.Reset()
	stderr.Reset()
	assert.Equal(t, 1, run([]string{"check-config", "--config", path}, &stdout, &stderr))
	assert.Contains(t, stderr.String(), "unknown field")
}

func TestPackageDigestAndUnknownCommand(t *testing.T) {
	path := filepath.Join(t.TempDir(), "package.yml")
	require.NoError(t, os.WriteFile(path, []byte("package"), 0o600))
	var stdout bytes.Buffer
	assert.Equal(t, 0, run([]string{"package-digest", path}, &stdout, &bytes.Buffer{}))
	assert.Equal(t, "sha256:bc4a71180870f7945155fbb02f4b0a2e3faa2a62d6d31b7039013055ed19869a\n", stdout.String())

	var stderr bytes.Buffer
	assert.Equal(t, 2, run([]string{"unknown"}, &bytes.Buffer{}, &stderr))
	assert.Contains(t, stderr.String(), "unknown command")
}

func TestPrintEffectiveConfigRedactsVariables(t *testing.T) {
	path := filepath.Join(t.TempDir(), "hercules.yml")
	require.NoError(t, os.WriteFile(path, []byte(`
version: 1
name: test
packages:
  - package: package.yml
    variables:
      token: secret
`), 0o600))
	var stdout bytes.Buffer
	assert.Equal(t, 0, run([]string{"print-effective-config", "--config", path}, &stdout, &bytes.Buffer{}))
	assert.NotContains(t, stdout.String(), "secret")
	assert.Contains(t, stdout.String(), "<redacted>")
}
