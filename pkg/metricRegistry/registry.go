package registry

import (
	"database/sql"

	herculespackage "github.com/dbecorp/hercules/pkg/herculesPackage"
	"github.com/dbecorp/hercules/pkg/labels"
	"github.com/dbecorp/hercules/pkg/metrics"
	herculestypes "github.com/dbecorp/hercules/pkg/types"
	"github.com/rs/zerolog/log"
)

type MetricRegistry struct {
	Package      herculespackage.Package
	MetricPrefix string
	Gauge        map[string]metrics.GaugeMetric
	Counter      map[string]metrics.CounterMetric
	Summary      map[string]metrics.SummaryMetric
	Histogram    map[string]metrics.HistogramMetric
}

func (mr *MetricRegistry) MaterializeWithConnection(conn *sql.Conn) error { // TODO -> Make this return a list of "materialization errors" if something fails
	for _, gauge := range mr.Gauge {
		err := gauge.MaterializeWithConnection(conn)
		if err != nil {
			log.Error().Err(err)
		}
	}

	for _, histogram := range mr.Histogram {
		err := histogram.MaterializeWithConnection(conn)
		if err != nil {
			log.Error().Err(err)
		}
	}

	for _, summary := range mr.Summary {
		err := summary.MaterializeWithConnection(conn)
		if err != nil {
			log.Error().Err(err)
		}
	}

	for _, counter := range mr.Counter {
		err := counter.MaterializeWithConnection(conn)
		if err != nil {
			log.Error().Err(err)
		}
	}
	return nil
}

func NewMetricRegistryfromPackage(p herculespackage.Package, l labels.GlobalLabels) *MetricRegistry {
	meta := herculestypes.MetricMetadata{
		PackageName:  p.Name,
		MetricPrefix: p.MetricPrefix,
		Labels:       l,
	}
	r := MetricRegistry{}
	r.Package = p
	r.MetricPrefix = string(meta.MetricPrefix)
	r.Gauge = make(map[string]metrics.GaugeMetric)
	r.Histogram = make(map[string]metrics.HistogramMetric)
	r.Summary = make(map[string]metrics.SummaryMetric)
	r.Counter = make(map[string]metrics.CounterMetric)

	for _, definition := range p.Metrics.Gauge {
		g := metrics.NewGaugeMetric(definition, meta)
		r.Gauge[g.Definition.Name] = g
	}
	for _, definition := range p.Metrics.Histogram {
		h := metrics.NewHistogramMetric(definition, meta)
		r.Histogram[h.Definition.Name] = h
	}
	for _, definition := range p.Metrics.Summary {
		s := metrics.NewSummaryMetric(definition, meta)
		r.Summary[s.Definition.Name] = s
	}
	for _, definition := range p.Metrics.Counter {
		c := metrics.NewCounterMetric(definition, meta)
		r.Counter[c.Definition.Name] = c
	}
	return &r
}
