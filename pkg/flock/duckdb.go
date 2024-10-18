package flock

import (
	"context"
	"database/sql"

	"github.com/dbecorp/ducktheus_exporter/pkg/metrics"
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

func RefreshSource(conn *sql.Conn, source metrics.MetricSource) error {
	log.Debug().Interface("source", source.Name).Msg("refreshing source")
	_, err := RunQuery(conn, string(source.CreateOrReplaceSql()))
	log.Debug().Interface("source", source.Name).Msg("source refreshed")
	return err
}

func RunMetric(conn *sql.Conn, metric metrics.Metric) ([]metrics.QueryResult, error) {
	log.Debug().Interface("metric", metric.Name).Msg("getting values for metric")
	rows, err := RunQuery(conn, string(metric.Sql))
	var results []metrics.QueryResult
	for rows.Next() {
		var result metrics.QueryResult
		if err := rows.Scan(&result.Labels, &result.Value); err != nil {
			log.Error().Err(err).Msg("error when scanning query results")
			return nil, err
		}
		results = append(results, result)
	}
	return results, err
}
