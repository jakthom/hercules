package metrics

import (
	"database/sql"

	"github.com/dbecorp/ducktheus_exporter/pkg/db"
	"github.com/prometheus/client_golang/prometheus"
)

type CounterMetricDefinition struct {
	Name    string     `json:"name"`
	Enabled bool       `json:"enabled"`
	Type    MetricType `json:"type"`
	Help    string     `json:"help"`
	Sql     db.Sql     `json:"sql"`
	Labels  []string   `json:"labels"`
}

func (m *CounterMetricDefinition) AsCounterVec() *prometheus.CounterVec {
	v := prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: m.Name,
		Help: m.Help,
	}, m.Labels)
	return v
}

func (m *CounterMetricDefinition) MaterializeWithConnection(conn *sql.Conn) ([]QueryResult, error) {
	return materializeMetric(conn, m.Sql)
}
