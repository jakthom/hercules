package runtime

import (
	"context"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/jakthom/hercules/pkg/db"
	"github.com/jakthom/hercules/pkg/source"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type stubbornSourceExecutor struct {
	started chan struct{}
	release chan struct{}
	once    sync.Once
}

func (s *stubbornSourceExecutor) Exec(context.Context, db.SQL) error {
	s.once.Do(func() { close(s.started) })
	<-s.release
	return nil
}

func (s *stubbornSourceExecutor) Query(context.Context, db.SQL, int) (db.Table, error) {
	return db.Table{}, nil
}

func TestCloseHonorsDeadlineAndAllowsCleanupToFinish(t *testing.T) {
	createdRuntime := buildRuntimeForTest(t)
	createdRuntime.sourceManager.Close()

	executor := &stubbornSourceExecutor{started: make(chan struct{}), release: make(chan struct{})}
	manager := source.NewManager(context.Background(), executor, []source.Source{{
		Name:                   "events",
		Type:                   source.SQLSourceType,
		Source:                 "select 1",
		Materialize:            true,
		RefreshIntervalSeconds: 1,
	}}, time.Hour)
	createdRuntime.sourceManager = manager
	manager.Start()

	var releaseOnce sync.Once
	release := func() { releaseOnce.Do(func() { close(executor.release) }) }
	t.Cleanup(func() {
		release()
		cleanupContext, cancelCleanup := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancelCleanup()
		_ = createdRuntime.Close(cleanupContext)
	})

	select {
	case <-executor.started:
	case <-time.After(2 * time.Second):
		t.Fatal("source refresh did not start")
	}

	shutdownContext, cancelShutdown := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancelShutdown()
	err := createdRuntime.Close(shutdownContext)
	require.Error(t, err)
	assert.ErrorIs(t, err, context.DeadlineExceeded)

	release()
	cleanupContext, cancelCleanup := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancelCleanup()
	require.NoError(t, createdRuntime.Close(cleanupContext))
	assert.Error(t, createdRuntime.engine.Exec(t.Context(), "select 1"))
	assert.False(t, createdRuntime.ready.Load())
	err = createdRuntime.Run(t.Context())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "runtime is closed")
}

func TestRunServesAndShutsDown(t *testing.T) {
	createdRuntime := buildRuntimeForTest(t)
	reservedListener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	address := reservedListener.Addr().String()
	require.NoError(t, reservedListener.Close())
	createdRuntime.server.Addr = address

	runErrors := make(chan error, 1)
	go func() { runErrors <- createdRuntime.Run(t.Context()) }()
	client := &http.Client{Timeout: 100 * time.Millisecond}
	require.Eventually(t, func() bool {
		response, requestErr := client.Get("http://" + address + "/readyz")
		if requestErr != nil {
			return false
		}
		return response.Body.Close() == nil && response.StatusCode == http.StatusOK
	}, 2*time.Second, 10*time.Millisecond)

	shutdownContext, cancelShutdown := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancelShutdown()
	require.NoError(t, createdRuntime.Close(shutdownContext))
	select {
	case err = <-runErrors:
		require.NoError(t, err)
	case <-time.After(time.Second):
		t.Fatal("runtime did not stop after shutdown")
	}
}

func buildRuntimeForTest(t *testing.T) *Runtime {
	t.Helper()
	configPath := filepath.Join(t.TempDir(), "hercules.yml")
	require.NoError(t, os.WriteFile(configPath, []byte(`
version: 1
name: shutdown-test
metrics:
  gauge:
    - name: answer
      help: The answer.
      sql: select 42 as value
`), 0o600))
	createdRuntime, err := Build(t.Context(), Options{ConfigPath: configPath, Version: "0.9.0"})
	require.NoError(t, err)
	t.Cleanup(func() {
		cleanupContext, cancelCleanup := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancelCleanup()
		_ = createdRuntime.Close(cleanupContext)
	})
	return createdRuntime
}
