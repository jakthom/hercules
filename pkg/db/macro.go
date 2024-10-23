package db

import (
	"database/sql"

	"github.com/rs/zerolog/log"
)

type Macro struct {
	Name string `json:"name"` // No-op. Probably overkill. Nice for future reasons.
	Sql  Sql    `json:"sql"`
}

func (m *Macro) CreateOrReplaceSql() Sql {
	// TODO -> be more flexible with how these are handled - allow "create", "create or replace", nameless macros, etc.
	return Sql(m.Sql)
}

func (m *Macro) ensureWithConnection(conn *sql.Conn) {
	_, err := RunSqlQuery(conn, m.CreateOrReplaceSql())
	if err != nil {

		log.Fatal().Err(err).Msg("could not ensure macro")
	}
	log.Debug().Interface("macro", m.Sql).Msg("macro ensured")
}

// Ensure all macros. Blow up if macro cannot be ensured
func EnsureMacrosWithConnection(macros []Macro, conn *sql.Conn) {
	for _, macro := range macros {
		macro.ensureWithConnection(conn)
	}
}
