package flock

import (
	"context"
	"database/sql"
	"database/sql/driver"

	"github.com/jakthom/hercules/pkg/config"
	"github.com/marcboeker/go-duckdb"
	"github.com/rs/zerolog/log"
)

func InitializeDB(conf config.Config) (*sql.DB, *sql.Conn) {
	connector, err := duckdb.NewConnector(conf.Db, func(execer driver.ExecerContext) error {
		// Stub for boot queries
		return nil
	})
	if err != nil {
		log.Fatal().Err(err).Msg("could not initialize duckdb database")
	}
	db := sql.OpenDB(connector)
	conn, err := db.Conn(context.Background())
	if err != nil {
		log.Fatal().Err(err).Msg("could not initialize duckdb connection")
	}
	defer db.Close()
	return db, conn
}
