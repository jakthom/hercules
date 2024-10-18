package flock

import (
	"context"
	"database/sql"
	"database/sql/driver"

	"github.com/dbecorp/ducktheus_exporter/pkg/config"
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

func EnsureMacros(conf config.Config, conn *sql.DB) {

}
