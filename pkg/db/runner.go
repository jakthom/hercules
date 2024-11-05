package db

import (
	"context"
	"database/sql"
	"math/big"
	"strings"

	"github.com/rs/zerolog/log"
	"github.com/spf13/cast"
)

type QueryResult struct {
	Value  float64
	Labels map[string]string
}

func toFloat64(v interface{}) float64 {
	switch v := v.(type) {
	case *big.Int:
		val, _ := v.Float64()
		return val
	case *big.Float:
		val, _ := v.Float64()
		return val
	default:
		return cast.ToFloat64(v)
	}
}
func isFunctionColumn(column string) bool {
	return strings.Contains(column, "(") && strings.Contains(column, ")")
}

func (qr *QueryResult) StringifiedLabels() map[string]string {
	r := make(map[string]string)
	for k, v := range qr.Labels {
		r[k] = v
	}
	return r
}

func RunSqlQuery(conn *sql.Conn, query Sql) (*sql.Rows, error) {
	log.Trace().Interface("query", query).Msg("running query")
	rows, err := conn.QueryContext(context.Background(), string(query))
	if err != nil {
		log.Error().Err(err).Interface("query", query).Msg("could not run query")
	}
	return rows, err
}

func Materialize(conn *sql.Conn, query Sql) ([]QueryResult, error) {
	rows, _ := RunSqlQuery(conn, query)
	var queryResults []QueryResult
	// Get column names and column count from query results
	columns, _ := rows.Columns()
	columnCount := len(columns)
	// Initialize values as interface{} pointers
	vals := make([]interface{}, columnCount)
	for i := range columns {
		var ii interface{}
		vals[i] = &ii
	}

	for rows.Next() {
		queryResult := QueryResult{}
		queryResult.Labels = make(map[string]string)
		if err := rows.Scan(vals...); err != nil {
			log.Error().Err(err).Msg("could not scan row")
		}
		for i := range vals {
			columnName := columns[i]
			value := *(vals[i].(*interface{}))
			if columnName == "value" || columnName == "val" || columnName == "v" || isFunctionColumn(columnName) {
				queryResult.Value = toFloat64(value)
			} else {
				queryResult.Labels[columnName] = value.(string)
			}
		}
		queryResults = append(queryResults, queryResult)
	}
	return queryResults, nil
}
