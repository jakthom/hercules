package metrics

import (
	"fmt"
)

type Sql string
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

func (ms *MetricSource) Sql() Sql {
	switch ms.Type {
	case ParquetFileSourceType:
		return Sql(fmt.Sprintf("select * from read_parquet('%s')", ms.Source))
	case CsvFileSourceType:
		return Sql(fmt.Sprintf("select * from read_csv_auto('%s')", ms.Source))
	case HttpSourceType:
		return Sql(fmt.Sprintf("select * from '%s'", ms.Source))
	default: // Default to sql
		return Sql(ms.Source)
	}
}

func (ms *MetricSource) CreateOrReplaceSql() Sql {
	return Sql("create or replace table " + ms.Name + " as " + string(ms.Sql()) + ";")
}

type Metric struct {
	Name    string     `json:"name"`
	Enabled bool       `json:"enabled"`
	Type    MetricType `json:"type"`
	Help    string     `json:"help"`
	Sql     Sql        `json:"sql"`
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

type CounterQueryResult QueryResult
type GuageQueryResult QueryResult
type SummaryQueryResult QueryResult
