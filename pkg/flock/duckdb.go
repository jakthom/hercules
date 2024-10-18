package flock

import (
	"context"
	"database/sql"

	"github.com/rs/zerolog/log"
)

func RunQuery(conn *sql.Conn, query string) (*sql.Rows, error) {
	log.Debug().Interface("query", query).Msg("running query")
	rows, err := conn.QueryContext(context.Background(), query)
	if err != nil {
		log.Error().Err(err).Interface("query", query).Msg("could not run query")
	}
	return rows, err
}
