package metric

import (
	"database/sql"

	db "github.com/jakthom/hercules/pkg/db"
	"github.com/jakthom/hercules/pkg/labels"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/rs/zerolog/log"
)

type Counter struct {
	Definition MetricDefinition
	Collector  *prometheus.CounterVec
}

func (m *Counter) AsVec() *prometheus.CounterVec {
	v := prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: m.Definition.FullName(),
		Help: m.Definition.Help,
	}, m.Definition.LabelNames())
	return v
}

func (m *Counter) register() error {
	collector := m.AsVec()
	err := prometheus.Register(collector)
	m.Collector = collector
	return err
}

func (m *Counter) reregister() error {
	// godd this is ugly, but it's the only way I've found to make a collector go back to zero (so data isn't dup'd per request)
	prometheus.Unregister(m.Collector)
	return m.register()
}

func (m *Counter) Materialize(conn *sql.Conn) error {
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
		m.Collector.With(map[string]string(l)).Inc()
	}
	return nil
}

func NewCounter(definition MetricDefinition) Counter {
	metric := Counter{
		Definition: definition,
	}
	err := metric.register()
	if err != nil {
		log.Error().Err(err).Interface("metric", definition.Name).Msg("could not register metric")
	}
	return metric
}
