package db

import (
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	testutil "github.com/jakthom/hercules/pkg/testUtil"
	"github.com/stretchr/testify/assert"
)

func TestRunSqlQuery(t *testing.T) {
	conn, mock, _ := testutil.GetMockedConnection()
	qr := Sql("test")

	mock.ExpectQuery(string(qr)).WithoutArgs().WillReturnRows(sqlmock.NewRows([]string{"col1"}).AddRow(1))
	rows, _ := RunSqlQuery(conn, qr)

	assert.Equal(t, true, rows.Next())
}
