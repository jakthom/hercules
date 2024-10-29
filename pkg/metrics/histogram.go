package metrics

import (
	"database/sql"

	"github.com/jakthom/hercules/pkg/labels"
	herculestypes "github.com/jakthom/hercules/pkg/types"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/rs/zerolog/log"
)

type HistogramMetricDefinition struct {
	metricDefinition `mapstructure:",squash"`
	Buckets          []float64
}

type HistogramMetric struct {
	Definition   HistogramMetricDefinition
	GlobalLabels labels.Labels
	Collector    *prometheus.HistogramVec
}

func (m *HistogramMetric) AsVec() *prometheus.HistogramVec {
	var labels = m.GlobalLabels.LabelNames()
	labels = append(labels, m.Definition.Labels...)
	v := prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Name:    m.Definition.Name,
		Help:    m.Definition.Help,
		Buckets: m.Definition.Buckets,
	}, labels)
	return v
}

func (m *HistogramMetric) register() error {
	collector := m.AsVec()
	err := prometheus.Register(collector)
	m.Collector = collector
	return err
}

func (m *HistogramMetric) reregister() error {
	// godd this is ugly, but it's the only way I've found to make a collector go back to zero (so data isn't dup'd per request)
	prometheus.Unregister(m.Collector)
	return m.register()
}

func (m *HistogramMetric) MaterializeWithConnection(conn *sql.Conn) error {
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

func NewHistogramMetric(definition HistogramMetricDefinition, meta herculestypes.MetricMetadata) HistogramMetric {
	// TODO! Turn this into a generic function instead of copy/pasta
	definition.Name = string(meta.MetricPrefix) + string(meta.PackageName) + "_" + definition.Name
	metric := HistogramMetric{
		Definition:   definition,
		GlobalLabels: meta.Labels,
	}
	err := metric.register()
	if err != nil {
		log.Error().Err(err).Interface("metric", definition.Name).Msg("could not register metric")
	}
	return metric
}
