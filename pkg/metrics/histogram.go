package metrics

import (
	"database/sql"

	"github.com/dbecorp/ducktheus_exporter/pkg/db"
	"github.com/prometheus/client_golang/prometheus"
)

type HistogramMetricDefinition struct {
	Name    string     `json:"name"`
	Enabled bool       `json:"enabled"`
	Type    MetricType `json:"type"`
	Help    string     `json:"help"`
	Sql     db.Sql     `json:"sql"`
	Labels  []string   `json:"labels"`
	Buckets []float64
}

func (m *HistogramMetricDefinition) AsHistogramVec() *prometheus.HistogramVec {
	v := prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Name:    m.Name,
		Help:    m.Help,
		Buckets: m.Buckets,
	}, m.Labels)
	return v
}

func (m *HistogramMetricDefinition) MaterializeWithConnection(conn *sql.Conn) ([]QueryResult, error) {
	return materializeMetric(conn, m.Sql)
}
