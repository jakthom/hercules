package metrics

import (
	"database/sql"

	"github.com/jakthom/hercules/pkg/labels"
	herculestypes "github.com/jakthom/hercules/pkg/types"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/rs/zerolog/log"
)

type SummaryMetricDefinition struct {
	metricDefinition `mapstructure:",squash"`
	Objectives       []float64
}

type SummaryMetric struct {
	Definition   SummaryMetricDefinition
	GlobalLabels labels.Labels
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

func (m *SummaryMetric) MaterializeWithConnection(conn *sql.Conn) error {
	err := m.reregister()
	if err != nil {
		log.Error().Err(err).Interface("metric", m.Definition.Name).Msg("could not materialize metric")
	}
	results, err := m.Definition.materializeWithConnection(conn)
	if err != nil {
		log.Error().Interface("metric", m.Definition.Name).Msg("could not materialize metric")
		return err
	}
	for _, r := range results {
		l := labels.Merge(r.StringifiedLabels(), m.GlobalLabels)
		m.Collector.With(map[string]string(l)).Observe(r.Value)
	}
	return nil
}

func NewSummaryMetric(definition SummaryMetricDefinition, meta herculestypes.MetricMetadata) SummaryMetric {
	// TODO! Turn this into a generic function instead of copy/pasta
	definition.Name = string(meta.MetricPrefix) + string(meta.PackageName) + "_" + definition.Name
	metric := SummaryMetric{
		Definition:   definition,
		GlobalLabels: meta.Labels,
	}
	err := metric.register()
	if err != nil {
		log.Error().Err(err).Interface("metric", definition.Name).Msg("could not register metric")
	}
	return metric
}
