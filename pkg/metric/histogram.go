package metric

import (
	"database/sql"

	db "github.com/jakthom/hercules/pkg/db"
	"github.com/jakthom/hercules/pkg/labels"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/rs/zerolog/log"
)

type Histogram struct {
	Definition MetricDefinition
	Collector  *prometheus.HistogramVec
}

func (m *Histogram) AsVec() *prometheus.HistogramVec {
	v := prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Name:    m.Definition.FullName(),
		Help:    m.Definition.Help,
		Buckets: m.Definition.Buckets,
	}, m.Definition.LabelNames())
	return v
}

func (m *Histogram) register() error {
	collector := m.AsVec()
	err := prometheus.Register(collector)
	m.Collector = collector
	return err
}

func (m *Histogram) reregister() error {
	// godd this is ugly, but it's the only way I've found to make a collector go back to zero (so data isn't dup'd per request)
	prometheus.Unregister(m.Collector)
	return m.register()
}

func (m *Histogram) Materialize(conn *sql.Conn) error {
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

func NewHistogram(definition MetricDefinition) Histogram {
	metric := Histogram{
		Definition: definition,
	}
	err := metric.register()
	if err != nil {
		log.Error().Err(err).Interface("metric", definition.FullName()).Msg("could not register metric")
	}
	return metric
}
