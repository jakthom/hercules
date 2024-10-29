package config

import (
	"os"
	"testing"

	herculespackage "github.com/dbecorp/hercules/pkg/herculesPackage"
	"github.com/dbecorp/hercules/pkg/labels"
	"github.com/stretchr/testify/assert"
)

func buildTestConfig() Config {
	c := Config{
		Name:  "fake",
		Debug: false,
		Port:  "9999",
		Db:    "hercules.db",
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
	os.Setenv("FROMENV", env)
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
	conf, _ := GetConfig()
	conf.Validate() // passthrough
	assert.Equal(t, DEFAULT_DEBUG, conf.Debug)
	assert.Equal(t, DEFAULT_PORT, conf.Port)
	assert.Equal(t, DEFAULT_DB, conf.Db)
}
