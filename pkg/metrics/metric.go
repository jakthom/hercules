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
	Gauge     map[string]*prometheus.GaugeVec
	Counter   map[string]*prometheus.CounterVec
	Summary   map[string]*prometheus.SummaryVec
	Histogram map[string]*prometheus.HistogramVec
}

func (mr *MetricRegistry) AddGauge(d GaugeMetricDefinition) error {
	g := d.AsVec()
	log.Trace().Interface("gauge", d.Name).Msg("adding gauge to registry")
	prometheus.MustRegister(g)
	mr.Gauge[d.Name] = g
	return nil
}

func (mr *MetricRegistry) AddHistogram(d HistogramMetricDefinition) error {
	h := d.AsVec()
	log.Trace().Interface("histogram", d.Name).Msg("adding histogram to registry")
	prometheus.MustRegister(h)
	mr.Histogram[d.Name] = h
	return nil
}

func (mr *MetricRegistry) AddSummary(d SummaryMetricDefinition) error {
	s := d.AsVec()
	log.Trace().Interface("summary", d.Name).Msg("adding summary to registry")
	prometheus.MustRegister(s)
	mr.Summary[d.Name] = s
	return nil
}

func (mr *MetricRegistry) AddCounter(d CounterMetricDefinition) error {
	c := d.AsVec()
	log.Trace().Interface("counter", d.Name).Msg("adding counter to registry")
	prometheus.MustRegister(c)
	mr.Counter[d.Name] = c
	return nil
}

func NewMetricRegistryFromMetricDefinitions(definitions MetricDefinitions) *MetricRegistry {
	r := MetricRegistry{}
	r.Gauge = make(map[string]*prometheus.GaugeVec)
	r.Histogram = make(map[string]*prometheus.HistogramVec)
	r.Summary = make(map[string]*prometheus.SummaryVec)
	r.Counter = make(map[string]*prometheus.CounterVec)

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
