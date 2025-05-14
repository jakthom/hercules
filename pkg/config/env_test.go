// Package config_test contains tests for config package
package config_test

import (
	"testing"

	"github.com/jakthom/hercules/pkg/config"
	"github.com/stretchr/testify/assert"
)

func TestIsTraceMode(t *testing.T) {
	assert.False(t, config.IsTraceMode())
	t.Setenv(config.TraceEnvVar, "true")
	assert.True(t, config.IsTraceMode())
	t.Setenv(config.TraceEnvVar, "True")
	assert.True(t, config.IsTraceMode())
	t.Setenv(config.TraceEnvVar, "1")
	assert.True(t, config.IsTraceMode())
}

func TestIsDebugMode(t *testing.T) {
	assert.False(t, config.IsDebugMode())
	t.Setenv(config.DebugEnvVar, "true")
	assert.True(t, config.IsDebugMode())
	t.Setenv(config.DebugEnvVar, "True")
	assert.True(t, config.IsDebugMode())
	t.Setenv(config.DebugEnvVar, "1")
	assert.True(t, config.IsDebugMode())
}
