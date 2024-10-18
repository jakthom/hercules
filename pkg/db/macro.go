package db

import "database/sql"

type Macro struct {
	Name string `json:"name"`
	Sql  Sql    `json:"sql"`
}

func (m *Macro) EnsureWithConnection(conn *sql.Conn) {

}
