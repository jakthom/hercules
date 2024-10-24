package middleware

import (
	"database/sql"
	"net/http"

	registry "github.com/dbecorp/hercules/pkg/metricRegistry"
)

func MetricsMiddleware(conn *sql.Conn, registries []*registry.MetricRegistry, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		for _, registry := range registries {
			registry.MaterializeWithConnection(conn)
		}
		next.ServeHTTP(w, r)
	})
}
