package metrics

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/dbecorp/ducktheus_exporter/pkg/db"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/rs/zerolog/log"
)

type SourceType string
type MetricType string

const (
	// Source Types
	SqlSourceType         SourceType = "sql"
	ParquetFileSourceType SourceType = "parquet"
	CsvFileSourceType     SourceType = "csv"
	HttpSourceType        SourceType = "http"
	// Metric Types
	CounterMetricType   MetricType = "counter"
	GaugeMetricType     MetricType = "gauge"
	HistogramMetricType MetricType = "histogram"
	SummaryMetricType   MetricType = "summary"
)

type MetricSource struct {
	Name                   string     `json:"name"`
	Type                   SourceType `json:"type"`
	Source                 string     `json:"source"`
	RefreshIntervalSeconds int        `json:"refreshIntervalSeconds"`
}

func (ms *MetricSource) Sql() db.Sql {
	switch ms.Type {
	case ParquetFileSourceType:
		return db.Sql(fmt.Sprintf("select * from read_parquet('%s')", ms.Source))
	case CsvFileSourceType:
		return db.Sql(fmt.Sprintf("select * from read_csv_auto('%s')", ms.Source))
	case HttpSourceType:
		return db.Sql(fmt.Sprintf("select * from '%s'", ms.Source))
	default: // Default to sql
		return db.Sql(ms.Source)
	}
}

func (ms *MetricSource) CreateOrReplaceSql() db.Sql {
	return db.Sql("create or replace table " + ms.Name + " as " + string(ms.Sql()) + ";")
}

func (ms *MetricSource) RefreshWithConn(conn *sql.Conn) error {
	log.Debug().Interface("source", ms.Name).Msg("refreshing source")
	_, err := db.RunSqlQuery(conn, ms.CreateOrReplaceSql())
	log.Debug().Interface("source", ms.Name).Msg("source refreshed")
	return err
}

func (ms *MetricSource) InitializeWithConnection(conn *sql.Conn) error {
	// Pre-populate the metric source
	ms.RefreshWithConn(conn)
	// Start a ticker to continously update the source according to the predefined interval
	ticker := time.NewTicker(time.Duration(ms.RefreshIntervalSeconds) * time.Second)
	done := make(chan bool)
	go func() {
		for {
			select {
			case <-done:
				return
			case <-ticker.C:
				go func(conn *sql.Conn, source *MetricSource) error {
					return source.RefreshWithConn(conn)
				}(conn, ms)
			}
		}
	}()
	return nil
}

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
