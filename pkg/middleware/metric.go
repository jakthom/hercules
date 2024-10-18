package middleware

import (
	"database/sql"
	"net/http"

	"github.com/dbecorp/ducktheus_exporter/pkg/flock"
	"github.com/dbecorp/ducktheus_exporter/pkg/metrics"
	"github.com/dbecorp/ducktheus_exporter/pkg/util"
	"github.com/rs/zerolog/log"
)

func MetricsMiddleware(conn *sql.Conn, metrics []metrics.Metric, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		for _, metric := range metrics {
			results, err := flock.RunMetric(conn, metric)
			if err != nil {
				log.Error().Err(err).Interface("metric", metric.Name).Msg("could not calculate metric: ")
			} else {
				util.Pprint(results)
			}
		}
		next.ServeHTTP(w, r)
	})
}
