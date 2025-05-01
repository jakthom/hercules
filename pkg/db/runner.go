package db

import (
	"context"
	"database/sql"
	"fmt"
	"math/big"
	"strings"

	"github.com/rs/zerolog/log"
	"github.com/spf13/cast"
)

type QueryResult struct {
	Value  float64
	Labels map[string]string
}

func toFloat64(v any) float64 {
	switch v := v.(type) {
	case *big.Int:
		val, accuracy := v.Float64()
		if accuracy != big.Exact {
			log.Debug().Interface("value", v).Msg("inexact conversion from big.Int to float64")
		}
		return val
	case *big.Float:
		val, accuracy := v.Float64()
		if accuracy != big.Exact {
			log.Debug().Interface("value", v).Msg("inexact conversion from big.Float to float64")
		}
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

// RunSQLQuery executes a SQL query on the provided connection.
func RunSQLQuery(conn *sql.Conn, query SQL) (*sql.Rows, error) {
	log.Trace().Interface("query", query).Msg("running query")
	rows, err := conn.QueryContext(context.Background(), string(query))
	if err != nil {
		log.Error().Err(err).Interface("query", query).Msg("could not run query")
		return nil, err
	}
	return rows, nil
}

func processRow(columns []string, vals []interface{}) QueryResult {
	queryResult := QueryResult{
		Labels: make(map[string]string),
	}

	for i := range vals {
		columnName := columns[i]
		valuePtr, ok := vals[i].(*interface{})
		if !ok {
			log.Error().Str("column", columnName).Msg("failed to cast column value pointer")
			continue
		}

		value := *valuePtr
		if isValueColumn(columnName) {
			queryResult.Value = toFloat64(value)
		} else {
			strValue := convertToString(columnName, value)
			queryResult.Labels[columnName] = strValue
		}
	}

	return queryResult
}

func isValueColumn(columnName string) bool {
	return columnName == "value" || columnName == "val" || columnName == "v" || isFunctionColumn(columnName)
}

func convertToString(columnName string, value interface{}) string {
	if value == nil {
		return "NULL"
	}

	strValue, ok := value.(string)
	if !ok {
		log.Warn().
			Str("column", columnName).
			Interface("value", value).
			Msg("converting non-string value to string")
		strValue = fmt.Sprintf("%v", value)
	}
	return strValue
}

func Materialize(conn *sql.Conn, query SQL) ([]QueryResult, error) {
	rows, err := RunSQLQuery(conn, query)
	if err != nil {
		return nil, err
	}
	// rows is guaranteed to be non-nil if err is nil
	defer rows.Close()

	var queryResults []QueryResult

	// Get column names and column count from query results
	columns, err := rows.Columns()
	if err != nil {
		log.Error().Err(err).Msg("could not get columns")
		return nil, err
	}

	// Initialize values as interface{} pointers
	vals := prepareValueSlice(len(columns))

	// Process each row
	for rows.Next() {
		scanErr := rows.Scan(vals...)
		if scanErr != nil {
			log.Error().Err(scanErr).Msg("could not scan row")
			continue // Skip this row and continue with the next one
		}

		queryResult := processRow(columns, vals)
		queryResults = append(queryResults, queryResult)
	}

	// Check for any errors during iteration
	rowsErr := rows.Err()
	if rowsErr != nil {
		log.Error().Err(rowsErr).Msg("error during rows iteration")
		return queryResults, rowsErr
	}

	return queryResults, nil
}

func prepareValueSlice(columnCount int) []interface{} {
	vals := make([]interface{}, columnCount)
	for i := range vals {
		var ii interface{}
		vals[i] = &ii
	}
	return vals
}
