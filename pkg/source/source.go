package source

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/jakthom/hercules/pkg/db"
	"github.com/rs/zerolog/log"
)

// Type represents the type of data source.
type Type string

const (
	SQLSourceType     Type = "sql"
	ParquetSourceType Type = "parquet"
	JSONSourceType    Type = "json"
	CSVSourceType     Type = "csv"
)

type Source struct {
	Name                   string    `json:"name"`
	Type                   Type      `json:"type"`
	Source                 string    `json:"source"`
	Materialize            bool      `json:"materialize"` // Whether or not to materialize as a table.
	RefreshIntervalSeconds int       `json:"refreshIntervalSeconds"`
	stopChan               chan bool // Channel to stop the refresh goroutine.
}

// Cleanup stops the background refresh process if it's running.
func (s *Source) Cleanup() {
	if s.stopChan != nil {
		s.stopChan <- true
		close(s.stopChan)
		s.stopChan = nil
	}
}

// SQL returns the SQL representation of the source.
func (s *Source) SQL() db.SQL {
	switch s.Type {
	case ParquetSourceType:
		return db.SQL(fmt.Sprintf("select * from read_parquet('%s')", s.Source))
	case CSVSourceType:
		return db.SQL(fmt.Sprintf("select * from read_csv_auto('%s')", s.Source))
	case JSONSourceType:
		return db.SQL(fmt.Sprintf("select * from read_json_auto('%s')", s.Source))
	case SQLSourceType:
		return db.SQL(s.Source)
	default: // Default to sql
		return db.SQL(s.Source)
	}
}

func (s *Source) createOrReplaceTableSQL() db.SQL {
	return db.SQL("create or replace table " + s.Name + " as " + string(s.SQL()) + ";")
}

func (s *Source) createOrReplaceViewSQL() db.SQL {
	return db.SQL("create or replace view " + s.Name + " as " + string(s.SQL()) + ";")
}

func (s *Source) refreshWithConn(conn *sql.Conn) error {
	if s.Materialize {
		rows, err := db.RunSQLQuery(conn, s.createOrReplaceTableSQL())
		if err != nil {
			return err
		}
		defer rows.Close()

		if rowsErr := rows.Err(); rowsErr != nil {
			log.Error().Err(rowsErr).Interface("source", s.Name).Msg("error during table materialization")
			return rowsErr
		}

		log.Debug().Interface("source", s.Name).Msg("source refreshed")
		return nil
	}

	rows, err := db.RunSQLQuery(conn, s.createOrReplaceViewSQL())
	if err != nil {
		return err
	}
	defer rows.Close()

	if rowsErr := rows.Err(); rowsErr != nil {
		log.Error().Err(rowsErr).Interface("source", s.Name).Msg("error during view creation")
		return rowsErr
	}

	log.Debug().Interface("source", s.Name).Msg("source refreshed")
	return nil
}

func (s *Source) initializeWithConnection(conn *sql.Conn) error {
	err := s.refreshWithConn(conn)
	if err != nil {
		log.Fatal().Err(err).Interface("source", s.Name).Msg("could not refresh source")
	}
	// If the source is a table, materialize it on the predefined frequency. Views do not need to be refreshed.
	if s.Materialize && s.RefreshIntervalSeconds > 0 {
		// Start a ticker to continously update the source according to the predefined interval.
		ticker := time.NewTicker(time.Duration(s.RefreshIntervalSeconds) * time.Second)
		s.stopChan = make(chan bool)
		go func() {
			for {
				select {
				case <-s.stopChan:
					ticker.Stop()
					return
				case <-ticker.C:
					go func(conn *sql.Conn, source *Source) {
						refreshErr := source.refreshWithConn(conn)
						if refreshErr != nil {
							log.Debug().Interface("source", source.Name).Msg("could not refresh source")
						}
					}(conn, s)
				}
			}
		}()
	}
	return nil
}

// CleanupSources stops background refresh processes for all sources.
func CleanupSources(sources []Source) {
	for i := range sources {
		sources[i].Cleanup()
	}
}

func InitializeSourcesWithConnection(sources []Source, conn *sql.Conn) error {
	for i := range sources {
		err := sources[i].initializeWithConnection(conn)
		if err != nil {
			log.Error().Err(err).Interface("source", sources[i].Name).Msg("could not initialize source")
			return err
		}
	}
	return nil
}

// CreateOrReplaceTableSQL generates SQL to create or replace a table for this source.
func (s *Source) CreateOrReplaceTableSQL() db.SQL {
	return s.createOrReplaceTableSQL()
}

// CreateOrReplaceViewSQL generates SQL to create or replace a view for this source.
func (s *Source) CreateOrReplaceViewSQL() db.SQL {
	return s.createOrReplaceViewSQL()
}

// TestHookRefreshWithConn exposes the refreshWithConn method for testing.
func TestHookRefreshWithConn(s *Source, conn *sql.Conn) error {
	return s.refreshWithConn(conn)
}
