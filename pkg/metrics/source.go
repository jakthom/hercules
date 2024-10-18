package metrics

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/dbecorp/ducktheus_exporter/pkg/db"
	"github.com/rs/zerolog/log"
)

const (
	// Source Types
	SqlSourceType         SourceType = "sql"
	ParquetFileSourceType SourceType = "parquet"
	CsvFileSourceType     SourceType = "csv"
	HttpSourceType        SourceType = "http"
)

type Source struct {
	Name                   string     `json:"name"`
	Type                   SourceType `json:"type"`
	Source                 string     `json:"source"`
	RefreshIntervalSeconds int        `json:"refreshIntervalSeconds"`
}

func (ms *Source) Sql() db.Sql {
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

func (ms *Source) CreateOrReplaceSql() db.Sql {
	return db.Sql("create or replace table " + ms.Name + " as " + string(ms.Sql()) + ";")
}

func (ms *Source) RefreshWithConn(conn *sql.Conn) error {
	log.Debug().Interface("source", ms.Name).Msg("refreshing source")
	_, err := db.RunSqlQuery(conn, ms.CreateOrReplaceSql())
	log.Debug().Interface("source", ms.Name).Msg("source refreshed")
	return err
}

func (ms *Source) InitializeWithConnection(conn *sql.Conn) error {
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
				go func(conn *sql.Conn, source *Source) error {
					return source.RefreshWithConn(conn)
				}(conn, ms)
			}
		}
	}()
	return nil
}
