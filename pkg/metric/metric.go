package metric

import (
	"fmt"

	"github.com/rs/zerolog/log"
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
	Name            string
	Type            SourceType
	Source          string
	RefreshInterval int
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
	Enabled bool
	Type    MetricType
	Source  MetricSource
	Sql     Sql
}

func GetMetricSourceDefinitions() []MetricSource {
	sourceFile := "s3://okta-snowflake/query_history.parquet"
	source := MetricSource{
		Name:            "snowflake_query_history",
		Type:            ParquetFileSourceType,
		Source:          sourceFile,
		RefreshInterval: 20,
	}
	source2 := MetricSource{
		Name:            "snowflake_login_history",
		Type:            ParquetFileSourceType,
		Source:          "s3://okta-snowflake/query_history.parquet",
		RefreshInterval: 10,
	}
	return []MetricSource{
		source,
		source2,
	}
}

func GetMetricDefinitions() []Metric {
	log.Debug().Msg("getting metric definitions")
	// Return an example definition now
	source := GetMetricSourceDefinitions()[0]
	return []Metric{{
		Enabled: true,
		Type:    GaugeMetricType,
		Source:  source,
	}}

	// TODO - But really get from file

	// But expect to get remote definitions...someday
}
