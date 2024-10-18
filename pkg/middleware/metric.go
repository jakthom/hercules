package middleware

import (
	"database/sql"
	"net/http"

	"github.com/dbecorp/ducktheus_exporter/pkg/flock"
	"github.com/dbecorp/ducktheus_exporter/pkg/metrics"
	"github.com/rs/zerolog/log"
)

func MetricsMiddleware(conn *sql.Conn, metrics []metrics.Metric, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		for _, metric := range metrics {
			collector := metric.AsCollector()
			// Get results from DuckDB database
			results, err := flock.RunMetric(conn, metric)
			for _, r := range results {
				collector.With(r.Labels).Set(r.Value)
			}
			if err != nil {
				log.Error().Err(err).Interface("metric", metric.Name).Msg("could not calculate metric")
			}
		}
		next.ServeHTTP(w, r)
	})
}
