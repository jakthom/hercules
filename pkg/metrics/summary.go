package metrics

import (
	"database/sql"

	"github.com/dbecorp/ducktheus_exporter/pkg/db"
	"github.com/prometheus/client_golang/prometheus"
)

type SummaryMetricDefinition struct {
	Name       string     `json:"name"`
	Enabled    bool       `json:"enabled"`
	Type       MetricType `json:"type"`
	Help       string     `json:"help"`
	Sql        db.Sql     `json:"sql"`
	Labels     []string   `json:"labels"`
	Objectives []float64
}

func (m *SummaryMetricDefinition) AsVec() *prometheus.SummaryVec {
	objectives := make(map[float64]float64)
	for _, o := range m.Objectives {
		objectives[o] = o
	}
	v := prometheus.NewSummaryVec(prometheus.SummaryOpts{
		Name:       m.Name,
		Help:       m.Help,
		Objectives: objectives,
	}, m.Labels)
	return v
}

func (m *SummaryMetricDefinition) MaterializeWithConnection(conn *sql.Conn) ([]QueryResult, error) {
	return materializeMetric(conn, m.Sql)
}
