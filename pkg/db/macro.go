package db

import (
	"database/sql"
)

type Macro struct {
	Name string `json:"name"` // Really a no-op. Probably overkill. Nice for future reasons.
	Sql  Sql    `json:"sql"`
}

func (m *Macro) CreateOrReplaceSql() Sql {
	// TODO -> be more flexible with how these are handled - allow "create", "create or replace", nameless macros, etc.
	return Sql(m.Sql)
}

func (m *Macro) EnsureWithConnection(conn *sql.Conn) {
	RunSqlQuery(conn, m.CreateOrReplaceSql())
}
