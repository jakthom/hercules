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

type Metric struct {
	Name    string     `json:"name"`
	Enabled bool       `json:"enabled"`
	Type    MetricType `json:"type"`
	Help    string     `json:"help"`
	Sql     db.Sql     `json:"sql"`
	Labels  []string   `json:"labels"`
}

type QueryResult struct {
	Value  float64
	Labels *map[string]interface{}
}

func (qr *QueryResult) StringifiedLabels() map[string]string {
	r := make(map[string]string)
	for k, v := range *qr.Labels {
		if v == nil {
			v = "null"
		}
		r[k] = v.(string)
	}
	return r
}

type GaugeMetricDefinition Metric

func (m *GaugeMetricDefinition) AsGaugeVec() *prometheus.GaugeVec {
	v := prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: m.Name,
		Help: m.Help,
	}, m.Labels)
	return v
}

func (m *GaugeMetricDefinition) AsGauge() *prometheus.Gauge {
	v := prometheus.NewGauge(prometheus.GaugeOpts{
		Name: m.Name,
		Help: m.Help,
	})
	return &v
}

func (m *GaugeMetricDefinition) MaterializeWithConnection(conn *sql.Conn) ([]QueryResult, error) {
	return materializeMetric(conn, m.Sql)
}

type GaugeQueryResult QueryResult

type SummaryMetricDefinition Metric
type SummaryQueryResult QueryResult

type CounterMetricDefinition Metric
type CounterQueryResult QueryResult

type HistogramMetricDefinition Metric
type HistogramQueryResult QueryResult

type MetricDefinitions struct {
	Gauge     []GaugeMetricDefinition     `json:"gauge"`
	Counter   []CounterMetricDefinition   `json:"counter"`
	Summary   []SummaryMetricDefinition   `json:"summary"`
	Histogram []HistogramMetricDefinition `json:"histogram"`
}
