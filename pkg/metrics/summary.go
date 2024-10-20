package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
)

type SummaryMetricDefinition struct {
	metricDefinition `mapstructure:",squash"`
	Objectives       []float64
}

func (m *SummaryMetricDefinition) AsVec() *prometheus.SummaryVec {
	objectives := make(map[float64]float64)
	for _, o := range m.Objectives {
		objectives[o] = o
	}
	v := prometheus.NewSummaryVec(prometheus.SummaryOpts{
		Name:       m.Name,
		Help:       m.Help,
		Objectives: objectives,
	}, m.Labels)
	return v
}
