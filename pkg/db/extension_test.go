package db

import (
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	testutil "github.com/dbecorp/hercules/pkg/testUtil"
)

func TestEnsureExtension(t *testing.T) {
	conn, mock, _ := testutil.GetMockedConnection()

	mock.ExpectQuery("install test;").WithoutArgs().WillReturnRows(sqlmock.NewRows([]string{}))
	mock.ExpectQuery("load test;").WithoutArgs().WillReturnRows(sqlmock.NewRows([]string{}))
	ensureExtension(conn, "test", CORE_EXTENSION)

	mock.ExpectQuery("install z from community;").WithoutArgs().WillReturnRows(sqlmock.NewRows([]string{}))
	mock.ExpectQuery("load z;").WithoutArgs().WillReturnRows(sqlmock.NewRows([]string{}))
	ensureExtension(conn, "z", COMMUNITY_EXTENSION)

}

func TestEnsureExtensionsWithConnection(t *testing.T) {
	conn, mock, _ := testutil.GetMockedConnection()

	extensions := Extensions{
		Core: []CoreExtension{
			CoreExtension{
				Name: "testcore",
			},
		},
		Community: []CommunityExtension{
			CommunityExtension{
				Name: "testcommunity",
			},
		},
	}

	mock.ExpectQuery("install testcore;").WithoutArgs().WillReturnRows(sqlmock.NewRows([]string{}))
	mock.ExpectQuery("load testcore;").WithoutArgs().WillReturnRows(sqlmock.NewRows([]string{}))
	mock.ExpectQuery("install testcommunity from community;").WithoutArgs().WillReturnRows(sqlmock.NewRows([]string{}))
	mock.ExpectQuery("load testcommunity;").WithoutArgs().WillReturnRows(sqlmock.NewRows([]string{}))
	EnsureExtensionsWithConnection(extensions, conn)
}
