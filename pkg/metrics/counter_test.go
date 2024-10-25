package metrics

import (
	"testing"

	"github.com/dbecorp/hercules/pkg/labels"
	herculestypes "github.com/dbecorp/hercules/pkg/types"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Helper function to create a dummy CounterMetricDefinition
func newCounterMetricDefinition(name string, help string, labels []string) CounterMetricDefinition {
	return CounterMetricDefinition{
		metricDefinition: metricDefinition{
			Name:   name,
			Help:   help,
			Labels: labels,
		},
	}
}

// Helper function to create a dummy CounterMetric
func newCounterMetric(name string) CounterMetric {
	definition := newCounterMetricDefinition(name, "help", []string{"label1"})
	meta := herculestypes.MetricMetadata{
		MetricPrefix: "prefix_",
		PackageName:  "pkg",
		Labels:       labels.GlobalLabels{},
	}
	return NewCounterMetric(definition, meta)
}

// func TestCounterMetric_AsVec(t *testing.T) {
// 	definition := newCounterMetricDefinition("test_metric", "test help", []string{"label1"})
// 	meta := herculestypes.MetricMetadata{MetricPrefix: "prefix_", PackageName: "pkg", Labels: labels.GlobalLabels{}}
// 	metric := NewCounterMetric(definition, meta)

// 	collector := metric.AsVec()
// 	require.NotNil(t, collector)
// 	assert.Equal(t, "prefix_pkg_test_metric", collector.Describe() .Name())
// 	assert.Equal(t, "test help", collector.Help)
// }

func TestCounterMetric_Register(t *testing.T) {
	// Unregister any previously registered collectors to avoid duplicates
	metric := newCounterMetric("test_metric")
	prometheus.Unregister(metric.Collector) // Ensure that any previous instance is unregistered

	err := metric.register()
	require.NoError(t, err)

	// Check if the metric is successfully registered in Prometheus
	assert.NotNil(t, metric.Collector)
}

func TestCounterMetric_ReRegister(t *testing.T) {
	metric := newCounterMetric("test_metric")

	err := metric.reregister()
	require.NoError(t, err)

	// Check if the collector is re-registered
	assert.NotNil(t, prometheus.DefaultRegisterer)
	assert.NotNil(t, metric.Collector)
}

// func TestCounterMetric_MaterializeWithConnection(t *testing.T) {
// 	metric := newCounterMetric("test_metric")

// 	// Mock database connection
// 	conn := &sql.Conn{}

// 	// Mock materializeWithConnection to return sample data
// 	metric.Definition.materializeWithConnection = func(conn *sql.Conn) ([]materializedResult, error) {
// 		return []materializedResult{
// 			{StringifiedLabels: func() map[string]string {
// 				return map[string]string{"label1": "value1"}
// 			}},
// 		}, nil
// 	}

// 	err := metric.MaterializeWithConnection(conn)
// 	require.NoError(t, err)

// 	// Check if the metric was correctly incremented
// 	count := testutil.ToFloat64(metric.Collector.WithLabelValues("value1"))
// 	assert.Equal(t, float64(1), count)
// }

// func TestCounterMetric_MaterializeWithConnection_Error(t *testing.T) {
// 	metric := newCounterMetric("test_metric")

// 	// Mock database connection
// 	conn := &sql.Conn{}

// 	// Mock materializeWithConnection to return an error
// 	metric.Definition.materializeWithConnection = func(conn *sql.Conn) ([]materializedResult, error) {
// 		return nil, errors.New("materialization error")
// 	}

// 	err := metric.MaterializeWithConnection(conn)
// 	require.Error(t, err)
// 	assert.EqualError(t, err, "materialization error")
// }

// func TestNewCounterMetric(t *testing.T) {
// 	definition := newCounterMetricDefinition("test_metric", "test help", []string{"label1"})
// 	meta := herculestypes.MetricMetadata{
// 		MetricPrefix: "prefix_",
// 		PackageName:  "pkg",
// 		Labels:       labels.GlobalLabels{},
// 	}
// 	metric := NewCounterMetric(definition, meta)

// 	require.NotNil(t, metric.Collector)

// 	// Extract the metric description and check for the expected name and help text
// 	desc := metric.Collector.WithLabelValues().Desc().String()

// 	assert.Contains(t, desc, "prefix_pkg_test_metric") // Check if the name is correct
// 	assert.Contains(t, desc, "test help")              // Check if the help text is correct
// }

func TestNewCounterMetric(t *testing.T) {
	definition := newCounterMetricDefinition("test_metric", "test help", []string{"label1"}) // 1 label expected
	meta := herculestypes.MetricMetadata{
		MetricPrefix: "prefix_",
		PackageName:  "pkg",
		Labels:       labels.GlobalLabels{},
	}
	metric := NewCounterMetric(definition, meta)

	require.NotNil(t, metric.Collector)

	// Test the collector with correct number of label values
	metric.Collector.WithLabelValues("value1").Inc() // You must provide a value for "label1"

	// Verify the metric increment
	count := testutil.ToFloat64(metric.Collector.WithLabelValues("value1"))
	assert.Equal(t, float64(1), count)
}
