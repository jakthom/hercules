package middleware

import (
	"database/sql"
	"net/http"

	"github.com/dbecorp/ducktheus_exporter/pkg/metrics"
)

func MetricsMiddleware(conn *sql.Conn, registry *metrics.MetricRegistry, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		registry.MaterializeWithConnection(conn)
		next.ServeHTTP(w, r)
	})
}
