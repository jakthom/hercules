package metrics

import (
	"database/sql"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/rs/zerolog/log"
)

type CounterMetricDefinition struct {
	metricDefinition `mapstructure:",squash"`
}

func (m *CounterMetricDefinition) AsVec() *prometheus.CounterVec {
	v := prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: m.Name,
		Help: m.Help,
	}, m.Labels)
	return v
}

type CounterMetric struct {
	Definition CounterMetricDefinition
	Collector  *prometheus.CounterVec
}

func (c *CounterMetric) reregister() error {
	// godd this is ugly, but it's the only way I've found to make a collector go back to zero (so data isn't dup'd per request)
	prometheus.Unregister(c.Collector)
	collector := c.Definition.AsVec()
	prometheus.Register(collector)
	c.Collector = collector
	return nil
}

func (c *CounterMetric) materializeWithConnection(conn *sql.Conn) error {
	c.reregister()
	results, err := c.Definition.materializeWithConnection(conn)
	for _, r := range results {
		c.Collector.With(r.StringifiedLabels()).Inc()
	}
	if err != nil {
		log.Error().Err(err).Interface("metric", c.Definition.Name).Msg("could not calculate metric")
		return err
	}
	return nil
}
