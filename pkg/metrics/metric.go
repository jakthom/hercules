package metrics

import (
	"database/sql"

	"github.com/dbecorp/ducktheus_exporter/pkg/db"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/rs/zerolog/log"
)

type SourceType string
type MetricType string

const (
	// Metric Types
	CounterMetricType   MetricType = "counter"
	GaugeMetricType     MetricType = "gauge"
	HistogramMetricType MetricType = "histogram"
	SummaryMetricType   MetricType = "summary"
)

type metricDefinition struct {
	Name    string     `json:"name"`
	Enabled bool       `json:"enabled"`
	Type    MetricType `json:"type"`
	Help    string     `json:"help"`
	Sql     db.Sql     `json:"sql"`
	Labels  []string   `json:"labels"`
}

func (md *metricDefinition) materializeWithConnection(conn *sql.Conn) ([]QueryResult, error) {
	return materializeMetric(conn, md.Sql)
}

func materializeMetric(conn *sql.Conn, sql db.Sql) ([]QueryResult, error) {
	rows, err := db.RunSqlQuery(conn, sql)
	var results []QueryResult
	for rows.Next() {
		var result QueryResult
		if err := rows.Scan(&result.Labels, &result.Value); err != nil {
			log.Error().Err(err).Msg("error when scanning query results")
			return nil, err
		}
		results = append(results, result)
	}
	return results, err
}

type MetricDefinitions struct {
	Gauge     []GaugeMetricDefinition     `json:"gauge"`
	Counter   []CounterMetricDefinition   `json:"counter"`
	Summary   []SummaryMetricDefinition   `json:"summary"`
	Histogram []HistogramMetricDefinition `json:"histogram"`
}

type MetricRegistry struct {
	Gauge     map[string]GaugeMetric
	Counter   map[string]CounterMetric
	Summary   map[string]SummaryMetric
	Histogram map[string]HistogramMetric
}

func (registry *MetricRegistry) AddGauge(d GaugeMetricDefinition) error {
	log.Trace().Interface("gauge", d.Name).Msg("adding gauge to registry")
	metric := GaugeMetric{
		Definition: d,
		Collector:  d.AsVec(),
	}
	prometheus.MustRegister(metric.Collector)
	registry.Gauge[d.Name] = metric
	return nil
}

func (registry *MetricRegistry) AddHistogram(d HistogramMetricDefinition) error {
	log.Trace().Interface("histogram", d.Name).Msg("adding histogram to registry")
	metric := HistogramMetric{
		Definition: d,
		Collector:  d.AsVec(),
	}
	prometheus.MustRegister(metric.Collector)
	registry.Histogram[d.Name] = metric
	return nil
}

func (registry *MetricRegistry) AddSummary(d SummaryMetricDefinition) error {
	log.Trace().Interface("summary", d.Name).Msg("adding summary to registry")
	metric := SummaryMetric{
		Definition: d,
		Collector:  d.AsVec(),
	}
	prometheus.MustRegister(metric.Collector)
	registry.Summary[d.Name] = metric
	return nil
}

func (mr *MetricRegistry) AddCounter(d CounterMetricDefinition) error {
	log.Trace().Interface("counter", d.Name).Msg("adding counter to registry")
	metric := CounterMetric{
		Definition: d,
		Collector:  d.AsVec(),
	}
	prometheus.MustRegister(metric.Collector)
	mr.Counter[d.Name] = metric
	return nil
}

func NewMetricRegistryFromMetricDefinitions(definitions MetricDefinitions) *MetricRegistry {
	r := MetricRegistry{}
	r.Gauge = make(map[string]GaugeMetric)
	r.Histogram = make(map[string]HistogramMetric)
	r.Summary = make(map[string]SummaryMetric)
	r.Counter = make(map[string]CounterMetric)

	for _, gauge := range definitions.Gauge {
		r.AddGauge(gauge)
	}
	for _, histogram := range definitions.Histogram {
		r.AddHistogram(histogram)
	}
	for _, summary := range definitions.Summary {
		r.AddSummary(summary)
	}
	for _, counter := range definitions.Counter {
		r.AddCounter(counter)
	}
	return &r
}
