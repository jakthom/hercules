package source_test

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/jakthom/hercules/pkg/db"
	"github.com/jakthom/hercules/pkg/source"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type sourceExecutor struct {
	mu      sync.Mutex
	queries []db.SQL
}

type blockingSourceExecutor struct {
	started chan struct{}
	once    sync.Once
	active  atomic.Int32
	maximum atomic.Int32
}

func (b *blockingSourceExecutor) Exec(ctx context.Context, _ db.SQL) error {
	active := b.active.Add(1)
	defer b.active.Add(-1)
	for {
		maximum := b.maximum.Load()
		if active <= maximum || b.maximum.CompareAndSwap(maximum, active) {
			break
		}
	}
	b.once.Do(func() { close(b.started) })
	<-ctx.Done()
	return ctx.Err()
}

func (b *blockingSourceExecutor) Query(context.Context, db.SQL, int) (db.Table, error) {
	return db.Table{}, nil
}

func (s *sourceExecutor) Exec(_ context.Context, query db.SQL) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.queries = append(s.queries, query)
	return nil
}

func (s *sourceExecutor) Query(context.Context, db.SQL, int) (db.Table, error) {
	return db.Table{}, nil
}

func TestSourceSQLAndValidation(t *testing.T) {
	configured := source.Source{Name: "events", Type: source.ParquetSourceType, Source: "it's.parquet", Materialize: true}
	require.NoError(t, configured.Validate())
	assert.Equal(t, "select * from read_parquet('it''s.parquet')", string(configured.SQL()))
	assert.Equal(t, `create or replace table "events" as select * from read_parquet('it''s.parquet');`, string(configured.CreateOrReplaceSQL()))

	configured.Type = "typo"
	require.Error(t, configured.Validate())
}

func TestInitializeIsOrderedAndRejectsDuplicates(t *testing.T) {
	executor := &sourceExecutor{}
	sources := []source.Source{
		{Name: "one", Type: source.SQLSourceType, Source: "select 1", Materialize: true},
		{Name: "two", Type: source.SQLSourceType, Source: "select 2", Materialize: false},
	}
	require.NoError(t, source.Initialize(t.Context(), executor, sources))
	assert.Equal(t, []db.SQL{
		`create or replace table "one" as select 1;`,
		`create or replace view "two" as select 2;`,
	}, executor.queries)
	require.Error(t, source.Initialize(t.Context(), executor, append(sources, sources[0])))
	require.Error(t, source.Initialize(t.Context(), executor, []source.Source{
		{Name: "Events", Type: source.SQLSourceType, Source: "select 1"},
		{Name: "events", Type: source.SQLSourceType, Source: "select 2"},
	}))
}

func TestManagerPreventsOverlapAndJoinsCanceledRefresh(t *testing.T) {
	executor := &blockingSourceExecutor{started: make(chan struct{})}
	manager := source.NewManager(t.Context(), executor, []source.Source{{
		Name:                   "events",
		Type:                   source.SQLSourceType,
		Source:                 "select 1",
		Materialize:            true,
		RefreshIntervalSeconds: 1,
	}}, time.Hour)
	manager.Start()
	manager.Start()
	select {
	case <-executor.started:
	case <-time.After(2 * time.Second):
		t.Fatal("source refresh did not start")
	}
	closed := make(chan struct{})
	go func() {
		manager.Close()
		close(closed)
	}()
	select {
	case <-closed:
	case <-time.After(time.Second):
		t.Fatal("source manager did not join canceled refresh")
	}
	manager.Close()
	assert.Equal(t, int32(1), executor.maximum.Load())
	assert.Equal(t, uint64(1), manager.Statuses()["events"].Failures)
}
