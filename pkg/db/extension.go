package db

import (
	"database/sql"

	"github.com/rs/zerolog/log"
)

const (
	COMMUNITY_EXTENSION string = "community"
	CORE_EXTENSION      string = "core"
)

func ensureExtension(conn *sql.Conn, extensionName string, extensionType string) {
	var installSql Sql
	var loadSql Sql = Sql("load " + extensionName + ";")
	if extensionType == COMMUNITY_EXTENSION {
		installSql = Sql("install " + extensionName + " from community;")
	} else {
		installSql = Sql("install " + extensionName + ";")
	}
	_, err := RunSqlQuery(conn, installSql)
	if err != nil {
		// Assume that the world depends on indicated extensions installing and loading properly
		log.Fatal().Err(err).Interface("extension", extensionName).Msg("unable to install " + extensionType + " extension")
	}
	_, err = RunSqlQuery(conn, loadSql)
	if err != nil {
		log.Fatal().Err(err).Interface("extension", extensionName).Msg("unable to load " + extensionType + " extension")
	}
}

type CoreExtension struct {
	Name string
}

func (ce *CoreExtension) EnsureWithConnection(conn *sql.Conn) {
	ensureExtension(conn, ce.Name, CORE_EXTENSION)
}

type CommunityExtension struct {
	Name string
}

func (e *CommunityExtension) EnsureWithConnection(conn *sql.Conn) {
	ensureExtension(conn, e.Name, COMMUNITY_EXTENSION)
}

type Extensions struct {
	Core      []string
	Community []string
}
