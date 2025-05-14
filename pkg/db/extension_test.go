// Package db_test contains tests for the db package
package db_test

import (
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/jakthom/hercules/pkg/db"
	"github.com/jakthom/hercules/pkg/testutil"
)

func TestEnsureExtension(_ *testing.T) {
	conn, mock, _ := testutil.GetMockedConnection()

	mock.ExpectQuery("install test;").WithoutArgs().WillReturnRows(sqlmock.NewRows([]string{}))
	mock.ExpectQuery("load test;").WithoutArgs().WillReturnRows(sqlmock.NewRows([]string{}))
	db.TestHookEnsureExtension(conn, "test", db.CoreExtensionType)

	mock.ExpectQuery("install z from community;").WithoutArgs().WillReturnRows(sqlmock.NewRows([]string{}))
	mock.ExpectQuery("load z;").WithoutArgs().WillReturnRows(sqlmock.NewRows([]string{}))
	db.TestHookEnsureExtension(conn, "z", db.CommunityExtensionType)
}

func TestEnsureExtensionsWithConnection(_ *testing.T) {
	conn, mock, _ := testutil.GetMockedConnection()

	extensions := db.Extensions{
		Core: []db.CoreExtension{
			{
				Name: "testcore",
			},
		},
		Community: []db.CommunityExtension{
			{
				Name: "testcommunity",
			},
		},
	}

	mock.ExpectQuery("install testcore;").WithoutArgs().WillReturnRows(sqlmock.NewRows([]string{}))
	mock.ExpectQuery("load testcore;").WithoutArgs().WillReturnRows(sqlmock.NewRows([]string{}))
	mock.ExpectQuery("install testcommunity from community;").WithoutArgs().WillReturnRows(sqlmock.NewRows([]string{}))
	mock.ExpectQuery("load testcommunity;").WithoutArgs().WillReturnRows(sqlmock.NewRows([]string{}))
	db.EnsureExtensionsWithConnection(extensions, conn)
}
