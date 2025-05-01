package db

import (
	"database/sql"

	"github.com/rs/zerolog/log"
)

type Macro struct {
	Name string `json:"name"` // No-op. Probably overkill. Nice for future reasons.
	SQL  SQL    `json:"sql"`
}

// CreateOrReplaceSQL returns the SQL statement to create or replace the macro.
func (m *Macro) CreateOrReplaceSQL() SQL {
	// TODO -> be more flexible with how these are handled - allow "create", "create or replace", nameless macros, etc.
	return SQL("create or replace macro " + string(m.SQL))
}

func (m *Macro) ensureWithConnection(conn *sql.Conn) {
	rows, err := RunSQLQuery(conn, m.CreateOrReplaceSQL())
	if err != nil {
		log.Error().Err(err).Msg("could not ensure macro")
		panic("Failed to ensure macro: " + err.Error())
	}
	defer func() {
		if closeErr := rows.Close(); closeErr != nil {
			log.Error().Err(closeErr).Msg("error closing rows after macro creation")
		}
	}()

	// Check for errors
	rowsErr := rows.Err()
	if rowsErr != nil {
		log.Error().Err(rowsErr).Msg("error during macro creation")
		panic("Error during macro creation: " + rowsErr.Error())
	}

	log.Debug().Interface("macro", m.SQL).Msg("macro ensured")
}

// TestHookEnsureMacro exposes the ensureWithConnection method for testing.
func TestHookEnsureMacro(conn *sql.Conn, macro Macro) {
	macro.ensureWithConnection(conn)
}

// EnsureMacrosWithConnection creates and ensures all macros are properly set up in the database.
func EnsureMacrosWithConnection(macros []Macro, conn *sql.Conn) {
	for _, macro := range macros {
		macro.ensureWithConnection(conn)
	}
}
