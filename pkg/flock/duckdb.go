package flock

import (
	"context"
	"database/sql"
	"database/sql/driver"

	"github.com/dbecorp/ducktheus_exporter/pkg/config"
	"github.com/dbecorp/ducktheus_exporter/pkg/db"
	"github.com/dbecorp/ducktheus_exporter/pkg/metrics"
	"github.com/marcboeker/go-duckdb"
	"github.com/rs/zerolog/log"
)

func InitializeDB(conf config.Config) (*sql.DB, *sql.Conn) {
	connector, err := duckdb.NewConnector(conf.Db, func(execer driver.ExecerContext) error {
		var err error
		bootQueries := []string{}

		for _, query := range bootQueries {
			_, err = execer.ExecContext(context.Background(), query, nil)
			if err != nil {
				return err
			}
		}
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

func RunMetric(conn *sql.Conn, metric metrics.Metric) ([]metrics.QueryResult, error) {
	log.Debug().Interface("metric", metric.Name).Msg("getting values for metric")
	rows, err := db.RunSqlQuery(conn, metric.Sql)
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
