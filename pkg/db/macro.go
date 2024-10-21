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

func (m *Macro) EnsureWithConnection(conn *sql.Conn) {
	_, err := RunSqlQuery(conn, m.CreateOrReplaceSql())
	if err != nil {

		log.Fatal().Err(err).Msg("could not ensure macro")
	}
	log.Info().Interface("macro", m.Sql).Msg("macro ensured")
}
