// Package db_test contains tests for the db package
package db_test

import (
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/jakthom/hercules/pkg/db"
	"github.com/jakthom/hercules/pkg/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRunSqlQuery(t *testing.T) {
	conn, mock, _ := testutil.GetMockedConnection()
	qr := db.SQL("test")

	mock.ExpectQuery(string(qr)).WithoutArgs().WillReturnRows(sqlmock.NewRows([]string{"col1"}).AddRow(1))
	rows, err := db.RunSQLQuery(conn, qr)
	require.NoError(t, err)
	defer rows.Close()

	assert.True(t, rows.Next())

	// Check rows.Err
	assert.NoError(t, rows.Err())
}
