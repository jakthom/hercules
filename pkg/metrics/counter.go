package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
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
