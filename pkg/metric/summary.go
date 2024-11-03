package metric

import (
	"database/sql"

	db "github.com/jakthom/hercules/pkg/db"
	"github.com/jakthom/hercules/pkg/labels"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/rs/zerolog/log"
)

type Summary struct {
	Definition MetricDefinition
	Collector  *prometheus.SummaryVec
}

func (m *Summary) AsVec() *prometheus.SummaryVec {
	objectives := make(map[float64]float64)
	for _, o := range m.Definition.Objectives {
		objectives[o] = o
	}
	v := prometheus.NewSummaryVec(prometheus.SummaryOpts{
		Name:       m.Definition.FullName(),
		Help:       m.Definition.Help,
		Objectives: objectives,
	}, m.Definition.LabelNames())
	return v
}

func (m *Summary) register() error {
	collector := m.AsVec()
	err := prometheus.Register(collector)
	m.Collector = collector
	return err
}

func (m *Summary) reregister() error {
	// godd this is ugly, but it's the only way I've found to make a collector go back to zero (so data isn't dup'd per request)
	prometheus.Unregister(m.Collector)
	return m.register()
}

func (m *Summary) Materialize(conn *sql.Conn) error {
	err := m.reregister()
	if err != nil {
		log.Error().Err(err).Interface("metric", m.Definition.FullName()).Msg("could not materialize metric")
	}
	results, err := db.Materialize(conn, m.Definition.Sql)
	if err != nil {
		log.Error().Interface("metric", m.Definition.FullName()).Msg("could not materialize metric")
		return err
	}
	for _, r := range results {
		l := labels.Merge(r.StringifiedLabels(), m.Definition.Metadata.Labels)
		m.Collector.With(map[string]string(l)).Observe(r.Value)
	}
	return nil
}

func NewSummary(definition MetricDefinition) Summary {
	metric := Summary{
		Definition: definition,
	}
	err := metric.register()
	if err != nil {
		log.Error().Err(err).Interface("metric", definition.Name).Msg("could not register metric")
	}
	return metric
}
