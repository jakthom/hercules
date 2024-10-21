package metrics

import (
	"database/sql"

	"github.com/dbecorp/hercules/pkg/labels"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/rs/zerolog/log"
)

type CounterMetricDefinition struct {
	metricDefinition `mapstructure:",squash"`
}

type CounterMetric struct {
	Definition   CounterMetricDefinition
	GlobalLabels labels.GlobalLabels
	Collector    *prometheus.CounterVec
}

func (m *CounterMetric) AsVec() *prometheus.CounterVec {
	var labels = m.GlobalLabels.LabelNames()
	labels = append(labels, m.Definition.Labels...)
	v := prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: m.Definition.Name,
		Help: m.Definition.Help,
	}, labels)
	return v
}

func (m *CounterMetric) register() error {
	collector := m.AsVec()
	err := prometheus.Register(collector)
	m.Collector = collector
	return err
}

func (m *CounterMetric) reregister() error {
	// godd this is ugly, but it's the only way I've found to make a collector go back to zero (so data isn't dup'd per request)
	prometheus.Unregister(m.Collector)
	return m.register()
}

func (m *CounterMetric) materializeWithConnection(conn *sql.Conn) error {
	m.reregister()
	results, err := m.Definition.materializeWithConnection(conn)
	for _, r := range results {
		l := labels.Merge(r.StringifiedLabels(), m.GlobalLabels)
		m.Collector.With(l).Inc()
	}
	if err != nil {
		log.Error().Err(err).Interface("metric", m.Definition.Name).Msg("could not calculate metric")
		return err
	}
	return nil
}

func NewCounterMetric(definition CounterMetricDefinition, labels labels.GlobalLabels) CounterMetric {
	metric := CounterMetric{
		Definition:   definition,
		GlobalLabels: labels,
	}
	metric.register()
	return metric
}
