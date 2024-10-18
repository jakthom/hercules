package db

import (
	"context"
	"database/sql"

	"github.com/rs/zerolog/log"
)

func RunSqlQuery(conn *sql.Conn, query Sql) (*sql.Rows, error) {
	log.Trace().Interface("query", query).Msg("running query")
	rows, err := conn.QueryContext(context.Background(), string(query))
	if err != nil {
		log.Error().Err(err).Interface("query", query).Msg("could not run query")
	}
	return rows, err
}
