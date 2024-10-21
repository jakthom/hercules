package metrics

import (
	"database/sql"

	"github.com/dbecorp/ducktheus/pkg/labels"
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

func (s *SummaryMetric) register() error {
	collector := s.AsVec()
	err := prometheus.Register(collector)
	s.Collector = collector
	return err
}

func (s *SummaryMetric) reregister() error {
	// godd this is ugly, but it's the only way I've found to make a collector go back to zero (so data isn't dup'd per request)
	prometheus.Unregister(s.Collector)
	return s.register()
}

func (s *SummaryMetric) materializeWithConnection(conn *sql.Conn) error {
	s.reregister()
	results, err := s.Definition.materializeWithConnection(conn)
	for _, r := range results {
		l := labels.Merge(r.StringifiedLabels(), s.GlobalLabels)
		s.Collector.With(l).Observe(r.Value)
	}
	if err != nil {
		log.Error().Err(err).Interface("metric", s.Definition.Name).Msg("could not calculate metric")
		return err
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
