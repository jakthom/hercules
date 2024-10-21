package metrics

import (
	"database/sql"

	"github.com/dbecorp/ducktheus/pkg/labels"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/rs/zerolog/log"
)

type HistogramMetricDefinition struct {
	metricDefinition `mapstructure:",squash"`
	Buckets          []float64
}

type HistogramMetric struct {
	Definition   HistogramMetricDefinition
	GlobalLabels labels.GlobalLabels
	Collector    *prometheus.HistogramVec
}

func (m *HistogramMetric) AsVec() *prometheus.HistogramVec {
	var labels = m.GlobalLabels.LabelNames()
	labels = append(labels, m.Definition.Labels...)
	v := prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Name:    m.Definition.Name,
		Help:    m.Definition.Help,
		Buckets: m.Definition.Buckets,
	}, labels)
	return v
}

func (m *HistogramMetric) register() error {
	collector := m.AsVec()
	err := prometheus.Register(collector)
	m.Collector = collector
	return err
}

func (m *HistogramMetric) reregister() error {
	// godd this is ugly, but it's the only way I've found to make a collector go back to zero (so data isn't dup'd per request)
	prometheus.Unregister(m.Collector)
	return m.register()
}

func (m *HistogramMetric) materializeWithConnection(conn *sql.Conn) error {
	m.reregister()
	results, err := m.Definition.materializeWithConnection(conn)
	for _, r := range results {
		l := labels.Merge(r.StringifiedLabels(), m.GlobalLabels)
		m.Collector.With(l).Observe(r.Value)
	}
	if err != nil {
		log.Error().Err(err).Interface("metric", m.Definition.Name).Msg("could not calculate metric")
		return err
	}
	return nil
}

func NewHistogramMetric(definition HistogramMetricDefinition, labels labels.GlobalLabels) HistogramMetric {
	metric := HistogramMetric{
		Definition:   definition,
		GlobalLabels: labels,
	}
	metric.register()
	return metric
}
