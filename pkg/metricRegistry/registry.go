package registry

import (
	"database/sql"

	"github.com/jakthom/hercules/pkg/metric"
	"github.com/rs/zerolog/log"
)

type MetricRegistry struct {
	Gauge     map[string]metric.Gauge
	Counter   map[string]metric.Counter
	Summary   map[string]metric.Summary
	Histogram map[string]metric.Histogram
}

func (mr *MetricRegistry) Materialize(conn *sql.Conn) error { // TODO -> Make this return a list of "materialization errors" if something fails
	var m []metric.Materializeable
	for _, metric := range mr.Gauge {
		m = append(m, &metric)
	}
	for _, metric := range mr.Histogram {
		m = append(m, &metric)
	}
	for _, metric := range mr.Summary {
		m = append(m, &metric)
	}
	for _, metric := range mr.Counter {
		m = append(m, &metric)
	}
	for _, materializable := range m {
		err := materializable.Materialize(conn)
		if err != nil {
			log.Error().Err(err).Msg("could not materialize metric")
		}
	}
	return nil
}

func NewMetricRegistry(definitions metric.MetricDefinitions) *MetricRegistry {
	r := MetricRegistry{}
	r.Gauge = make(map[string]metric.Gauge)
	r.Histogram = make(map[string]metric.Histogram)
	r.Summary = make(map[string]metric.Summary)
	r.Counter = make(map[string]metric.Counter)

	for _, definition := range definitions.Gauge {
		g := metric.NewGauge(*definition)
		r.Gauge[g.Definition.FullName()] = g
	}
	for _, definition := range definitions.Histogram {
		h := metric.NewHistogram(*definition)
		r.Histogram[h.Definition.FullName()] = h
	}
	for _, definition := range definitions.Summary {
		s := metric.NewSummary(*definition)
		r.Summary[s.Definition.FullName()] = s
	}
	for _, definition := range definitions.Counter {
		c := metric.NewCounter(*definition)
		r.Counter[c.Definition.FullName()] = c
	}
	return &r
}
