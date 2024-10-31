package metrics

import (
	"database/sql"

	"github.com/jakthom/hercules/pkg/labels"
	herculestypes "github.com/jakthom/hercules/pkg/types"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/rs/zerolog/log"
)

type Gauge struct {
	Definition   MetricDefinition
	GlobalLabels labels.Labels
	Collector    *prometheus.GaugeVec
}

func (m *Gauge) AsVec() *prometheus.GaugeVec {
	// TODO -> Combine definition labels and global labels
	var labels = m.GlobalLabels.LabelNames()
	labels = append(labels, m.Definition.Labels...)
	v := prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: m.Definition.Name,
		Help: m.Definition.Help,
	}, labels)
	return v
}

func (m *Gauge) register() error {
	collector := m.AsVec()
	err := prometheus.Register(collector)
	m.Collector = collector
	return err
}

func (m *Gauge) reregister() error {
	// godd this is ugly, but it's the only way I've found to make a collector go back to zero (so data isn't dup'd per request)
	prometheus.Unregister(m.Collector)
	return m.register()
}

func (m *Gauge) MaterializeWithConnection(conn *sql.Conn) error {
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
		m.Collector.With(map[string]string(l)).Set(r.Value)
	}
	return nil
}

func NewGauge(definition MetricDefinition, meta herculestypes.MetricMetadata) Gauge {
	// TODO! Turn this into a generic function instead of copy/pasta
	definition.Name = meta.Prefix() + definition.Name
	metric := Gauge{
		Definition:   definition,
		GlobalLabels: meta.Labels,
	}
	err := metric.register()
	if err != nil {
		log.Error().Err(err).Interface("metric", definition.Name).Msg("could not register metric")
	}
	return metric
}
