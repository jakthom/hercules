package registry

import (
	"database/sql"

	"github.com/jakthom/hercules/pkg/metrics"
	herculestypes "github.com/jakthom/hercules/pkg/types"
	"github.com/rs/zerolog/log"
)

type MetricRegistry struct {
	PackageName  herculestypes.PackageName
	MetricPrefix string
	Gauge        map[string]metrics.Gauge
	Counter      map[string]metrics.Counter
	Summary      map[string]metrics.Summary
	Histogram    map[string]metrics.Histogram
}

func (mr *MetricRegistry) Materialize(conn *sql.Conn) error { // TODO -> Make this return a list of "materialization errors" if something fails
	var m []metrics.Materializeable
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

func NewMetricRegistry(definitions metrics.MetricDefinitions, meta herculestypes.MetricMetadata) *MetricRegistry {
	r := MetricRegistry{}
	r.PackageName = meta.PackageName
	r.MetricPrefix = string(meta.MetricPrefix)
	r.Gauge = make(map[string]metrics.Gauge)
	r.Histogram = make(map[string]metrics.Histogram)
	r.Summary = make(map[string]metrics.Summary)
	r.Counter = make(map[string]metrics.Counter)

	for _, definition := range definitions.Gauge {
		g := metrics.NewGauge(definition, meta)
		r.Gauge[g.Definition.Name] = g
	}
	for _, definition := range definitions.Histogram {
		h := metrics.NewHistogram(definition, meta)
		r.Histogram[h.Definition.Name] = h
	}
	for _, definition := range definitions.Summary {
		s := metrics.NewSummary(definition, meta)
		r.Summary[s.Definition.Name] = s
	}
	for _, definition := range definitions.Counter {
		c := metrics.NewCounter(definition, meta)
		r.Counter[c.Definition.Name] = c
	}
	return &r
}
