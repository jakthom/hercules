package metrics

import (
	"database/sql"

	"github.com/dbecorp/ducktheus/pkg/db"
	"github.com/dbecorp/ducktheus/pkg/labels"
	"github.com/dbecorp/ducktheus/pkg/util"
	"github.com/rs/zerolog/log"
)

type SourceType string
type MetricType string

const (
	// Metric Types
	CounterMetricType   MetricType = "counter"
	GaugeMetricType     MetricType = "gauge"
	HistogramMetricType MetricType = "histogram"
	SummaryMetricType   MetricType = "summary"
)

type metricDefinition struct {
	Name    string     `json:"name"`
	Enabled bool       `json:"enabled"`
	Type    MetricType `json:"type"`
	Help    string     `json:"help"`
	Sql     db.Sql     `json:"sql"`
	Labels  []string   `json:"labels"`
}

func (md *metricDefinition) materializeWithConnection(conn *sql.Conn) ([]QueryResult, error) {
	rows, err := db.RunSqlQuery(conn, md.Sql)
	var results []QueryResult
	for rows.Next() {
		var result QueryResult
		if err := rows.Scan(&result.Labels, &result.Value); err != nil {
			log.Error().Err(err).Msg("error when scanning query results")
			return nil, err
		}
		results = append(results, result)
	}
	return results, err
}

type MetricDefinitions struct {
	Gauge     []GaugeMetricDefinition     `json:"gauge"`
	Counter   []CounterMetricDefinition   `json:"counter"`
	Summary   []SummaryMetricDefinition   `json:"summary"`
	Histogram []HistogramMetricDefinition `json:"histogram"`
}

type MetricRegistry struct {
	Gauge     map[string]GaugeMetric
	Counter   map[string]CounterMetric
	Summary   map[string]SummaryMetric
	Histogram map[string]HistogramMetric
}

func (mr *MetricRegistry) MaterializeWithConnection(conn *sql.Conn) error {
	for _, gauge := range mr.Gauge {
		err := gauge.materializeWithConnection(conn)
		if err != nil {
			log.Error().Err(err)
			return err
		}
	}

	for _, histogram := range mr.Histogram {
		err := histogram.materializeWithConnection(conn)
		if err != nil {
			log.Error().Err(err)
			return err
		}
	}

	for _, summary := range mr.Summary {
		err := summary.materializeWithConnection(conn)
		if err != nil {
			log.Error().Err(err)
			return err
		}
	}

	for _, counter := range mr.Counter {
		err := counter.materializeWithConnection(conn)
		if err != nil {
			log.Error().Err(err)
			return err
		}
	}
	return nil
}

func NewMetricRegistry(definitions MetricDefinitions, labels labels.Labels) *MetricRegistry {
	util.Pprint(labels)
	r := MetricRegistry{}
	r.Gauge = make(map[string]GaugeMetric)
	r.Histogram = make(map[string]HistogramMetric)
	r.Summary = make(map[string]SummaryMetric)
	r.Counter = make(map[string]CounterMetric)

	for _, definition := range definitions.Gauge {
		g := NewGaugeMetric(definition, labels)
		r.Gauge[g.Definition.Name] = g
	}
	for _, definition := range definitions.Histogram {
		h := NewHistogramMetric(definition)
		r.Histogram[h.Definition.Name] = h
	}
	for _, definition := range definitions.Summary {
		s := NewSummaryMetric(definition)
		r.Summary[s.Definition.Name] = s
	}
	for _, definition := range definitions.Counter {
		c := NewCounterMetric(definition)
		r.Counter[c.Definition.Name] = c
	}
	return &r
}
