// Package metric_test contains tests for the metric package
package metric_test

import (
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/jakthom/hercules/pkg/metric"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDefinition_LabelNames(t *testing.T) {
	tests := []struct {
		name     string
		metric   metric.Definition
		expected []string
	}{
		{
			name: "With no labels",
			metric: metric.Definition{
				Name:     "test_metric",
				Labels:   []string{},
				Metadata: metric.Metadata{Labels: map[string]string{}},
			},
			expected: []string{},
		},
		{
			name: "With metric labels only",
			metric: metric.Definition{
				Name:     "test_metric",
				Labels:   []string{"label1", "label2"},
				Metadata: metric.Metadata{Labels: map[string]string{}},
			},
			expected: []string{"label1", "label2"},
		},
		{
			name: "With metadata labels only",
			metric: metric.Definition{
				Name:   "test_metric",
				Labels: []string{},
				Metadata: metric.Metadata{Labels: map[string]string{
					"meta1": "value1",
					"meta2": "value2",
				}},
			},
			expected: []string{"meta1", "meta2"},
		},
		{
			name: "With both types of labels",
			metric: metric.Definition{
				Name:   "test_metric",
				Labels: []string{"label1", "label2"},
				Metadata: metric.Metadata{Labels: map[string]string{
					"meta1": "value1",
					"meta2": "value2",
				}},
			},
			expected: []string{"label1", "label2", "meta1", "meta2"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.metric.LabelNames()
			assert.ElementsMatch(t, tt.expected, result, "LabelNames() should return all labels")
		})
	}
}

func TestDefinition_FullName(t *testing.T) {
	tests := []struct {
		name     string
		metric   metric.Definition
		expected string
	}{
		{
			name: "No prefix",
			metric: metric.Definition{
				Name: "metric_name",
				Metadata: metric.Metadata{
					PackageName: "test",
					Prefix:      "",
				},
			},
			expected: "test_metric_name",
		},
		{
			name: "With prefix",
			metric: metric.Definition{
				Name: "metric_name",
				Metadata: metric.Metadata{
					PackageName: "test",
					Prefix:      "prefix_",
				},
			},
			expected: "prefix_test_metric_name",
		},
		{
			name: "Package name with hyphens",
			metric: metric.Definition{
				Name: "metric_name",
				Metadata: metric.Metadata{
					PackageName: "test-package",
					Prefix:      "prefix_",
				},
			},
			expected: "prefix_test_package_metric_name",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.metric.FullName()
			assert.Equal(t, tt.expected, result, "FullName() should return the correct full metric name")
		})
	}
}

func TestDefinition_InjectLabels(t *testing.T) {
	// Create a mock database and connection that uses query matching by regexp
	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	require.NoError(t, err)
	defer db.Close()

	conn, err := db.Conn(t.Context())
	require.NoError(t, err)
	defer conn.Close()

	// Use a pattern that matches any query containing json_serialize_sql
	mock.ExpectQuery("json_serialize_sql").
		WillReturnRows(sqlmock.NewRows([]string{"column"}).
			AddRow("id").
			AddRow("name").
			AddRow("value"))

	// Create a metric definition for testing
	metricDef := &metric.Definition{
		Name: "test_metric",
		SQL:  "SELECT * FROM test_table",
	}

	// Call the method being tested
	err = metricDef.InjectLabels(conn)
	require.NoError(t, err)

	// Verify "value" column isn't included in labels (special column name)
	assert.ElementsMatch(t, []string{"id", "name"}, metricDef.Labels)

	// Verify all expectations were met
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestDefinitions_Merge(t *testing.T) {
	base := metric.Definitions{
		Gauge: []*metric.Definition{
			{Name: "gauge1"},
		},
		Counter: []*metric.Definition{
			{Name: "counter1"},
		},
	}

	toMerge := metric.Definitions{
		Gauge: []*metric.Definition{
			{Name: "gauge2"},
		},
		Summary: []*metric.Definition{
			{Name: "summary1"},
		},
		Histogram: []*metric.Definition{
			{Name: "histogram1"},
		},
	}

	base.Merge(toMerge)

	assert.Len(t, base.Gauge, 2, "Should have 2 gauge metrics")
	assert.Len(t, base.Counter, 1, "Should have 1 counter metric")
	assert.Len(t, base.Summary, 1, "Should have 1 summary metric")
	assert.Len(t, base.Histogram, 1, "Should have 1 histogram metric")

	assert.Equal(t, "gauge1", base.Gauge[0].Name)
	assert.Equal(t, "gauge2", base.Gauge[1].Name)
}

func TestDefinitions_InjectMetadata(t *testing.T) {
	// Create a mock database and connection that uses query matching by regexp
	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	require.NoError(t, err)
	defer db.Close()

	conn, err := db.Conn(t.Context())
	require.NoError(t, err)
	defer conn.Close()

	// Set up expectations for SQL parsing queries (using simplified regex patterns)
	mock.ExpectQuery("json_serialize_sql\\('SELECT 1'\\)").
		WillReturnRows(sqlmock.NewRows([]string{"column"}).AddRow("val"))

	mock.ExpectQuery("json_serialize_sql\\('SELECT 2'\\)").
		WillReturnRows(sqlmock.NewRows([]string{"column"}).AddRow("val"))

	mock.ExpectQuery("json_serialize_sql\\('SELECT 3'\\)").
		WillReturnRows(sqlmock.NewRows([]string{"column"}).AddRow("val"))

	mock.ExpectQuery("json_serialize_sql\\('SELECT 4'\\)").
		WillReturnRows(sqlmock.NewRows([]string{"column"}).AddRow("val"))

	metrics := metric.Definitions{
		Gauge: []*metric.Definition{
			{Name: "gauge1", SQL: "SELECT 1"},
		},
		Counter: []*metric.Definition{
			{Name: "counter1", SQL: "SELECT 2"},
		},
		Summary: []*metric.Definition{
			{Name: "summary1", SQL: "SELECT 3"},
		},
		Histogram: []*metric.Definition{
			{Name: "histogram1", SQL: "SELECT 4"},
		},
	}

	metadata := metric.Metadata{
		PackageName: "test-package",
		Prefix:      "prefix_",
		Labels: map[string]string{
			"env": "test",
		},
	}

	// Update the call to handle error return
	err = metrics.InjectMetadata(conn, metadata)
	require.NoError(t, err)

	// Check that metadata was injected into all metrics
	assert.Equal(t, metadata, metrics.Gauge[0].Metadata)
	assert.Equal(t, metadata, metrics.Counter[0].Metadata)
	assert.Equal(t, metadata, metrics.Summary[0].Metadata)
	assert.Equal(t, metadata, metrics.Histogram[0].Metadata)

	// Verify all expectations were met
	assert.NoError(t, mock.ExpectationsWereMet())
}
