package metrics

import (
	"database/sql"

	"github.com/dbecorp/hercules/pkg/labels"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/rs/zerolog/log"
)

type SummaryMetricDefinition struct {
	metricDefinition `mapstructure:",squash"`
	Objectives       []float64
}

type SummaryMetric struct {
	Definition   SummaryMetricDefinition
	GlobalLabels labels.GlobalLabels
	Collector    *prometheus.SummaryVec
}

func (m *SummaryMetric) AsVec() *prometheus.SummaryVec {
	objectives := make(map[float64]float64)
	for _, o := range m.Definition.Objectives {
		objectives[o] = o
	}
	var labels = m.GlobalLabels.LabelNames()
	labels = append(labels, m.Definition.Labels...)
	v := prometheus.NewSummaryVec(prometheus.SummaryOpts{
		Name:       m.Definition.Name,
		Help:       m.Definition.Help,
		Objectives: objectives,
	}, labels)
	return v
}

func (m *SummaryMetric) register() error {
	collector := m.AsVec()
	err := prometheus.Register(collector)
	m.Collector = collector
	return err
}

func (m *SummaryMetric) reregister() error {
	// godd this is ugly, but it's the only way I've found to make a collector go back to zero (so data isn't dup'd per request)
	prometheus.Unregister(m.Collector)
	return m.register()
}

func (m *SummaryMetric) materializeWithConnection(conn *sql.Conn) error {
	m.reregister()
	results, err := m.Definition.materializeWithConnection(conn)
	if err != nil {
		log.Error().Interface("metric", m.Definition.Name).Msg("could not materialize metric")
		return err
	}
	for _, r := range results {
		l := labels.Merge(r.StringifiedLabels(), m.GlobalLabels)
		m.Collector.With(l).Observe(r.Value)
	}
	return nil
}

func NewSummaryMetric(definition SummaryMetricDefinition, labels labels.GlobalLabels) SummaryMetric {
	metric := SummaryMetric{
		Definition:   definition,
		GlobalLabels: labels,
	}
	metric.register()
	return metric
}
