package db

import (
	"database/sql"
)

type Macro struct {
	Name string `json:"name"`
	Sql  Sql    `json:"sql"`
}

func (m *Macro) CreateOrReplaceSql() Sql {
	// TODO -> allow "create macro" and "create or replace macro" to be included in the sql statement
	return Sql(m.Sql)
}

func (m *Macro) EnsureWithConnection(conn *sql.Conn) {
	RunSqlQuery(conn, m.CreateOrReplaceSql())
}
