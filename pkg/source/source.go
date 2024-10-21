package source

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/dbecorp/hercules/pkg/db"
	"github.com/rs/zerolog/log"
)

type SourceType string

const (
	// Source Types
	SqlSourceType     SourceType = "sql"
	ParquetSourceType SourceType = "parquet"
	JsonSourceType    SourceType = "json"
	CsvSourceType     SourceType = "csv"
)

type Source struct {
	Name                   string     `json:"name"`
	Type                   SourceType `json:"type"`
	Source                 string     `json:"source"`
	Materialize            bool       `json:"materialize"` // Whether or not to materialize as a table.
	RefreshIntervalSeconds int        `json:"refreshIntervalSeconds"`
}

func (s *Source) Sql() db.Sql {
	switch s.Type {
	case ParquetSourceType:
		return db.Sql(fmt.Sprintf("select * from read_parquet('%s')", s.Source))
	case CsvSourceType:
		return db.Sql(fmt.Sprintf("select * from read_csv_auto('%s')", s.Source))
	case JsonSourceType:
		return db.Sql(fmt.Sprintf("select * from read_json_auto('%s')", s.Source))
	default: // Default to sql
		return db.Sql(s.Source)
	}
}

func (s *Source) createOrReplaceTableSql() db.Sql {
	return db.Sql("create or replace table " + s.Name + " as " + string(s.Sql()) + ";")
}

func (s *Source) createOrReplaceViewSql() db.Sql {
	return db.Sql("create or replace view " + s.Name + " as " + string(s.Sql()) + ";")
}

func (s *Source) refreshWithConn(conn *sql.Conn) error {
	if s.Materialize {
		_, err := db.RunSqlQuery(conn, s.createOrReplaceTableSql())
		log.Info().Interface("source", s.Name).Msg("source refreshed")
		return err
	} else {
		_, err := db.RunSqlQuery(conn, s.createOrReplaceViewSql())
		log.Info().Interface("source", s.Name).Msg("source refreshed")
		return err
	}
}

func (s *Source) InitializeWithConnection(conn *sql.Conn) error {
	err := s.refreshWithConn(conn)
	if err != nil {
		log.Fatal().Err(err).Interface("source", s.Name).Msg("could not refresh source")
	}
	// If the source is a table, materialize it on the predefined frequency. Views do not need to be refreshed.
	if s.Materialize {
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
	}
	return nil
}
