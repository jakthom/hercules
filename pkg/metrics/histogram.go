package metrics

import (
	"database/sql"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/rs/zerolog/log"
)

type HistogramMetricDefinition struct {
	metricDefinition `mapstructure:",squash"`
	Buckets          []float64
}

func (m *HistogramMetricDefinition) AsVec() *prometheus.HistogramVec {
	v := prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Name:    m.Name,
		Help:    m.Help,
		Buckets: m.Buckets,
	}, m.Labels)
	return v
}

type HistogramMetric struct {
	Definition HistogramMetricDefinition
	Collector  *prometheus.HistogramVec
}

func (h *HistogramMetric) register() error {
	collector := h.Definition.AsVec()
	err := prometheus.Register(collector)
	h.Collector = collector
	return err
}

func (h *HistogramMetric) reregister() error {
	// godd this is ugly, but it's the only way I've found to make a collector go back to zero (so data isn't dup'd per request)
	prometheus.Unregister(h.Collector)
	return h.register()
}

func (h *HistogramMetric) materializeWithConnection(conn *sql.Conn) error {
	h.reregister()
	results, err := h.Definition.materializeWithConnection(conn)
	for _, r := range results {
		h.Collector.With(r.StringifiedLabels()).Observe(r.Value)
	}
	if err != nil {
		log.Error().Err(err).Interface("metric", h.Definition.Name).Msg("could not calculate metric")
		return err
	}
	return nil
}

func NewHistogramMetric(definition HistogramMetricDefinition) HistogramMetric {
	metric := HistogramMetric{
		Definition: definition,
	}
	metric.register()
	return metric
}
