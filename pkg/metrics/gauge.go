package metrics

import (
	"database/sql"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/rs/zerolog/log"
)

type GaugeMetricDefinition struct {
	metricDefinition `mapstructure:",squash"`
}

func (m *GaugeMetricDefinition) AsVec() *prometheus.GaugeVec {
	v := prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: m.Name,
		Help: m.Help,
	}, m.Labels)
	return v
}

func (m *GaugeMetricDefinition) AsGauge() *prometheus.Gauge {
	v := prometheus.NewGauge(prometheus.GaugeOpts{
		Name: m.Name,
		Help: m.Help,
	})
	return &v
}

type GaugeMetric struct {
	Definition GaugeMetricDefinition
	Collector  *prometheus.GaugeVec
}

func (g *GaugeMetric) register() error {
	collector := g.Definition.AsVec()
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
		g.Collector.With(r.StringifiedLabels()).Set(r.Value)
	}
	if err != nil {
		log.Error().Err(err).Interface("metric", g.Definition.Name).Msg("could not calculate metric")
		return err
	}
	return nil
}

func NewGaugeMetric(definition GaugeMetricDefinition) GaugeMetric {
	metric := GaugeMetric{
		Definition: definition,
	}
	metric.register()
	return metric
}
