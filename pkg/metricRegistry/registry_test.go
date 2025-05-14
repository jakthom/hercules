// Package registry_test contains tests for the registry package
package registry_test

import (
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/jakthom/hercules/pkg/metric"
	registry "github.com/jakthom/hercules/pkg/metricRegistry"
	"github.com/jakthom/hercules/pkg/testutil"
	"github.com/stretchr/testify/assert"
)

func TestNewMetricRegistry(t *testing.T) {
	// Create test metric definitions
	definitions := metric.Definitions{
		Gauge: []*metric.Definition{
			{
				Name: "test_gauge",
				Help: "Test gauge metric",
				SQL:  "SELECT 1",
				Metadata: metric.Metadata{
					PackageName: "test",
					Prefix:      "prefix_",
				},
			},
		},
		Counter: []*metric.Definition{
			{
				Name: "test_counter",
				Help: "Test counter metric",
				SQL:  "SELECT 1",
				Metadata: metric.Metadata{
					PackageName: "test",
					Prefix:      "prefix_",
				},
			},
		},
		Summary: []*metric.Definition{
			{
				Name:       "test_summary",
				Help:       "Test summary metric",
				SQL:        "SELECT 1",
				Objectives: []float64{0.5, 0.9, 0.99},
				Metadata: metric.Metadata{
					PackageName: "test",
					Prefix:      "prefix_",
				},
			},
		},
		Histogram: []*metric.Definition{
			{
				Name:    "test_histogram",
				Help:    "Test histogram metric",
				SQL:     "SELECT 1",
				Buckets: []float64{0.1, 0.5, 1.0, 5.0},
				Metadata: metric.Metadata{
					PackageName: "test",
					Prefix:      "prefix_",
				},
			},
		},
	}

	// Create a new registry
	reg := registry.NewMetricRegistry(definitions)

	// Check that all metrics were registered correctly
	assert.Len(t, reg.Gauge, 1, "Should have 1 gauge metric")
	assert.Len(t, reg.Counter, 1, "Should have 1 counter metric")
	assert.Len(t, reg.Summary, 1, "Should have 1 summary metric")
	assert.Len(t, reg.Histogram, 1, "Should have 1 histogram metric")

	// Check the metrics are stored with the correct keys (full names)
	_, exists := reg.Gauge["prefix_test_test_gauge"]
	assert.True(t, exists, "Gauge metric should be stored with its full name")
	_, exists = reg.Counter["prefix_test_test_counter"]
	assert.True(t, exists, "Counter metric should be stored with its full name")
	_, exists = reg.Summary["prefix_test_test_summary"]
	assert.True(t, exists, "Summary metric should be stored with its full name")
	_, exists = reg.Histogram["prefix_test_test_histogram"]
	assert.True(t, exists, "Histogram metric should be stored with its full name")
}

func TestMetricRegistry_Materialize(t *testing.T) {
	// Create test metric definitions
	definitions := metric.Definitions{
		Gauge: []*metric.Definition{
			{
				Name: "test_gauge",
				Help: "Test gauge metric",
				SQL:  "SELECT 1 AS value",
				Metadata: metric.Metadata{
					PackageName: "test",
				},
			},
		},
		Counter: []*metric.Definition{
			{
				Name: "test_counter",
				Help: "Test counter metric",
				SQL:  "SELECT 2 AS value",
				Metadata: metric.Metadata{
					PackageName: "test",
				},
			}},
	}

	// Create a new registry
	reg := registry.NewMetricRegistry(definitions)

	// Setup mock connection
	conn, mock, _ := testutil.GetMockedConnection()

	// We expect gauge and counter metrics to query the database
	mock.ExpectQuery("SELECT 1 AS value").
		WillReturnRows(sqlmock.NewRows([]string{"value"}).AddRow(1))
	mock.ExpectQuery("SELECT 2 AS value").
		WillReturnRows(sqlmock.NewRows([]string{"value"}).AddRow(2))

	// Materialize metrics
	err := reg.Materialize(conn)

	// Verify
	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}
