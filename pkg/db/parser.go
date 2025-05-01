package db

import (
	"database/sql"
	"strings"
)

// GetLabelNamesFromQuery uses DuckDB's own parser to extract all column names from a query.
func GetLabelNamesFromQuery(conn *sql.Conn, query SQL) ([]string, error) {
	parseSQL := SQL(`select coalesce(nullif(row->>'alias', ''), row->>'$.column_names[-1]') 
                     from (select unnest::json as row 
                           from unnest(json_serialize_sql('` + strings.ReplaceAll(string(query), "'", "''") +
		`')->>'$.statements[0].node.select_list[*]'));`)

	rows, err := RunSQLQuery(conn, parseSQL)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var columns []string
	for rows.Next() {
		var column string
		scanErr := rows.Scan(&column)
		if scanErr != nil {
			return columns, scanErr
		}
		if column != "value" && column != "val" && column != "v" && column != "" {
			columns = append(columns, column)
		}
	}

	// Check for any errors during iteration.
	rowsErr := rows.Err()
	if rowsErr != nil {
		return columns, rowsErr
	}

	return columns, nil
}
