package testutil

import (
	"context"
	"database/sql"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/rs/zerolog/log"
)

func GetMockedConnection() (*sql.Conn, sqlmock.Sqlmock, error) {
	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherEqual))
	conn, _ := db.Conn(context.Background())
	if err != nil {
		log.Fatal().Err(err).Msg("error '%s' was not expected when opening a stub database connection")
		return nil, nil, err
	}
	defer db.Close()
	return conn, mock, nil
}
