package metric_test

import (
	"testing"

	"github.com/jakthom/hercules/pkg/labels"
	"github.com/jakthom/hercules/pkg/metric"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"sigs.k8s.io/yaml"
)

func TestEnabledDefaultsTrue(t *testing.T) {
	assert.True(t, (metric.Definition{}).IsEnabled())
	disabled := false
	assert.False(t, (metric.Definition{Enabled: &disabled}).IsEnabled())
}

func TestDefinitionValidation(t *testing.T) {
	definition := metric.Definition{
		Name: "latency_seconds",
		Help: "Latency.",
		SQL:  "select 1 as value",
		Metadata: metric.Metadata{
			PackageName: "test",
			Labels:      labels.Labels{"env": "test"},
		},
	}
	require.NoError(t, definition.Validate(metric.GaugeType))
	definition.Labels = []string{"env"}
	require.Error(t, definition.Validate(metric.GaugeType))
}

func TestLegacySummaryObjectivesDecode(t *testing.T) {
	var definition metric.Definition
	require.NoError(t, yaml.UnmarshalStrict([]byte(`
name: latency
help: Latency.
sql: select 1 as value
objectives: [0.5, 0.9, 0.99]
`), &definition))
	require.Len(t, definition.Objectives, 3)
	assert.Equal(t, metric.SummaryObjective{Quantile: 0.99, Error: 0.001}, definition.Objectives[2])
}

func TestFullNameAndMetadata(t *testing.T) {
	definitions := metric.Definitions{Gauge: []*metric.Definition{{Name: "value", Help: "Value.", SQL: "select 1"}}}
	metadata := metric.Metadata{PackageName: "test-package", Prefix: "prefix_"}
	definitions.InjectMetadata(metadata)
	assert.Equal(t, "prefix_test_package_value", definitions.Gauge[0].FullName())
}

func TestMetricConventionsAndOutputCollisions(t *testing.T) {
	metadata := metric.Metadata{PackageName: "test"}
	definitions := metric.Definitions{
		Counter: []*metric.Definition{{
			Name: "requests", Help: "Requests.", SQL: "select 1", Metadata: metadata,
		}},
	}
	require.Error(t, definitions.Validate())

	definitions = metric.Definitions{
		Gauge: []*metric.Definition{{
			Name: "latency_count", Help: "Conflicting count.", SQL: "select 1", Metadata: metadata,
		}},
		Histogram: []*metric.Definition{{
			Name: "latency", Help: "Latency.", SQL: "select 1", Buckets: []float64{1, 2}, Metadata: metadata,
		}},
	}
	err := definitions.Validate()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "test_latency_count")
}

func TestHistogramBucketsMustBeFiniteStrictAndBounded(t *testing.T) {
	definition := metric.Definition{
		Name: "latency", Help: "Latency.", SQL: "select 1", Buckets: []float64{1, 1},
	}
	require.Error(t, definition.Validate(metric.HistogramType))
}
