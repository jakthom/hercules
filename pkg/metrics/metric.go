package metrics

import (
	"database/sql"
	"strconv"

	"github.com/jakthom/hercules/pkg/db"
	"github.com/rs/zerolog/log"
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

func (md *MetricDefinition) materializeWithConnection(conn *sql.Conn) ([]QueryResult, error) {
	rows, _ := db.RunSqlQuery(conn, md.Sql)
	var queryResults []QueryResult
	columns, _ := rows.Columns()
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
				if columns[i] == "value" || columns[i] == "val" || columns[i] == "v" {
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
