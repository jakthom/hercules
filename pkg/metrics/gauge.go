package metrics

import (
	"database/sql"

	"github.com/dbecorp/ducktheus_exporter/pkg/db"
	"github.com/prometheus/client_golang/prometheus"
)

type GaugeMetricDefinition struct {
	Name    string     `json:"name"`
	Enabled bool       `json:"enabled"`
	Type    MetricType `json:"type"`
	Help    string     `json:"help"`
	Sql     db.Sql     `json:"sql"`
	Labels  []string   `json:"labels"`
}

func (m *GaugeMetricDefinition) AsVec() *prometheus.GaugeVec {
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
