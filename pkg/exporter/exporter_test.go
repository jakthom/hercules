package exporter_test

import (
	"context"
	"errors"
	"math"
	"sync"
	"testing"

	"github.com/jakthom/hercules/pkg/db"
	"github.com/jakthom/hercules/pkg/exporter"
	"github.com/jakthom/hercules/pkg/labels"
	"github.com/jakthom/hercules/pkg/metric"
	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type queryExecutor struct {
	mu     sync.RWMutex
	tables map[db.SQL]db.Table
	err    error
}

func (q *queryExecutor) Exec(context.Context, db.SQL) error { return nil }

func (q *queryExecutor) Query(_ context.Context, query db.SQL, _ int) (db.Table, error) {
	q.mu.RLock()
	defer q.mu.RUnlock()
	if q.err != nil {
		return db.Table{}, q.err
	}
	return q.tables[query], nil
}

func TestSnapshotMetricSemantics(t *testing.T) {
	executor := &queryExecutor{tables: map[db.SQL]db.Table{
		"gauge":   {Columns: []string{"region", "value"}, Rows: [][]any{{"east", 4.0}}},
		"counter": {Columns: []string{"region", "value"}, Rows: [][]any{{"east", 2.0}, {"east", 3.0}}},
		"hist":    {Columns: []string{"value"}, Rows: [][]any{{1.0}, {2.0}}},
		"summary": {Columns: []string{"value"}, Rows: [][]any{{1.0}, {2.0}, {3.0}}},
	}}
	metadata := metric.Metadata{PackageName: "test", Labels: labels.Labels{"env": "test"}}
	definitions := metric.Definitions{
		Gauge:     []*metric.Definition{{Name: "gauge", Help: "Gauge.", SQL: "gauge", Labels: []string{"region"}, Metadata: metadata}},
		Counter:   []*metric.Definition{{Name: "counter_total", Help: "Counter.", SQL: "counter", Labels: []string{"region"}, Metadata: metadata}},
		Histogram: []*metric.Definition{{Name: "hist", Help: "Histogram.", SQL: "hist", Buckets: []float64{1, 2}, Metadata: metadata}},
		Summary:   []*metric.Definition{{Name: "summary", Help: "Summary.", SQL: "summary", Objectives: []metric.SummaryObjective{{Quantile: 0.5, Error: 0.05}}, Metadata: metadata}},
	}
	collector, err := exporter.New(executor, definitions.All(), 100, 100)
	require.NoError(t, err)
	require.NoError(t, collector.Refresh(t.Context()))
	families := gather(t, collector)

	assert.Equal(t, 4.0, families["test_gauge"].Metric[0].GetGauge().GetValue())
	assert.Equal(t, 5.0, families["test_counter_total"].Metric[0].GetCounter().GetValue())
	histogram := families["test_hist"].Metric[0].GetHistogram()
	assert.Equal(t, uint64(2), histogram.GetSampleCount())
	assert.Equal(t, 3.0, histogram.GetSampleSum())
	summary := families["test_summary"].Metric[0].GetSummary()
	assert.Equal(t, uint64(3), summary.GetSampleCount())
	assert.Equal(t, 2.0, summary.Quantile[0].GetValue())
}

func TestFailedRefreshRetainsLastKnownGoodSnapshot(t *testing.T) {
	executor := &queryExecutor{tables: map[db.SQL]db.Table{
		"gauge": {Columns: []string{"value"}, Rows: [][]any{{7.0}}},
	}}
	definitions := metric.Definitions{Gauge: []*metric.Definition{{
		Name: "gauge", Help: "Gauge.", SQL: "gauge", Metadata: metric.Metadata{PackageName: "test"},
	}}}
	collector, err := exporter.New(executor, definitions.All(), 100, 100)
	require.NoError(t, err)
	require.NoError(t, collector.Refresh(t.Context()))
	executor.mu.Lock()
	executor.err = errors.New("database unavailable")
	executor.mu.Unlock()
	require.Error(t, collector.Refresh(t.Context()))
	families := gather(t, collector)
	assert.Equal(t, 7.0, families["test_gauge"].Metric[0].GetGauge().GetValue())
	assert.Equal(t, 1.0, families["hercules_collection_errors_total"].Metric[0].GetCounter().GetValue())
}

func TestMalformedExecutorTableFailsWithoutReplacingSnapshot(t *testing.T) {
	executor := &queryExecutor{tables: map[db.SQL]db.Table{
		"gauge": {Columns: []string{"value"}, Rows: [][]any{{7.0}}},
	}}
	definitions := metric.Definitions{Gauge: []*metric.Definition{{
		Name: "gauge", Help: "Gauge.", SQL: "gauge", Metadata: metric.Metadata{PackageName: "test"},
	}}}
	collector, err := exporter.New(executor, definitions.All(), 100, 100)
	require.NoError(t, err)
	require.NoError(t, collector.Refresh(t.Context()))

	executor.mu.Lock()
	executor.tables["gauge"] = db.Table{Columns: []string{"value"}, Rows: [][]any{{}}}
	executor.mu.Unlock()
	err = collector.Refresh(t.Context())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "row 0 has 0 values for 1 columns")
	assert.Equal(t, 7.0, gather(t, collector)["test_gauge"].Metric[0].GetGauge().GetValue())
}

