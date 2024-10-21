package metrics

import (
	"database/sql"

	"github.com/dbecorp/ducktheus/pkg/labels"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/rs/zerolog/log"
)

type GaugeMetricDefinition struct {
	metricDefinition `mapstructure:",squash"`
}

type GaugeMetric struct {
	Definition   GaugeMetricDefinition
	GlobalLabels labels.GlobalLabels
	Collector    *prometheus.GaugeVec
}

func (m *GaugeMetric) AsVec() *prometheus.GaugeVec {
	// TODO -> Combine definition labels and global labels
	var labels = m.GlobalLabels.LabelNames()
	labels = append(labels, m.Definition.Labels...)
	v := prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: m.Definition.Name,
		Help: m.Definition.Help,
	}, labels)
	return v
}

func (g *GaugeMetric) register() error {
	collector := g.AsVec()
	err := prometheus.Register(collector)
	g.Collector = collector
	return err
}

func (g *GaugeMetric) reregister() error {
	// godd this is ugly, but it's the only way I've found to make a collector go back to zero (so data isn't dup'd per request)
	prometheus.Unregister(g.Collector)
	return g.register()
}

func (g *GaugeMetric) materializeWithConnection(conn *sql.Conn) error {
	g.reregister()
	results, err := g.Definition.materializeWithConnection(conn)
	for _, r := range results {
		l := labels.Merge(r.StringifiedLabels(), g.GlobalLabels)
		g.Collector.With(l).Set(r.Value)
	}
	if err != nil {
		log.Error().Err(err).Interface("metric", g.Definition.Name).Msg("could not calculate metric")
		return err
	}
	return nil
}

func NewGaugeMetric(definition GaugeMetricDefinition, labels labels.GlobalLabels) GaugeMetric {
	metric := GaugeMetric{
		Definition:   definition,
		GlobalLabels: labels,
	}
	metric.register()
	return metric
}
