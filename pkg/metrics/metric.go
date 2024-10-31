package metrics

import (
	"database/sql"

	"github.com/jakthom/hercules/pkg/db"
)

type MetricDefinition struct {
	PackageName string    `json:"package_name"`
	Name        string    `json:"name"`
	Enabled     bool      `json:"enabled"`
	Help        string    `json:"help"`
	Sql         db.Sql    `json:"sql"`
	Labels      []string  `json:"labels"`
	Buckets     []float64 `json:"buckets"`    // If the metric is a histogram
	Objectives  []float64 `json:"objectives"` // If the metric is a summary
}

type MetricDefinitions struct {
	Gauge     []MetricDefinition `json:"gauge"`
	Counter   []MetricDefinition `json:"counter"`
	Summary   []MetricDefinition `json:"summary"`
	Histogram []MetricDefinition `json:"histogram"`
}

func (m *MetricDefinitions) Merge(metricDefinitions MetricDefinitions) {
	m.Gauge = append(m.Gauge, metricDefinitions.Gauge...)
	m.Counter = append(m.Counter, metricDefinitions.Counter...)
	m.Summary = append(m.Summary, metricDefinitions.Summary...)
	m.Histogram = append(m.Histogram, metricDefinitions.Histogram...)
}

type Materializeable interface {
	MaterializeWithConnection(conn *sql.Conn) error
}
