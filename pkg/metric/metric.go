package metric

import (
	"database/sql"
	"strings"

	"github.com/jakthom/hercules/pkg/db"
)

type MetricDefinition struct {
	Name       string    `json:"name"`
	Enabled    bool      `json:"enabled"`
	Help       string    `json:"help"`
	Sql        db.Sql    `json:"sql"`
	Labels     []string  `json:"labels"`
	Buckets    []float64 `json:"buckets,omitempty"`    // If the metric is a histogram
	Objectives []float64 `json:"objectives,omitempty"` // If the metric is a summary
	// Internal
	Metadata Metadata `json:"metadata"`
}

func (m *MetricDefinition) LabelNames() []string {
	names := []string{}
	names = append(names, m.Labels...)
	for k := range m.Metadata.Labels {
		names = append(names, k)
	}
	return names
}

func (m *MetricDefinition) injectLabels(conn *sql.Conn) {
	labels := db.GetLabelNamesFromQuery(conn, m.Sql)
	m.Labels = labels
}

func (m *MetricDefinition) injectMetadata(metadata Metadata) {
	m.Metadata = metadata
}

func (m *MetricDefinition) FullName() string {
	prefix := string(m.Metadata.Prefix) + string(strings.ReplaceAll(string(m.Metadata.PackageName), "-", "_")) + "_"
	return prefix + m.Name
}

type MetricDefinitions struct {
	Gauge     []*MetricDefinition `json:"gauge"`
	Counter   []*MetricDefinition `json:"counter"`
	Summary   []*MetricDefinition `json:"summary"`
	Histogram []*MetricDefinition `json:"histogram"`
}

func (m *MetricDefinitions) InjectMetadata(conn *sql.Conn, metadata Metadata) {
	for _, metricDefinition := range m.Gauge {
		metricDefinition.injectLabels(conn)
		metricDefinition.injectMetadata(metadata)
	}
	for _, metricDefinition := range m.Counter {
		metricDefinition.injectLabels(conn)
		metricDefinition.injectMetadata(metadata)
	}
	for _, metricDefinition := range m.Summary {
		metricDefinition.injectLabels(conn)
		metricDefinition.injectMetadata(metadata)
	}
	for _, metricDefinition := range m.Histogram {
		metricDefinition.injectLabels(conn)
		metricDefinition.injectMetadata(metadata)
	}
}

func (m *MetricDefinitions) Merge(metricDefinitions MetricDefinitions) {
	m.Gauge = append(m.Gauge, metricDefinitions.Gauge...)
	m.Counter = append(m.Counter, metricDefinitions.Counter...)
	m.Summary = append(m.Summary, metricDefinitions.Summary...)
	m.Histogram = append(m.Histogram, metricDefinitions.Histogram...)
}

type Materializeable interface {
	Materialize(conn *sql.Conn) error
}
