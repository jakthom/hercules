package metrics

import (
	"database/sql"

	"github.com/dbecorp/ducktheus_exporter/pkg/db"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/rs/zerolog/log"
)

type SourceType string
type MetricType string

const (
	// Metric Types
	CounterMetricType   MetricType = "counter"
	GaugeMetricType     MetricType = "gauge"
	HistogramMetricType MetricType = "histogram"
	SummaryMetricType   MetricType = "summary"
)

type Metric struct {
	Name    string     `json:"name"`
	Enabled bool       `json:"enabled"`
	Type    MetricType `json:"type"`
	Help    string     `json:"help"`
	Sql     db.Sql     `json:"sql"`
	Labels  []string   `json:"labels"`
}

func (m *Metric) AsGaugeVec() *prometheus.GaugeVec {
	v := prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: m.Name,
		Help: m.Help,
	}, m.Labels)
	return v
}

func (m *Metric) MaterializeWithConnection(conn *sql.Conn) ([]QueryResult, error) {
	rows, err := db.RunSqlQuery(conn, m.Sql)
	var results []QueryResult
	for rows.Next() {
		var result QueryResult
		if err := rows.Scan(&result.Labels, &result.Value); err != nil {
			log.Error().Err(err).Msg("error when scanning query results")
			return nil, err
		}
		results = append(results, result)
	}
	return results, err
}

type QueryResult struct {
	Value  float64
	Labels *map[string]interface{}
}

func (qr *QueryResult) StringifiedLabels() map[string]string {
	r := make(map[string]string)
	for k, v := range *qr.Labels {
		if v == nil {
			v = "null"
		}
		r[k] = v.(string)
	}
	return r
}

type GaugeMetricDefinition Metric
type GaugeQueryResult QueryResult

type SummaryMetricDefinition Metric
type SummaryQueryResult QueryResult

type CounterMetricDefinition Metric
type CounterQueryResult QueryResult

type HistogramMetricDefinition Metric
type HistogramQueryResult QueryResult
