package flock

import (
	"context"
	"database/sql"
	"database/sql/driver"

	"github.com/dbecorp/ducktheus/pkg/config"
	"github.com/dbecorp/ducktheus/pkg/db"
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
	ensureMacros(conf.Macros, conn)
	ensureExtensions(conf.Extensions, conn)
	defer db.Close()
	return db, conn
}

func ensureMacros(macros []db.Macro, conn *sql.Conn) {
	// Ensure built-in macros are present
	for _, macro := range ducktheusMacros() {
		macro.EnsureWithConnection(conn)
	}
	// Ensure configured macros are present
	for _, macro := range macros {
		macro.EnsureWithConnection(conn)
	}
}

func ensureExtensions(extensions db.Extensions, conn *sql.Conn) {
	for _, coreExtension := range extensions.Core {
		coreExtension.EnsureWithConnection(conn)
	}
	for _, communityExtension := range extensions.Community {
		communityExtension.EnsureWithConnection(conn)
	}
}

func ducktheusMacros() []db.Macro {
	// Stub global macros
	return []db.Macro{}
}
