package middleware

import (
	"database/sql"
	"net/http"

	"github.com/dbecorp/ducktheus_exporter/pkg/metrics"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/rs/zerolog/log"
)

func MetricsMiddleware(conn *sql.Conn, metricDefinitions metrics.MetricDefinitions, gauges map[string]*prometheus.GaugeVec, histograms map[string]*prometheus.HistogramVec, summaries map[string]*prometheus.SummaryVec, counters map[string]*prometheus.CounterVec, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		// TODO -> This needs a full/consolidated rewrite

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

			// DYING BUT IT'S THE ONLY WAY TO MAKE A SUMMARY GO BACK TO 0 AND REUSE THE NAME
			prometheus.Unregister(h)
			// OGOD THIS IS UGLY
			h = histogram.AsHistogramVec()
			prometheus.Register(h)
			for _, r := range results {
				h.With(r.StringifiedLabels()).Observe(r.Value)
			}
			if err != nil {
				log.Error().Err(err).Interface("metric", histogram.Name).Msg("could not calculate metric")
			}
		}

		for _, summary := range metricDefinitions.Summary {
			// Get results from DuckDB database
			results, err := summary.MaterializeWithConnection(conn)
			// Get corresponding prom collector
			s := summaries[summary.Name]
			// DYING BUT IT'S THE ONLY WAY TO MAKE A SUMMARY GO BACK TO 0 AND REUSE THE NAME
			prometheus.Unregister(s)
			// OGOD THIS IS UGLY
			s = summary.AsSummaryVec()
			prometheus.Register(s)
			for _, r := range results {
				s.With(r.StringifiedLabels()).Observe(r.Value)
			}
			if err != nil {
				log.Error().Err(err).Interface("metric", summary.Name).Msg("could not calculate metric")
			}
		}

		for _, counter := range metricDefinitions.Counter {
			// Get results from DuckDB database
			results, err := counter.MaterializeWithConnection(conn)
			// Get corresponding prom collector
			c := counters[counter.Name]
			// DYING BUT IT'S THE ONLY WAY TO MAKE A COUNTER GO BACK TO 0 AND REUSE THE NAME
			prometheus.Unregister(c)
			// OGOD THIS IS UGLY
			c = counter.AsCounterVec()
			prometheus.Register(c)
			// AHHHHHHHHH
			counters[counter.Name] = c
			for _, r := range results {
				c.With(r.StringifiedLabels()).Inc()
			}
			if err != nil {
				log.Error().Err(err).Interface("metric", counter.Name).Msg("could not calculate metric")
			}
		}

		next.ServeHTTP(w, r)
	})
}
