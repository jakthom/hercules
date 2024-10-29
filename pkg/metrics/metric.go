package metrics

import (
	"database/sql"

	"github.com/jakthom/hercules/pkg/db"
	herculestypes "github.com/jakthom/hercules/pkg/types"
	"github.com/rs/zerolog/log"
)

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
	rows, err := db.RunSqlQuery(conn, md.Sql)
	if err != nil {
		return nil, err
	}
	var results []QueryResult
	for rows.Next() {
		result := QueryResult{}
		if md.Labels == nil { // Scalar results
			if err := rows.Scan(&result.Value); err != nil {
				log.Error().Err(err).Msg("error when scanning query results")
				return nil, err
			}
		} else {
			if err := rows.Scan(&result.Labels, &result.Value); err != nil {
				log.Error().Err(err).Msg("error when scanning query results")
				return nil, err
			}
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

func (m *MetricDefinitions) Merge(metricDefinitions MetricDefinitions) {
	m.Gauge = append(m.Gauge, metricDefinitions.Gauge...)
	m.Counter = append(m.Counter, metricDefinitions.Counter...)
	m.Summary = append(m.Summary, metricDefinitions.Summary...)
	m.Histogram = append(m.Histogram, metricDefinitions.Histogram...)
}

type MetricMetadata struct {
	PackageName herculestypes.PackageName
}
