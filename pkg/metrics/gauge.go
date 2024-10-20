package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
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