func TestCollectorOwnsImmutableDefinitionCopy(t *testing.T) {
	enabled := true
	definition := &metric.Definition{
		Name:    "gauge",
		Enabled: &enabled,
		Help:    "Gauge.",
		SQL:     "gauge",
		Metadata: metric.Metadata{
			PackageName: "test",
			Labels:      labels.Labels{"env": "production"},
		},
	}
	executor := &queryExecutor{tables: map[db.SQL]db.Table{
		"gauge": {Columns: []string{"value"}, Rows: [][]any{{7.0}}},
	}}
	collector, err := exporter.New(
		executor,
		[]metric.TypedDefinition{{Type: metric.GaugeType, Definition: definition}},
		100,
		100,
	)
	require.NoError(t, err)

	enabled = false
	definition.Name = "mutated"
	definition.SQL = "missing"
	definition.Metadata.PackageName = "mutated"
	definition.Metadata.Labels["env"] = "mutated"
	require.NoError(t, collector.Refresh(t.Context()))

	family := gather(t, collector)["test_gauge"]
	require.NotNil(t, family)
	assert.Equal(t, "production", family.GetMetric()[0].GetLabel()[0].GetValue())
}

func TestAggregationRejectsNonFiniteSums(t *testing.T) {
	executor := &queryExecutor{tables: map[db.SQL]db.Table{
		"counter": {Columns: []string{"value"}, Rows: [][]any{{math.MaxFloat64}, {math.MaxFloat64}}},
	}}
	definitions := metric.Definitions{Counter: []*metric.Definition{{
		Name: "counter_total", Help: "Counter.", SQL: "counter", Metadata: metric.Metadata{PackageName: "test"},
	}}}
	collector, err := exporter.New(executor, definitions.All(), 100, 100)
	require.NoError(t, err)
	err = collector.Refresh(t.Context())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "counter sum must be finite")
}

func TestSummaryInterpolationDoesNotOverflow(t *testing.T) {
	executor := &queryExecutor{tables: map[db.SQL]db.Table{
		"summary": {Columns: []string{"value"}, Rows: [][]any{{-math.MaxFloat64}, {math.MaxFloat64}}},
	}}
	definitions := metric.Definitions{Summary: []*metric.Definition{{
		Name:       "summary",
		Help:       "Summary.",
		SQL:        "summary",
		Objectives: []metric.SummaryObjective{{Quantile: 0.5, Error: 0.05}},
		Metadata:   metric.Metadata{PackageName: "test"},
	}}}
	collector, err := exporter.New(executor, definitions.All(), 100, 100)
	require.NoError(t, err)
	require.NoError(t, collector.Refresh(t.Context()))
	assert.Zero(t, gather(t, collector)["test_summary"].GetMetric()[0].GetSummary().GetQuantile()[0].GetValue())
}

func TestConcurrentGatherIsDeterministicForSQLMetrics(t *testing.T) {
	executor := &queryExecutor{tables: map[db.SQL]db.Table{
		"gauge": {Columns: []string{"label", "value"}, Rows: [][]any{{"a", 1.0}, {"b", 2.0}}},
	}}
	definitions := metric.Definitions{Gauge: []*metric.Definition{{
		Name: "gauge", Help: "Gauge.", SQL: "gauge", Labels: []string{"label"}, Metadata: metric.Metadata{PackageName: "test"},
	}}}
	collector, err := exporter.New(executor, definitions.All(), 100, 100)
	require.NoError(t, err)
	require.NoError(t, collector.Refresh(t.Context()))
	registry := prometheus.NewRegistry()
	require.NoError(t, registry.Register(collector))

	const workers = 100
	errorsChannel := make(chan error, workers)
	var wait sync.WaitGroup
	for range workers {
		wait.Add(1)
		go func() {
			defer wait.Done()
			families, err := registry.Gather()
			if err != nil {
				errorsChannel <- err
				return
			}
			for _, family := range families {
				if family.GetName() == "test_gauge" {
					if len(family.Metric) != 2 || family.Metric[0].GetGauge() == nil {
						errorsChannel <- errors.New("inconsistent test_gauge family")
					}
					return
				}
			}
			errorsChannel <- errors.New("test_gauge family missing")
		}()
	}
	wait.Wait()
	close(errorsChannel)
	for err := range errorsChannel {
		require.NoError(t, err)
	}
}

func TestHistogramSeriesLimitIncludesImplicitInfinityBucket(t *testing.T) {
	executor := &queryExecutor{tables: map[db.SQL]db.Table{
		"hist": {Columns: []string{"value"}, Rows: [][]any{{1.0}}},
	}}
	definitions := metric.Definitions{Histogram: []*metric.Definition{{
		Name: "hist", Help: "Histogram.", SQL: "hist", Buckets: []float64{1, 2},
		Metadata: metric.Metadata{PackageName: "test"},
	}}}
	collector, err := exporter.New(executor, definitions.All(), 100, 4)
	require.NoError(t, err)
	err = collector.Refresh(t.Context())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "maximum of 4 series")
}

func gather(t *testing.T, collector prometheus.Collector) map[string]*dto.MetricFamily {
	t.Helper()
	registry := prometheus.NewRegistry()
	require.NoError(t, registry.Register(collector))
	families, err := registry.Gather()
	require.NoError(t, err)
	result := make(map[string]*dto.MetricFamily, len(families))
	for _, family := range families {
		result[family.GetName()] = family
	}
	return result
}
