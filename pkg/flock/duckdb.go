package flock

import (
	"context"
	"database/sql"
	"database/sql/driver"

	"github.com/dbecorp/ducktheus/pkg/config"
	"github.com/marcboeker/go-duckdb"
	"github.com/rs/zerolog/log"
)

func InitializeDB(conf config.Config) (*sql.DB, *sql.Conn) {
	connector, err := duckdb.NewConnector(conf.Db, func(execer driver.ExecerContext) error {
		// STUB OUT A SPOT FOR BOOT QUERIES
		// var err error
		// bootQueries := []string{}

		// for _, query := range bootQueries {
		// 	_, err = execer.ExecContext(context.Background(), query, nil)
		// 	if err != nil {
		// 		return err
		// 	}
		// }
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
	ensureMacros(conf, conn)
	ensureExtensions(conf, conn)
	defer db.Close()
	return db, conn
}

func ensureMacros(conf config.Config, conn *sql.Conn) {
	for _, macro := range conf.Macros {
		macro.EnsureWithConnection(conn)
	}
}

func ensureExtensions(conf config.Config, conn *sql.Conn) {
	for _, coreExtension := range conf.Extensions.Core {
		coreExtension.EnsureWithConnection(conn)
	}
	for _, communityExtension := range conf.Extensions.Community {
		communityExtension.EnsureWithConnection(conn)
	}
}
