// Package db_test contains tests for the db package
package db_test

import (
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/jakthom/hercules/pkg/db"
	"github.com/jakthom/hercules/pkg/testutil"
	"github.com/stretchr/testify/assert"
)

func TestMacro(t *testing.T) {
	conn, mock, _ := testutil.GetMockedConnection()

	macroSQL := db.SQL("test() as (select 1)")

	macro := db.Macro{
		Name: "test",
		SQL:  macroSQL,
	}

	// Ensure creation/replacement sql
	assert.Equal(t, "create or replace macro "+string(macroSQL), string(macro.CreateOrReplaceSQL()))
	// Ensure query is executed appropriately
	mock.ExpectQuery(string(macro.CreateOrReplaceSQL())).WithoutArgs().WillReturnRows(sqlmock.NewRows([]string{}))
	db.TestHookEnsureMacro(conn, macro)
}

func TestEnsureMacrosWithConnection(_ *testing.T) {
	conn, mock, _ := testutil.GetMockedConnection()

	macros := []db.Macro{
		{
			Name: "test",
			SQL:  db.SQL("test() as (select 1)"),
		},
	}

	mock.ExpectQuery(string(macros[0].CreateOrReplaceSQL())).WithoutArgs().WillReturnRows(sqlmock.NewRows([]string{}))
	db.EnsureMacrosWithConnection(macros, conn)
}
