package metrics

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/dbecorp/ducktheus/pkg/db"
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

func (s *Source) Sql() db.Sql {
	switch s.Type {
	case ParquetFileSourceType:
		return db.Sql(fmt.Sprintf("select * from read_parquet('%s')", s.Source))
	case CsvFileSourceType:
		return db.Sql(fmt.Sprintf("select * from read_csv_auto('%s')", s.Source))
	case HttpSourceType:
		return db.Sql(fmt.Sprintf("select * from '%s'", s.Source))
	default: // Default to sql
		return db.Sql(s.Source)
	}
}

func (s *Source) createOrReplaceSql() db.Sql {
	return db.Sql("create or replace table " + s.Name + " as " + string(s.Sql()) + ";")
}

func (s *Source) refreshWithConn(conn *sql.Conn) error {
	_, err := db.RunSqlQuery(conn, s.createOrReplaceSql())
	log.Debug().Interface("source", s.Name).Msg("source refreshed")
	return err
}

func (s *Source) InitializeWithConnection(conn *sql.Conn) error {
	// Pre-populate the metric source
	s.refreshWithConn(conn)
	// Start a ticker to continously update the source according to the predefined interval
	ticker := time.NewTicker(time.Duration(s.RefreshIntervalSeconds) * time.Second)
	done := make(chan bool)
	go func() {
		for {
			select {
			case <-done:
				return
			case <-ticker.C:
				go func(conn *sql.Conn, source *Source) error {
					return source.refreshWithConn(conn)
				}(conn, s)
			}
		}
	}()
	return nil
}
