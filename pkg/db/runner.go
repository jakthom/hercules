package db

import (
	"context"
	"database/sql"
	"strconv"

	"github.com/rs/zerolog/log"
)

type QueryResult struct {
	Value  float64
	Labels map[string]interface{}
}

func (qr *QueryResult) StringifiedLabels() map[string]string {
	r := make(map[string]string)
	for k, v := range qr.Labels {
		if v == nil {
			v = "null"
		}
		r[k] = v.(string)
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

func MaterializeWithConnection(c *sql.Conn, s Sql) ([]QueryResult, error) {
	rows, _ := RunSqlQuery(c, s)
	var queryResults []QueryResult
	columns, _ := rows.Columns()
	for rows.Next() {
		queryResult := QueryResult{}

		queryResult.Labels = make(map[string]interface{})
		results := make([]interface{}, len(columns))
		for i := range results {
			results[i] = new(sql.RawBytes)
		}
		if err := rows.Scan(results...); err != nil {
			log.Error().Err(err).Msg("could not scan row")
		}
		for i, v := range results {
			if sb, ok := v.(*sql.RawBytes); ok {
				if columns[i] == "value" || columns[i] == "val" || columns[i] == "v" {
					queryResult.Value, _ = strconv.ParseFloat(string(*sb), 64)
				} else {
					queryResult.Labels[columns[i]] = string(*sb)
				}
			}
			queryResults = append(queryResults, queryResult)
		}
	}
	return queryResults, nil
}
