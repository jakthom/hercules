package metric

import (
	"database/sql"

	db "github.com/jakthom/hercules/pkg/db"
	"github.com/jakthom/hercules/pkg/labels"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/rs/zerolog/log"
)

type Gauge struct {
	Definition Definition
	Collector  *prometheus.GaugeVec
}

func NewGauge(definition Definition) Gauge {
	// TODO! Turn this into a generic function instead of copy/pasta
	metric := Gauge{
		Definition: definition,
	}
	err := metric.register()
	if err != nil {
		log.Error().Err(err).Interface("metric", definition.FullName()).Msg("could not register metric")
	}
	return metric
}

func (m *Gauge) asVec() *prometheus.GaugeVec {
	v := prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: m.Definition.FullName(),
		Help: m.Definition.Help,
	}, m.Definition.LabelNames())
	return v
}

func (m *Gauge) register() error {
	collector := m.asVec()
	err := prometheus.Register(collector)
	m.Collector = collector
	return err
}

func (m *Gauge) reregister() error {
	// godd this is ugly, but it's the only way I've found to make a collector go back to zero (so data isn't dup'd per request)
	prometheus.Unregister(m.Collector)
	return m.register()
}

func (m *Gauge) Materialize(conn *sql.Conn) error {
	err := m.reregister()
	if err != nil {
		log.Error().Err(err).Interface("metric", m.Definition.FullName()).Msg("could not materialize metric")
	}

	results, err := db.Materialize(conn, m.Definition.SQL)
	if err != nil {
		log.Error().Interface("metric", m.Definition.FullName()).Msg("could not materialize metric")
		return err
	}
	for _, r := range results {
		l := labels.Merge(r.StringifiedLabels(), m.Definition.Metadata.Labels)
		m.Collector.With(map[string]string(l)).Set(r.Value)
	}
	return nil
}
