// Package config_test contains tests for the config package
package config_test

import (
	"bytes"
	"testing"

	"github.com/jakthom/hercules/pkg/config"
	herculespackage "github.com/jakthom/hercules/pkg/herculesPackage"
	"github.com/jakthom/hercules/pkg/labels"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/stretchr/testify/assert"
)

func buildTestConfig() config.Config {
	c := config.Config{
		Name:  "fake",
		Debug: false,
		Port:  "9999",
		DB:    "hercules.db",
		GlobalLabels: labels.Labels{
			"cell":    "ausw1",
			"fromEnv": "$FROMENV",
		},
		Packages: []herculespackage.PackageConfig{},
	}
	return c
}

func TestInstanceLabels(t *testing.T) {
	env := "testing"
	t.Setenv("FROMENV", env)
	conf := buildTestConfig()
	got := conf.InstanceLabels()
	want := labels.Labels{
		"cell":     "ausw1",
		"fromEnv":  env,
		"hercules": "fake",
	}
	assert.Equal(t, want, got)
}

func TestGetConfigNoFile(t *testing.T) {
	// Ensure a non-existent config path is used.
	t.Setenv(config.HerculesConfigPath, "non_existent_config.yml")

	// Temporarily capture logs to prevent error message in test output.
	var buf bytes.Buffer
	oldLogger := log.Logger
	log.Logger = zerolog.New(&buf).With().Timestamp().Logger()
	defer func() { log.Logger = oldLogger }()

	// Call GetConfig and capture the error.
	conf, err := config.GetConfig()

	// Verify an error was returned.
	assert.NotNil(t, err, "Expected an error when config file doesn't exist")
	assert.Contains(t, err.Error(), "config file not found")

	// Verify the error was logged.
	assert.Contains(t, buf.String(), "config file does not exist")

	// Verify default values are still set correctly.
	assert.Equal(t, config.DefaultDebug, conf.Debug)
	assert.Equal(t, config.DefaultPort, conf.Port)
	assert.Equal(t, config.DefaultDB, conf.DB)

	// Validate should not panic even with default config.
	conf.Validate()
}
