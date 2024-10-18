package middleware

import (
	"database/sql"
	"net/http"

	"github.com/dbecorp/ducktheus_exporter/pkg/flock"
	"github.com/dbecorp/ducktheus_exporter/pkg/metrics"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/rs/zerolog/log"
)

func MetricsMiddleware(conn *sql.Conn, metrics []metrics.Metric, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		for _, metric := range metrics {
			// Get results from DuckDB database
			results, err := flock.RunMetric(conn, metric)
			for _, r := range results {

				prometheus.MustRegister(
					prometheus.NewGaugeFunc(
						prometheus.GaugeOpts{
							Name:        metric.Name,
							Help:        metric.Help,
							ConstLabels: r.StringifiedLabels(),
						},
						func() float64 { return r.Value },
					),
				)

			}
			if err != nil {
				log.Error().Err(err).Interface("metric", metric.Name).Msg("could not calculate metric")
			}
		}
		next.ServeHTTP(w, r)
	})
}
