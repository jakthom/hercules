package db

import (
	"database/sql"
	"strings"
)

// Use DuckDB's own parser to get all column names from the query
func GetLabelNamesFromQuery(conn *sql.Conn, query Sql) []string {
	parseSql := Sql(`select coalesce(nullif(row->>'alias', ''), row->>'$.column_names[-1]') from (select unnest::json as row from unnest(json_serialize_sql('` + strings.Replace(string(query), "'", "''", -1) + `')->>'$.statements[0].node.select_list[*]'));`)
	rows, _ := RunSqlQuery(conn, parseSql)
	var columns []string
	for rows.Next() {
		var column string
		_ = rows.Scan(&column)
		if column != "value" && column != "val" && column != "v" && column != "" {
			columns = append(columns, column)
		}
	}
	return columns
}
