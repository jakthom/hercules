package metrics

import (
	"database/sql"

	"github.com/dbecorp/hercules/pkg/db"
	"github.com/dbecorp/hercules/pkg/labels"
	"github.com/rs/zerolog/log"
)

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
	if err != nil {
		return nil, err
	}
	var results []QueryResult
	for rows.Next() {
		result := QueryResult{}
		if md.Labels == nil { // Scalar results
			if err := rows.Scan(&result.Value); err != nil {
				log.Error().Err(err).Msg("error when scanning query results")
				return nil, err
			}
		} else {
			if err := rows.Scan(&result.Labels, &result.Value); err != nil {
				log.Error().Err(err).Msg("error when scanning query results")
				return nil, err
			}
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

func (mr *MetricRegistry) MaterializeWithConnection(conn *sql.Conn) error { // TODO -> Make this return a list of "materialization errors" if something fails
	for _, gauge := range mr.Gauge {
		err := gauge.materializeWithConnection(conn)
		if err != nil {
			log.Error().Err(err)
		}
	}

	for _, histogram := range mr.Histogram {
		err := histogram.materializeWithConnection(conn)
		if err != nil {
			log.Error().Err(err)
		}
	}

	for _, summary := range mr.Summary {
		err := summary.materializeWithConnection(conn)
		if err != nil {
			log.Error().Err(err)
		}
	}

	for _, counter := range mr.Counter {
		err := counter.materializeWithConnection(conn)
		if err != nil {
			log.Error().Err(err)
		}
	}
	return nil
}

func NewMetricRegistry(definitions MetricDefinitions, labels labels.GlobalLabels) *MetricRegistry {
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
		h := NewHistogramMetric(definition, labels)
		r.Histogram[h.Definition.Name] = h
	}
	for _, definition := range definitions.Summary {
		s := NewSummaryMetric(definition, labels)
		r.Summary[s.Definition.Name] = s
	}
	for _, definition := range definitions.Counter {
		c := NewCounterMetric(definition, labels)
		r.Counter[c.Definition.Name] = c
	}
	return &r
}
