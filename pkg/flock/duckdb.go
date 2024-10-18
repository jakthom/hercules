package flock

import (
	"context"
	"database/sql"

	"github.com/dbecorp/ducktheus_exporter/pkg/metric"
	"github.com/rs/zerolog/log"
)

func RunQuery(conn *sql.Conn, query string) (*sql.Rows, error) {
	log.Trace().Interface("query", query).Msg("running query")
	rows, err := conn.QueryContext(context.Background(), query)
	if err != nil {
		log.Error().Err(err).Interface("query", query).Msg("could not run query")
	}
	return rows, err
}

func RefreshSource(conn *sql.Conn, source metric.MetricSource) error {
	log.Debug().Interface("source", source.Name).Msg("refreshing source")
	_, err := RunQuery(conn, string(source.CreateOrReplaceSql()))
	log.Debug().Interface("source", source.Name).Msg("source refreshed")
	return err
}
