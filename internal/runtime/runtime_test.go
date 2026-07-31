package runtime_test

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	herculesruntime "github.com/jakthom/hercules/internal/runtime"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildExposesReadyImmutableSnapshot(t *testing.T) {
	path := writeRuntimeConfig(t, "select 42 as value")
	runtime, err := herculesruntime.Build(t.Context(), herculesruntime.Options{ConfigPath: path, Version: "0.9.0"})
	require.NoError(t, err)
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		require.NoError(t, runtime.Close(ctx))
	})

	server := httptest.NewServer(runtime.Handler())
	defer server.Close()
	response, err := http.Get(server.URL + "/readyz")
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, response.StatusCode)
	require.NoError(t, response.Body.Close())

	const workers = 100
	errorsChannel := make(chan error, workers)
	results := make(chan string, workers)
	var wait sync.WaitGroup
	for range workers {
		wait.Add(1)
		go func() {
			defer wait.Done()
			response, err := http.Get(server.URL + "/metrics")
			if err != nil {
				errorsChannel <- err
				return
			}
			payload, err := io.ReadAll(response.Body)
			if err != nil {
				_ = response.Body.Close()
				errorsChannel <- err
				return
			}
			if err := response.Body.Close(); err != nil {
				errorsChannel <- err
				return
			}
			for _, line := range strings.Split(string(payload), "\n") {
				if strings.HasPrefix(line, "core_answer") {
					results <- line
					return
				}
			}
			errorsChannel <- assert.AnError
		}()
	}
	wait.Wait()
	close(errorsChannel)
	close(results)
	for err := range errorsChannel {
		require.NoError(t, err)
	}
	for result := range results {
		assert.Equal(t, `core_answer{hercules="test"} 42`, result)
	}
}

func TestBuildFailsWhenInitialSnapshotFails(t *testing.T) {
	path := writeRuntimeConfig(t, "select missing_column from missing_table")
	_, err := herculesruntime.Build(t.Context(), herculesruntime.Options{ConfigPath: path, Version: "0.9.0"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "initial metric snapshot")
}

func TestBuildResolvesInlineFileSourcesFromConfigurationDirectory(t *testing.T) {
	directory := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(directory, "values.csv"), []byte("value\n42\n"), 0o600))
	configPath := filepath.Join(directory, "hercules.yml")
	require.NoError(t, os.WriteFile(configPath, []byte(`
version: 1
name: test
sources:
  - name: values
    type: csv
    source: values.csv
    materialize: false
metrics:
  gauge:
    - name: answer
      help: The answer.
      sql: select value from values
`), 0o600))
	createdRuntime, err := herculesruntime.Build(t.Context(), herculesruntime.Options{
		ConfigPath: configPath,
		Version:    "0.9.0",
	})
	require.NoError(t, err)
	require.NoError(t, createdRuntime.Close(t.Context()))
}

func TestCloseRemovesOnlyExplicitEphemeralDatabaseAndDisablesReadiness(t *testing.T) {
	directory := t.TempDir()
	path := filepath.Join(directory, "hercules.yml")
	databasePath := filepath.Join(directory, "state.db")
	require.NoError(t, os.WriteFile(path, []byte(`
version: 1
name: test
db: state.db
ephemeralDB: true
metrics:
  gauge:
    - name: answer
      help: The answer.
      sql: select 42 as value
`), 0o600))
	createdRuntime, err := herculesruntime.Build(t.Context(), herculesruntime.Options{
		ConfigPath: path,
		Version:    "0.9.0",
	})
	require.NoError(t, err)
	require.FileExists(t, databasePath)
	require.NoError(t, createdRuntime.Close(t.Context()))
	assert.NoFileExists(t, databasePath)

	request := httptest.NewRequest(http.MethodGet, "/readyz", nil)
	recorder := httptest.NewRecorder()
	createdRuntime.Handler().ServeHTTP(recorder, request)
	assert.Equal(t, http.StatusServiceUnavailable, recorder.Code)
}

func TestMetricsBearerAuthentication(t *testing.T) {
	directory := t.TempDir()
	tokenPath := filepath.Join(directory, "metrics.token")
	require.NoError(t, os.WriteFile(tokenPath, []byte("secret-token\n"), 0o600))
	configPath := filepath.Join(directory, "hercules.yml")
	require.NoError(t, os.WriteFile(configPath, []byte(`
version: 1
name: test
http:
  bearerTokenFile: metrics.token
metrics:
  gauge:
    - name: answer
      help: The answer.
      sql: select 42 as value
`), 0o600))
	createdRuntime, err := herculesruntime.Build(t.Context(), herculesruntime.Options{
		ConfigPath: configPath,
		Version:    "0.9.0",
	})
	require.NoError(t, err)
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		require.NoError(t, createdRuntime.Close(ctx))
	})

	unauthorized := httptest.NewRecorder()
	createdRuntime.Handler().ServeHTTP(unauthorized, httptest.NewRequest(http.MethodGet, "/metrics", nil))
	assert.Equal(t, http.StatusUnauthorized, unauthorized.Code)

	authorizedRequest := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	authorizedRequest.Header.Set("Authorization", "Bearer secret-token")
	authorized := httptest.NewRecorder()
	createdRuntime.Handler().ServeHTTP(authorized, authorizedRequest)
	assert.Equal(t, http.StatusOK, authorized.Code)
	assert.Contains(t, authorized.Body.String(), "core_answer")

	health := httptest.NewRecorder()
	createdRuntime.Handler().ServeHTTP(health, httptest.NewRequest(http.MethodGet, "/healthz", nil))
	assert.Equal(t, http.StatusOK, health.Code)
}

func writeRuntimeConfig(t *testing.T, metricSQL string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "hercules.yml")
	content := `
version: 1
name: test
listenAddress: 127.0.0.1
port: 9999
db: ":memory:"
ephemeralDB: true
globalLabels: {}
metrics:
  gauge:
    - name: answer
      help: The answer.
      sql: ` + metricSQL + "\n"
	require.NoError(t, os.WriteFile(path, []byte(content), 0o600))
	return path
}
