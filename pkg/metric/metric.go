package metric

import (
	"database/sql"
	"strings"

	"github.com/jakthom/hercules/pkg/db"
)

// Definition defines a metric with its SQL query and metadata.
type Definition struct {
	Name       string    `json:"name"`
	Enabled    bool      `json:"enabled"`
	Help       string    `json:"help"`
	SQL        db.SQL    `json:"sql"`
	Labels     []string  `json:"labels"`
	Buckets    []float64 `json:"buckets,omitempty"`    // If the metric is a histogram.
	Objectives []float64 `json:"objectives,omitempty"` // If the metric is a summary.
	// Internal.
	Metadata Metadata `json:"metadata"`
}

func (m *Definition) LabelNames() []string {
	names := []string{}
	names = append(names, m.Labels...)
	for k := range m.Metadata.Labels {
		names = append(names, k)
	}
	return names
}

func (m *Definition) injectLabels(conn *sql.Conn) error {
	labels, err := db.GetLabelNamesFromQuery(conn, m.SQL)
	if err != nil {
		return err
	}
	m.Labels = labels
	return nil
}

// InjectLabels fetches the labels from the SQL query.
func (m *Definition) InjectLabels(conn *sql.Conn) error {
	return m.injectLabels(conn)
}

func (m *Definition) injectMetadata(metadata Metadata) {
	m.Metadata = metadata
}

func (m *Definition) FullName() string {
	prefix := string(m.Metadata.Prefix) + strings.ReplaceAll(m.Metadata.PackageName, "-", "_") + "_"
	return prefix + m.Name
}

// Definitions holds collections of different metric types.
type Definitions struct {
	Gauge     []*Definition `json:"gauge"`
	Counter   []*Definition `json:"counter"`
	Summary   []*Definition `json:"summary"`
	Histogram []*Definition `json:"histogram"`
}

func (m *Definitions) InjectMetadata(conn *sql.Conn, metadata Metadata) error {
	for _, metricDefinition := range m.Gauge {
		if err := metricDefinition.injectLabels(conn); err != nil {
			return err
		}
		metricDefinition.injectMetadata(metadata)
	}
	for _, metricDefinition := range m.Counter {
		if err := metricDefinition.injectLabels(conn); err != nil {
			return err
		}
		metricDefinition.injectMetadata(metadata)
	}
	for _, metricDefinition := range m.Summary {
		if err := metricDefinition.injectLabels(conn); err != nil {
			return err
		}
		metricDefinition.injectMetadata(metadata)
	}
	for _, metricDefinition := range m.Histogram {
		if err := metricDefinition.injectLabels(conn); err != nil {
			return err
		}
		metricDefinition.injectMetadata(metadata)
	}
	return nil
}

func (m *Definitions) Merge(definitions Definitions) {
	m.Gauge = append(m.Gauge, definitions.Gauge...)
	m.Counter = append(m.Counter, definitions.Counter...)
	m.Summary = append(m.Summary, definitions.Summary...)
	m.Histogram = append(m.Histogram, definitions.Histogram...)
}

type Materializeable interface {
	Materialize(conn *sql.Conn) error
}
