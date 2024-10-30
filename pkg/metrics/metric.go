package metrics

import (
	"database/sql"
	"strconv"
	"strings"

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
	rows, _ := db.RunSqlQuery(conn, md.Sql)
	var queryResults []QueryResult

	columns, _ := rows.Columns()
	// Explicitly lowercase all column names
	for i, col := range columns {
		columns[i] = strings.ToLower(col)
	}

	for rows.Next() {
		queryResult := QueryResult{}

		queryResult.Labels = make(map[string]interface{})
		results := make([]interface{}, len(columns))
		for i := range results {
			results[i] = new(sql.RawBytes)
		}
		if err := rows.Scan(results...); err != nil {
			log.Error().Err(err).Msg("could not scan row")
		}
		for i, v := range results {
			if sb, ok := v.(*sql.RawBytes); ok {
				if columns[i] == "value" {
					queryResult.Value, _ = strconv.ParseFloat(string(*sb), 64)
				} else {
					queryResult.Labels[columns[i]] = string(*sb)
				}
			}
			queryResults = append(queryResults, queryResult)
		}
	}
	return queryResults, nil
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
