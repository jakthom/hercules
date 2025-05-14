package db

import (
	"database/sql"

	"github.com/rs/zerolog/log"
)

// Function to update constants to follow Go naming conventions.
const (
	// CommunityExtension represents a community extension type.
	CommunityExtensionType string = "community"
	// CoreExtension represents a core extension type.
	CoreExtensionType string = "core"
)

func ensureExtension(conn *sql.Conn, extensionName string, extensionType string) {
	var installSQL SQL
	var loadSQL = SQL("load " + extensionName + ";")
	if extensionType == CommunityExtensionType {
		installSQL = SQL("install " + extensionName + " from community;")
	} else {
		installSQL = SQL("install " + extensionName + ";")
	}

	// Run installation query
	rows, err := RunSQLQuery(conn, installSQL)
	if err != nil {
		// Assume that the world depends on indicated extensions installing and loading properly
		log.Error().Err(err).
			Interface("extension", extensionName).
			Msg("unable to install " + extensionType + " extension")
		panic("Failed to install extension: " + err.Error())
	}
	defer func() {
		if closeErr := rows.Close(); closeErr != nil {
			log.Error().Err(closeErr).
				Interface("extension", extensionName).
				Msg("error closing rows after extension installation")
		}
	}()

	// Check for installation errors
	rowsErr := rows.Err()
	if rowsErr != nil {
		log.Error().Err(rowsErr).
			Interface("extension", extensionName).
			Msg("error during installation of " + extensionType + " extension")
		panic("Error during extension installation: " + rowsErr.Error())
	}

	// Run load query
	loadRows, err := RunSQLQuery(conn, loadSQL)
	if err != nil {
		log.Error().Err(err).
			Interface("extension", extensionName).
			Msg("unable to load " + extensionType + " extension")
		panic("Failed to load extension: " + err.Error())
	}
	defer func() {
		if closeErr := loadRows.Close(); closeErr != nil {
			log.Error().Err(closeErr).
				Interface("extension", extensionName).
				Msg("error closing rows after extension loading")
		}
	}()

	// Check for loading errors
	loadRowsErr := loadRows.Err()
	if loadRowsErr != nil {
		log.Error().Err(loadRowsErr).
			Interface("extension", extensionName).
			Msg("error during loading of " + extensionType + " extension")
		panic("Error during extension loading: " + loadRowsErr.Error())
	}

	log.Debug().Interface("extension", extensionName).Msg(extensionType + " extension ensured")
}

// TestHookEnsureExtension exposes the ensureExtension function for testing.
func TestHookEnsureExtension(conn *sql.Conn, extensionName string, extensionType string) {
	ensureExtension(conn, extensionName, extensionType)
}

type CoreExtension struct {
	Name string
}

func (ce *CoreExtension) ensureWithConnection(conn *sql.Conn) {
	ensureExtension(conn, ce.Name, CoreExtensionType)
}

type CommunityExtension struct {
	Name string
}

func (e *CommunityExtension) ensureWithConnection(conn *sql.Conn) {
	ensureExtension(conn, e.Name, CommunityExtensionType)
}

type Extensions struct {
	Core      []CoreExtension
	Community []CommunityExtension
}

func EnsureExtensionsWithConnection(extensions Extensions, conn *sql.Conn) {
	for _, coreExtension := range extensions.Core {
		coreExtension.ensureWithConnection(conn)
	}
	for _, communityExtension := range extensions.Community {
		communityExtension.ensureWithConnection(conn)
	}
}
