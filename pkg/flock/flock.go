package flock

import (
	"context"
	"database/sql"
	"database/sql/driver"

	"github.com/dbecorp/hercules/pkg/config"
	"github.com/dbecorp/hercules/pkg/db"
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

func EnsureMacrosWithConnection(macros []db.Macro, conn *sql.Conn) {
	// Ensure built-in macros are present
	for _, macro := range herculesMacros() {
		macro.EnsureWithConnection(conn)
	}
	// Ensure configured macros are present
	for _, macro := range macros {
		macro.EnsureWithConnection(conn)
	}
}

func EnsureExtensionsWithConnection(extensions db.Extensions, conn *sql.Conn) {
	for _, coreExtension := range extensions.Core {
		coreExtension.EnsureWithConnection(conn)
	}
	for _, communityExtension := range extensions.Community {
		communityExtension.EnsureWithConnection(conn)
	}
}

func herculesMacros() []db.Macro {
	// Stub global macros
	return []db.Macro{}
}
