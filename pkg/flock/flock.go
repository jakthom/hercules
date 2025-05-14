package flock

import (
	"context"
	"database/sql"

	"github.com/jakthom/hercules/pkg/config"
	"github.com/marcboeker/go-duckdb/v2"
	"github.com/rs/zerolog/log"
)

func InitializeDB(conf config.Config) (*sql.DB, *sql.Conn) {
	// Open a connection to DuckDB using the new API
	connector, err := duckdb.NewConnector(conf.DB, nil)
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
