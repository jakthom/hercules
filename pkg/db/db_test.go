package db_test

import (
	"context"
	"testing"

	"github.com/jakthom/hercules/pkg/db"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type recordingExecutor struct {
	queries []db.SQL
	err     error
}

func (r *recordingExecutor) Exec(_ context.Context, query db.SQL) error {
	r.queries = append(r.queries, query)
	return r.err
}

func (r *recordingExecutor) Query(context.Context, db.SQL, int) (db.Table, error) {
	return db.Table{}, nil
}

func TestEnsureExtensionsEnforcesPolicyAndOrder(t *testing.T) {
	executor := &recordingExecutor{}
	err := db.EnsureExtensions(t.Context(), executor, db.Extensions{
		Core: []db.CoreExtension{{Name: "json"}},
	}, db.ExtensionPolicy{Allowed: []string{"json"}})
	require.NoError(t, err)
	assert.Equal(t, []db.SQL{"install json;", "load json;"}, executor.queries)

	err = db.EnsureExtensions(t.Context(), executor, db.Extensions{
		Community: []db.CommunityExtension{{Name: "httpfs"}},
	}, db.ExtensionPolicy{Allowed: []string{"httpfs"}})
	require.Error(t, err)
}

func TestEnsureMacros(t *testing.T) {
	executor := &recordingExecutor{}
	err := db.EnsureMacros(t.Context(), executor, []db.Macro{{Name: "one", SQL: "one() as (select 1)"}})
	require.NoError(t, err)
	assert.Equal(t, []db.SQL{"create or replace macro one() as (select 1)"}, executor.queries)

	err = db.EnsureMacros(t.Context(), executor, []db.Macro{{Name: "one", SQL: "two() as (select 2)"}})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "must declare the same macro name")
}

func TestStrictValueConversion(t *testing.T) {
	value, err := db.ToFloat64(int64(42))
	require.NoError(t, err)
	assert.Equal(t, 42.0, value)
	_, err = db.ToFloat64(nil)
	require.Error(t, err)
	_, err = db.ToFloat64("not-a-number")
	require.Error(t, err)
	_, err = db.ToFloat64(int64(1<<53 + 1))
	require.Error(t, err)
	label, err := db.ToLabelValue(nil)
	require.NoError(t, err)
	assert.Equal(t, "NULL", label)
	label, err = db.ToLabelValue(uint16(42))
	require.NoError(t, err)
	assert.Equal(t, "42", label)
}
