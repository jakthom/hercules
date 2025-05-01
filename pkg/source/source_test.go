// Package source_test contains tests for the source package
package source_test

import (
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/jakthom/hercules/pkg/source"
	"github.com/jakthom/hercules/pkg/testutil"
	"github.com/stretchr/testify/assert"
)

func TestSource_Sql(t *testing.T) {
	tests := []struct {
		name     string
		source   source.Source
		expected string
	}{
		{
			name: "SQL source type",
			source: source.Source{
				Name:   "test_source",
				Type:   source.SQLSourceType,
				Source: "SELECT * FROM test_table",
			},
			expected: "SELECT * FROM test_table",
		},
		{
			name: "Parquet source type",
			source: source.Source{
				Name:   "test_parquet",
				Type:   source.ParquetSourceType,
				Source: "/path/to/file.parquet",
			},
			expected: "select * from read_parquet('/path/to/file.parquet')",
		},
		{
			name: "JSON source type",
			source: source.Source{
				Name:   "test_json",
				Type:   source.JSONSourceType,
				Source: "/path/to/file.json",
			},
			expected: "select * from read_json_auto('/path/to/file.json')",
		},
		{
			name: "CSV source type",
			source: source.Source{
				Name:   "test_csv",
				Type:   source.CSVSourceType,
				Source: "/path/to/file.csv",
			},
			expected: "select * from read_csv_auto('/path/to/file.csv')",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.source.SQL()
			assert.Equal(t, tt.expected, string(result))
		})
	}
}

func TestSource_CreateOrReplaceTableSql(t *testing.T) {
	src := source.Source{
		Name:   "test_table",
		Type:   source.SQLSourceType,
		Source: "SELECT * FROM source_data",
	}

	expected := "create or replace table test_table as SELECT * FROM source_data;"
	result := src.CreateOrReplaceTableSQL()
	assert.Equal(t, expected, string(result))
}

func TestSource_CreateOrReplaceViewSql(t *testing.T) {
	src := source.Source{
		Name:   "test_view",
		Type:   source.SQLSourceType,
		Source: "SELECT * FROM source_data",
	}

	expected := "create or replace view test_view as SELECT * FROM source_data;"
	result := src.CreateOrReplaceViewSQL()
	assert.Equal(t, expected, string(result))
}

func TestSource_RefreshWithConn_Table(t *testing.T) {
	src := source.Source{
		Name:        "test_table",
		Type:        source.SQLSourceType,
		Source:      "SELECT * FROM source_data",
		Materialize: true,
	}

	conn, mock, _ := testutil.GetMockedConnection()

	// Setup mock expectations - using Query instead of Exec to match RunSqlQuery implementation
	mock.ExpectQuery("create or replace table test_table as SELECT * FROM source_data;").
		WillReturnRows(sqlmock.NewRows([]string{"result"}))

	// Call the method through the test hook
	err := source.TestHookRefreshWithConn(&src, conn)

	// Verify
	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestSource_RefreshWithConn_View(t *testing.T) {
	src := source.Source{
		Name:        "test_view",
		Type:        source.SQLSourceType,
		Source:      "SELECT * FROM source_data",
		Materialize: false,
	}

	conn, mock, _ := testutil.GetMockedConnection()

	// Setup mock expectations - using Query instead of Exec to match RunSqlQuery implementation
	mock.ExpectQuery("create or replace view test_view as SELECT * FROM source_data;").
		WillReturnRows(sqlmock.NewRows([]string{"result"}))

	// Call the method through the test hook
	err := source.TestHookRefreshWithConn(&src, conn)

	// Verify
	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestInitializeSourcesWithConnection(t *testing.T) {
	sources := []source.Source{
		{
			Name:                   "source1",
			Type:                   source.SQLSourceType,
			Source:                 "SELECT 1",
			Materialize:            true,
			RefreshIntervalSeconds: 60, // Set a positive refresh interval
		},
		{
			Name:        "source2",
			Type:        source.SQLSourceType,
			Source:      "SELECT 2",
			Materialize: false, // Views don't use the ticker
		},
	}

	conn, mock, _ := testutil.GetMockedConnection()

	// Setup mock expectations - using Query instead of Exec to match RunSqlQuery implementation
	mock.ExpectQuery("create or replace table source1 as SELECT 1;").
		WillReturnRows(sqlmock.NewRows([]string{"result"}))
	mock.ExpectQuery("create or replace view source2 as SELECT 2;").
		WillReturnRows(sqlmock.NewRows([]string{"result"}))

	// Call the function
	err := source.InitializeSourcesWithConnection(sources, conn)

	// Verify
	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}
