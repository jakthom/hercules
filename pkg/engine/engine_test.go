package engine_test

import (
	"testing"

	"github.com/jakthom/hercules/pkg/db"
	"github.com/jakthom/hercules/pkg/engine"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEngineOwnsDuckDBLifecycle(t *testing.T) {
	duckDB, err := engine.Open(t.Context(), ":memory:")
	require.NoError(t, err)
	require.NoError(t, duckDB.Exec(t.Context(), "create table example as select 42 as value"))
	table, err := duckDB.Query(t.Context(), "select value from example", 10)
	require.NoError(t, err)
	assert.Equal(t, []string{"value"}, table.Columns)
	require.Len(t, table.Rows, 1)
	value, err := db.ToFloat64(table.Rows[0][0])
	require.NoError(t, err)
	assert.Equal(t, 42.0, value)
	require.NoError(t, duckDB.Close())
	require.Error(t, duckDB.Exec(t.Context(), "select 1"))
}

func TestEngineEnforcesRowLimit(t *testing.T) {
	duckDB, err := engine.Open(t.Context(), ":memory:")
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, duckDB.Close()) })
	_, err = duckDB.Query(t.Context(), "select * from range(10)", 5)
	require.Error(t, err)
}

func TestEngineAppliesResourceOptions(t *testing.T) {
	duckDB, err := engine.OpenWithOptions(t.Context(), ":memory:", engine.Options{
		MemoryLimit: "64MB",
		Threads:     1,
	})
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, duckDB.Close()) })
	table, err := duckDB.Query(t.Context(), "select current_setting('threads') as value", 1)
	require.NoError(t, err)
	value, err := db.ToFloat64(table.Rows[0][0])
	require.NoError(t, err)
	assert.Equal(t, 1.0, value)
}
