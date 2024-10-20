package middleware

import (
	"database/sql"
	"net/http"

	"github.com/dbecorp/ducktheus_exporter/pkg/metrics"
)

func MetricsMiddleware(conn *sql.Conn, registry *metrics.MetricRegistry, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		for _, gauge := range registry.Gauge {
			gauge.MaterializeWithConnection(conn)
		}

		for _, histogram := range registry.Histogram {
			histogram.MaterializeWithConnection(conn)
		}

		for _, summary := range registry.Summary {
			summary.MaterializeWithConnection(conn)
		}

		for _, counter := range registry.Counter {
			counter.MaterializeWithConnection(conn)
		}

		next.ServeHTTP(w, r)
	})
}
