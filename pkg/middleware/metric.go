package middleware

import (
	"database/sql"
	"net/http"

	"github.com/dbecorp/ducktheus_exporter/pkg/metrics"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/rs/zerolog/log"
)

func MetricsMiddleware(conn *sql.Conn, metricDefinitions metrics.MetricDefinitions, gauges map[string]*prometheus.GaugeVec, histograms map[string]*prometheus.HistogramVec, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		for _, gauge := range metricDefinitions.Gauge {
			// Get results from DuckDB database
			results, err := gauge.MaterializeWithConnection(conn)
			// Get corresponding prom collector
			g := gauges[gauge.Name]
			for _, r := range results {
				g.With(r.StringifiedLabels()).Set(r.Value)
			}
			if err != nil {
				log.Error().Err(err).Interface("metric", gauge.Name).Msg("could not calculate metric")
			}
		}

		for _, histogram := range metricDefinitions.Histogram {
			// Get results from DuckDB database
			results, err := histogram.MaterializeWithConnection(conn)
			// Get corresponding prom collector
			h := histograms[histogram.Name]
			for _, r := range results {
				h.With(r.StringifiedLabels()).Observe(r.Value)
			}
			if err != nil {
				log.Error().Err(err).Interface("metric", histogram.Name).Msg("could not calculate metric")
			}
		}
		next.ServeHTTP(w, r)
	})
}
