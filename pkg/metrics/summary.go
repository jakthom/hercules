package metrics

import (
	"database/sql"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/rs/zerolog/log"
)

type SummaryMetricDefinition struct {
	metricDefinition `mapstructure:",squash"`
	Objectives       []float64
}

func (m *SummaryMetricDefinition) AsVec() *prometheus.SummaryVec {
	objectives := make(map[float64]float64)
	for _, o := range m.Objectives {
		objectives[o] = o
	}
	v := prometheus.NewSummaryVec(prometheus.SummaryOpts{
		Name:       m.Name,
		Help:       m.Help,
		Objectives: objectives,
	}, m.Labels)
	return v
}

type SummaryMetric struct {
	Definition SummaryMetricDefinition
	Collector  *prometheus.SummaryVec
}

func (s *SummaryMetric) register() error {
	collector := s.Definition.AsVec()
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
		s.Collector.With(r.StringifiedLabels()).Observe(r.Value)
	}
	if err != nil {
		log.Error().Err(err).Interface("metric", s.Definition.Name).Msg("could not calculate metric")
		return err
	}
	return nil
}

func NewSummaryMetric(definition SummaryMetricDefinition) SummaryMetric {
	metric := SummaryMetric{
		Definition: definition,
	}
	metric.register()
	return metric
}
