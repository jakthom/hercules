package db

import (
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	testutil "github.com/dbecorp/hercules/pkg/testUtil"
	"github.com/stretchr/testify/assert"
)

func TestMacro(t *testing.T) {
	conn, mock, _ := testutil.GetMockedConnection()

	macroSql := Sql("test() as (select 1)")

	macro := Macro{
		Name: "test",
		Sql:  macroSql,
	}

	// Ensure creation/replacement sql
	assert.Equal(t, "create or replace macro "+macroSql, macro.CreateOrReplaceSql())
	// Ensure query is executed appropriately
	mock.ExpectQuery(string(macro.CreateOrReplaceSql())).WithoutArgs().WillReturnRows(sqlmock.NewRows([]string{}))
	macro.ensureWithConnection(conn)
}

func TestEnsureMacrosWithConnection(t *testing.T) {
	conn, mock, _ := testutil.GetMockedConnection()

	macros := []Macro{
		{
			Name: "test",
			Sql:  Sql("test() as (select 1)"),
		},
	}

	mock.ExpectQuery(string(macros[0].CreateOrReplaceSql())).WithoutArgs().WillReturnRows(sqlmock.NewRows([]string{}))
	EnsureMacrosWithConnection(macros, conn)
}
