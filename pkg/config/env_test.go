package config

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIsTraceMode(t *testing.T) {
	assert.False(t, IsTraceMode())
	os.Setenv(TRACE, "true")
	assert.True(t, IsTraceMode())
	os.Setenv(TRACE, "True")
	assert.True(t, IsTraceMode())
	os.Setenv(TRACE, "1")
	assert.True(t, IsTraceMode())
}

func TestIsDebugMode(t *testing.T) {
	assert.False(t, IsDebugMode())
	os.Setenv(DEBUG, "true")
	assert.True(t, IsDebugMode())
	os.Setenv(DEBUG, "True")
	assert.True(t, IsDebugMode())
	os.Setenv(DEBUG, "1")
	assert.True(t, IsDebugMode())
}
