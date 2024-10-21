package metrics

import (
	"database/sql"

	"github.com/dbecorp/hercules/pkg/labels"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/rs/zerolog/log"
)

type GaugeMetricDefinition struct {
	metricDefinition `mapstructure:",squash"`
}

type GaugeMetric struct {
	Definition   GaugeMetricDefinition
	GlobalLabels labels.GlobalLabels
	Collector    *prometheus.GaugeVec
}

func (m *GaugeMetric) AsVec() *prometheus.GaugeVec {
	// TODO -> Combine definition labels and global labels
	var labels = m.GlobalLabels.LabelNames()
	labels = append(labels, m.Definition.Labels...)
	v := prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: m.Definition.Name,
		Help: m.Definition.Help,
	}, labels)
	return v
}

func (m *GaugeMetric) register() error {
	collector := m.AsVec()
	err := prometheus.Register(collector)
	m.Collector = collector
	return err
}

func (m *GaugeMetric) reregister() error {
	// godd this is ugly, but it's the only way I've found to make a collector go back to zero (so data isn't dup'd per request)
	prometheus.Unregister(m.Collector)
	return m.register()
}

func (m *GaugeMetric) materializeWithConnection(conn *sql.Conn) error {
	m.reregister()
	results, err := m.Definition.materializeWithConnection(conn)
	if err != nil {
		log.Error().Interface("metric", m.Definition.Name).Msg("could not materialize metric")
		return err
	}
	for _, r := range results {
		l := labels.Merge(r.StringifiedLabels(), m.GlobalLabels)
		m.Collector.With(l).Set(r.Value)
	}
	return nil
}

func NewGaugeMetric(definition GaugeMetricDefinition, labels labels.GlobalLabels) GaugeMetric {
	metric := GaugeMetric{
		Definition:   definition,
		GlobalLabels: labels,
	}
	metric.register()
	return metric
}
