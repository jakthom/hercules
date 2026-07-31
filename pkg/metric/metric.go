package metric

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"regexp"
	"slices"
	"strings"

	"github.com/jakthom/hercules/pkg/db"
)

type Type string

const (
	GaugeType     Type = "gauge"
	CounterType   Type = "counter"
	SummaryType   Type = "summary"
	HistogramType Type = "histogram"

	HistogramObservationsMode = "observations"
	MaxLabelsPerMetric        = 20
	MaxHistogramBuckets       = 100
	MaxSummaryObjectives      = 20
)

var (
	metricNamePattern = regexp.MustCompile(`^[a-zA-Z_:][a-zA-Z0-9_:]*$`)
	labelNamePattern  = regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_]*$`)
)

type SummaryObjective struct {
	Quantile float64 `json:"quantile"`
	Error    float64 `json:"error"`
}

func (o *SummaryObjective) UnmarshalJSON(payload []byte) error {
	var quantile float64
	if err := json.Unmarshal(payload, &quantile); err == nil {
		o.Quantile = quantile
		switch {
		case quantile >= 0.99:
			o.Error = 0.001
		case quantile >= 0.9:
			o.Error = 0.01
		default:
			o.Error = 0.05
		}
		return nil
	}
	type objective SummaryObjective
	var decoded objective
	if err := json.Unmarshal(payload, &decoded); err != nil {
		return fmt.Errorf("summary objective must be a number or quantile/error object: %w", err)
	}
	*o = SummaryObjective(decoded)
	return nil
}

type Definition struct {
	Name          string             `json:"name"`
	Enabled       *bool              `json:"enabled,omitempty"`
	Help          string             `json:"help"`
	SQL           db.SQL             `json:"sql"`
	Labels        []string           `json:"labels,omitempty"`
	Buckets       []float64          `json:"buckets,omitempty"`
	Objectives    []SummaryObjective `json:"objectives,omitempty"`
	HistogramMode string             `json:"histogramMode,omitempty"`
	Metadata      Metadata           `json:"-"`
}

type TypedDefinition struct {
	Type       Type
	Definition *Definition
}

type Definitions struct {
	Gauge     []*Definition `json:"gauge,omitempty"`
	Counter   []*Definition `json:"counter,omitempty"`
	Summary   []*Definition `json:"summary,omitempty"`
	Histogram []*Definition `json:"histogram,omitempty"`
}

func (m Definition) IsEnabled() bool {
	return m.Enabled == nil || *m.Enabled
}

func (m Definition) FullName() string {
	packageName := strings.ReplaceAll(m.Metadata.PackageName, "-", "_")
	return string(m.Metadata.Prefix) + packageName + "_" + m.Name
}

func (m Definition) LabelNames() []string {
	names := append([]string(nil), m.Labels...)
	names = append(names, m.Metadata.Labels.LabelNames()...)
	slices.Sort(names)
	return names
}

func (m Definition) Validate(metricType Type) error {
	err := errors.Join(m.validateBase(), m.validateTypeName(metricType))
	switch metricType {
	case GaugeType, CounterType:
		return err
	case HistogramType:
		return errors.Join(err, m.validateHistogram())
	case SummaryType:
		return errors.Join(err, m.validateSummary())
	default:
		return errors.Join(err, fmt.Errorf("metric %q has unsupported type %q", m.Name, metricType))
	}
}

func (m Definition) validateBase() error {
	var errs []error
	if !metricNamePattern.MatchString(m.Name) {
		errs = append(errs, fmt.Errorf("invalid metric name %q", m.Name))
	}
	if strings.TrimSpace(m.Help) == "" {
		errs = append(errs, fmt.Errorf("metric %q help must not be empty", m.Name))
	}
	if strings.TrimSpace(string(m.SQL)) == "" {
		errs = append(errs, fmt.Errorf("metric %q sql must not be empty", m.Name))
	}
	if len(m.Labels) > MaxLabelsPerMetric {
		errs = append(errs, fmt.Errorf("metric %q exceeds maximum of %d labels", m.Name, MaxLabelsPerMetric))
	}
	seenLabels := make(map[string]struct{}, len(m.Labels))
	for _, name := range m.Labels {
		if !labelNamePattern.MatchString(name) {
			errs = append(errs, fmt.Errorf("metric %q has invalid label name %q", m.Name, name))
		}
		if _, exists := seenLabels[name]; exists {
			errs = append(errs, fmt.Errorf("metric %q repeats label %q", m.Name, name))
		}
		if _, exists := m.Metadata.Labels[name]; exists {
			errs = append(errs, fmt.Errorf("metric %q label %q collides with a global label", m.Name, name))
		}
		seenLabels[name] = struct{}{}
	}
	if !metricNamePattern.MatchString(m.FullName()) {
		errs = append(errs, fmt.Errorf("invalid full metric name %q", m.FullName()))
	}
	for _, prefix := range []string{"hercules_", "go_", "process_"} {
		if strings.HasPrefix(m.FullName(), prefix) {
			errs = append(errs, fmt.Errorf("metric name %q uses reserved prefix %q", m.FullName(), prefix))
		}
	}
	return errors.Join(errs...)
}

func (m Definition) validateTypeName(metricType Type) error {
	if metricType == CounterType && !strings.HasSuffix(m.Name, "_total") {
		return fmt.Errorf("counter metric %q must end in _total", m.Name)
	}
	if metricType == HistogramType || metricType == SummaryType {
		for _, suffix := range []string{"_bucket", "_sum", "_count", "_total"} {
			if strings.HasSuffix(m.Name, suffix) {
				return fmt.Errorf("%s metric %q must not use reserved suffix %q", metricType, m.Name, suffix)
			}
		}
	}
	return nil
}

func (m Definition) validateHistogram() error {
	var errs []error
	if m.HistogramMode != "" && m.HistogramMode != HistogramObservationsMode {
		errs = append(errs, fmt.Errorf("metric %q has unsupported histogramMode %q", m.Name, m.HistogramMode))
	}
	if len(m.Buckets) == 0 {
		errs = append(errs, fmt.Errorf("histogram %q must define buckets", m.Name))
	}
	if len(m.Buckets) > MaxHistogramBuckets {
		errs = append(errs, fmt.Errorf("histogram %q exceeds maximum of %d buckets", m.Name, MaxHistogramBuckets))
	}
	for index, bucket := range m.Buckets {
		if math.IsNaN(bucket) || math.IsInf(bucket, 0) {
			errs = append(errs, fmt.Errorf("histogram %q bucket %d must be finite", m.Name, index))
		}
		if index > 0 && bucket <= m.Buckets[index-1] {
			errs = append(errs, fmt.Errorf("histogram %q buckets must be strictly increasing", m.Name))
			break
		}
	}
	return errors.Join(errs...)
}

func (m Definition) validateSummary() error {
	var errs []error
	if len(m.Objectives) == 0 {
		errs = append(errs, fmt.Errorf("summary %q must define objectives", m.Name))
	}
	if len(m.Objectives) > MaxSummaryObjectives {
		errs = append(errs, fmt.Errorf("summary %q exceeds maximum of %d objectives", m.Name, MaxSummaryObjectives))
	}
	seenQuantiles := make(map[float64]struct{}, len(m.Objectives))
	for _, objective := range m.Objectives {
		if math.IsNaN(objective.Quantile) || math.IsInf(objective.Quantile, 0) ||
			objective.Quantile <= 0 || objective.Quantile >= 1 {
			errs = append(errs, fmt.Errorf("summary %q quantile must be finite and between 0 and 1", m.Name))
		}
		if math.IsNaN(objective.Error) || math.IsInf(objective.Error, 0) ||
			objective.Error <= 0 || objective.Error >= 1 {
			errs = append(errs, fmt.Errorf("summary %q error must be finite and between 0 and 1", m.Name))
		}
		if _, exists := seenQuantiles[objective.Quantile]; exists {
			errs = append(errs, fmt.Errorf("summary %q repeats quantile %v", m.Name, objective.Quantile))
		}
		seenQuantiles[objective.Quantile] = struct{}{}
	}
	return errors.Join(errs...)
}

func (m Definition) OutputNames(metricType Type) []string {
	fullName := m.FullName()
	switch metricType {
	case GaugeType, CounterType:
		return []string{fullName}
	case HistogramType:
		return []string{fullName, fullName + "_bucket", fullName + "_sum", fullName + "_count"}
	case SummaryType:
		return []string{fullName, fullName + "_sum", fullName + "_count"}
	default:
		return []string{fullName}
	}
}

func (m Definitions) All() []TypedDefinition {
	result := make([]TypedDefinition, 0, len(m.Gauge)+len(m.Counter)+len(m.Summary)+len(m.Histogram))
	for _, definition := range m.Gauge {
		result = append(result, TypedDefinition{Type: GaugeType, Definition: definition})
	}
	for _, definition := range m.Counter {
		result = append(result, TypedDefinition{Type: CounterType, Definition: definition})
	}
	for _, definition := range m.Histogram {
		result = append(result, TypedDefinition{Type: HistogramType, Definition: definition})
	}
	for _, definition := range m.Summary {
		result = append(result, TypedDefinition{Type: SummaryType, Definition: definition})
	}
	return result
}

func (m Definitions) Validate() error {
	seen := make(map[string]Type)
	outputOwners := make(map[string]string)
	var errs []error
	for _, typed := range m.All() {
		if typed.Definition == nil {
			errs = append(errs, fmt.Errorf("%s metric definition must not be null", typed.Type))
			continue
		}
		if previous, exists := seen[typed.Definition.Name]; exists {
			errs = append(errs, fmt.Errorf("metric %q is declared as both %s and %s", typed.Definition.Name, previous, typed.Type))
		}
		seen[typed.Definition.Name] = typed.Type
		if err := typed.Definition.Validate(typed.Type); err != nil {
			errs = append(errs, err)
		}
		for _, outputName := range typed.Definition.OutputNames(typed.Type) {
			if previous, exists := outputOwners[outputName]; exists {
				errs = append(
					errs,
					fmt.Errorf(
						"metric output %q is produced by both %q and %q",
						outputName,
						previous,
						typed.Definition.Name,
					),
				)
			}
			outputOwners[outputName] = typed.Definition.Name
		}
	}
	return errors.Join(errs...)
}

func (m Definitions) InjectMetadata(metadata Metadata) {
	for _, typed := range m.All() {
		if typed.Definition != nil {
			typed.Definition.Metadata = metadata
		}
	}
}

func (m Definitions) ExpandVariables(expand func(string) (string, error)) error {
	for _, typed := range m.All() {
		if typed.Definition == nil {
			continue
		}
		for target, value := range map[*string]string{
			&typed.Definition.Name: typed.Definition.Name,
			&typed.Definition.Help: typed.Definition.Help,
		} {
			expanded, err := expand(value)
			if err != nil {
				return err
			}
			*target = expanded
		}
		expandedSQL, err := expand(string(typed.Definition.SQL))
		if err != nil {
			return err
		}
		typed.Definition.SQL = db.SQL(expandedSQL)
	}
	return nil
}

func (m Definitions) OutputNames(metadata Metadata) []string {
	result := make([]string, 0, len(m.All()))
	for _, typed := range m.All() {
		if typed.Definition == nil {
			continue
		}
		definitionCopy := *typed.Definition
		definitionCopy.Metadata = metadata
		result = append(result, definitionCopy.OutputNames(typed.Type)...)
	}
	return result
}
