package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
)

type HistogramMetricDefinition struct {
	metricDefinition `mapstructure:",squash"`
	Buckets          []float64
}

func (m *HistogramMetricDefinition) AsVec() *prometheus.HistogramVec {
	v := prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Name:    m.Name,
		Help:    m.Help,
		Buckets: m.Buckets,
	}, m.Labels)
	return v
}
