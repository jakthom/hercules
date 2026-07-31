package exporter

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"math"
	"slices"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/jakthom/hercules/pkg/db"
	"github.com/jakthom/hercules/pkg/labels"
	"github.com/jakthom/hercules/pkg/metric"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/rs/zerolog/log"
)

const (
	DefaultMaxLabelValueLength = 1_024
	histogramFixedSeries       = 3 // +Inf, _sum, and _count.
	summaryFixedSeries         = 2 // _sum and _count.
)

type Snapshot struct {
	Metrics []prometheus.Metric
	Created time.Time
	Rows    int
	Series  int
}

type Collector struct {
	executor    db.Executor
	definitions []metric.TypedDefinition
	maxRows     int
	maxSeries   int
	maxLabelLen int

	refreshMu sync.Mutex
	snapshot  atomic.Pointer[Snapshot]

	statsMu          sync.RWMutex
	lastDuration     time.Duration
	lastError        string
	collectionErrors uint64

	collectionDurationDesc *prometheus.Desc
	collectionErrorsDesc   *prometheus.Desc
	lastSuccessDesc        *prometheus.Desc
	snapshotAgeDesc        *prometheus.Desc
	snapshotRowsDesc       *prometheus.Desc
	snapshotSeriesDesc     *prometheus.Desc
	readyDesc              *prometheus.Desc
}

type sample struct {
	labelNames  []string
	labelValues []string
	value       float64
}

type group struct {
	labelNames  []string
	labelValues []string
	values      []float64
}

type queryLayout struct {
	valueIndex   int
	labelIndexes []int
	labelNames   []string
	columnCount  int
}

func New(executor db.Executor, definitions []metric.TypedDefinition, maxRows int, maxSeries int) (*Collector, error) {
	if executor == nil {
		return nil, errors.New("exporter requires a DuckDB executor")
	}
	if maxRows < 1 || maxSeries < 1 {
		return nil, errors.New("exporter row and series limits must be positive")
	}
	collector := &Collector{
		executor:               executor,
		definitions:            cloneDefinitions(definitions),
		maxRows:                maxRows,
		maxSeries:              maxSeries,
		maxLabelLen:            DefaultMaxLabelValueLength,
		collectionDurationDesc: prometheus.NewDesc("hercules_collection_duration_seconds", "Duration of the most recent Hercules metric collection.", nil, nil),
		collectionErrorsDesc:   prometheus.NewDesc("hercules_collection_errors_total", "Total failed Hercules metric collections.", nil, nil),
		lastSuccessDesc:        prometheus.NewDesc("hercules_last_successful_collection_timestamp_seconds", "Unix timestamp of the most recent successful Hercules metric collection.", nil, nil),
		snapshotAgeDesc:        prometheus.NewDesc("hercules_snapshot_age_seconds", "Age of the current Hercules metric snapshot.", nil, nil),
		snapshotRowsDesc:       prometheus.NewDesc("hercules_snapshot_rows", "Rows read while producing the current Hercules metric snapshot.", nil, nil),
		snapshotSeriesDesc:     prometheus.NewDesc("hercules_snapshot_series", "Prometheus series represented by the current Hercules metric snapshot.", nil, nil),
		readyDesc:              prometheus.NewDesc("hercules_ready", "Whether Hercules has a successful metric snapshot.", nil, nil),
	}
	outputOwners := make(map[string]string)
	for _, typed := range collector.definitions {
		if typed.Definition == nil {
			return nil, fmt.Errorf("%s metric definition must not be null", typed.Type)
		}
		if err := typed.Definition.Validate(typed.Type); err != nil {
			return nil, err
		}
		for _, outputName := range typed.Definition.OutputNames(typed.Type) {
			if previous, exists := outputOwners[outputName]; exists {
				return nil, fmt.Errorf(
					"metric output %q is produced by both %q and %q",
					outputName,
					previous,
					typed.Definition.Name,
				)
			}
			outputOwners[outputName] = typed.Definition.Name
		}
	}
	return collector, nil
}

func cloneDefinitions(definitions []metric.TypedDefinition) []metric.TypedDefinition {
	cloned := make([]metric.TypedDefinition, len(definitions))
	for index, typed := range definitions {
		cloned[index].Type = typed.Type
		if typed.Definition == nil {
			continue
		}
		definition := *typed.Definition
		if definition.Enabled != nil {
			enabled := *definition.Enabled
			definition.Enabled = &enabled
		}
		definition.Labels = append([]string(nil), definition.Labels...)
		definition.Buckets = append([]float64(nil), definition.Buckets...)
		definition.Objectives = append([]metric.SummaryObjective(nil), definition.Objectives...)
		definition.Metadata.Labels = maps.Clone(definition.Metadata.Labels)
		cloned[index].Definition = &definition
	}
	return cloned
}

// Describe intentionally emits no descriptors, making this an unchecked collector.
// Metric descriptors are validated while each immutable snapshot is built.
func (c *Collector) Describe(_ chan<- *prometheus.Desc) {}

func (c *Collector) Collect(ch chan<- prometheus.Metric) {
	snapshot := c.snapshot.Load()
	for _, configuredMetric := range snapshotMetrics(snapshot) {
		ch <- configuredMetric
	}

	c.statsMu.RLock()
	duration := c.lastDuration.Seconds()
	errorsTotal := c.collectionErrors
	c.statsMu.RUnlock()

	ready := 0.0
	lastSuccess := 0.0
	age := 0.0
	rows := 0.0
	series := 0.0
	if snapshot != nil {
		ready = 1
		lastSuccess = float64(snapshot.Created.Unix())
		age = time.Since(snapshot.Created).Seconds()
		rows = float64(snapshot.Rows)
		series = float64(snapshot.Series)
	}
	ch <- prometheus.MustNewConstMetric(c.collectionDurationDesc, prometheus.GaugeValue, duration)
	ch <- prometheus.MustNewConstMetric(c.collectionErrorsDesc, prometheus.CounterValue, float64(errorsTotal))
	ch <- prometheus.MustNewConstMetric(c.lastSuccessDesc, prometheus.GaugeValue, lastSuccess)
	ch <- prometheus.MustNewConstMetric(c.snapshotAgeDesc, prometheus.GaugeValue, age)
	ch <- prometheus.MustNewConstMetric(c.snapshotRowsDesc, prometheus.GaugeValue, rows)
	ch <- prometheus.MustNewConstMetric(c.snapshotSeriesDesc, prometheus.GaugeValue, series)
	ch <- prometheus.MustNewConstMetric(c.readyDesc, prometheus.GaugeValue, ready)
}

func (c *Collector) Refresh(ctx context.Context) error {
	c.refreshMu.Lock()
	defer c.refreshMu.Unlock()
	started := time.Now()
	snapshot, err := c.evaluate(ctx)
	duration := time.Since(started)
	c.statsMu.Lock()
	c.lastDuration = duration
	if err != nil {
		c.collectionErrors++
		c.lastError = err.Error()
	} else {
		c.lastError = ""
	}
	c.statsMu.Unlock()
	if err != nil {
		return err
	}
	c.snapshot.Store(snapshot)
	return nil
}

func (c *Collector) Run(ctx context.Context, interval time.Duration, timeout time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			collectionContext, cancel := context.WithTimeout(ctx, timeout)
			err := c.Refresh(collectionContext)
			cancel()
			if err != nil {
				log.Error().Err(err).Msg("metric snapshot collection failed; retaining last-known-good snapshot")
			}
		}
	}
}

func (c *Collector) Ready() bool {
	return c.snapshot.Load() != nil
}

func (c *Collector) LastError() string {
	c.statsMu.RLock()
	defer c.statsMu.RUnlock()
	return c.lastError
}

func (c *Collector) evaluate(ctx context.Context) (*Snapshot, error) {
	result := &Snapshot{}
	for _, typed := range c.definitions {
		definition := typed.Definition
		if !definition.IsEnabled() {
			continue
		}
		table, err := c.executor.Query(ctx, definition.SQL, c.maxRows)
		if err != nil {
			return nil, fmt.Errorf("query metric %q: %w", definition.FullName(), err)
		}
		result.Rows += len(table.Rows)
		metrics, series, err := c.buildMetric(typed, table, c.maxSeries-result.Series)
		if err != nil {
			return nil, fmt.Errorf("build metric %q: %w", definition.FullName(), err)
		}
		result.Series += series
		result.Metrics = append(result.Metrics, metrics...)
	}
	result.Created = time.Now()
	return result, nil
}

func (c *Collector) buildMetric(
	typed metric.TypedDefinition,
	table db.Table,
	remainingSeries int,
) ([]prometheus.Metric, int, error) {
	definition := typed.Definition
	configuredSamples, err := c.samples(definition, table)
	if err != nil {
		return nil, 0, err
	}

	switch typed.Type {
	case metric.GaugeType:
		series, err := c.seriesWithinBudget(len(configuredSamples), 1, remainingSeries)
		if err != nil {
			return nil, 0, err
		}
		metrics, err := buildGauges(definition, configuredSamples)
		return metrics, series, err
	case metric.CounterType:
		groups := groupSamples(configuredSamples)
		series, err := c.seriesWithinBudget(len(groups), 1, remainingSeries)
		if err != nil {
			return nil, 0, err
		}
		metrics, err := buildCounters(definition, groups)
		return metrics, series, err
	case metric.HistogramType:
		groups := groupSamples(configuredSamples)
		series, err := c.seriesWithinBudget(
			len(groups),
			len(definition.Buckets)+histogramFixedSeries,
			remainingSeries,
		)
		if err != nil {
			return nil, 0, err
		}
		metrics, err := buildHistograms(definition, groups)
		return metrics, series, err
	case metric.SummaryType:
		groups := groupSamples(configuredSamples)
		series, err := c.seriesWithinBudget(
			len(groups),
			len(definition.Objectives)+summaryFixedSeries,
			remainingSeries,
		)
		if err != nil {
			return nil, 0, err
		}
		metrics, err := buildSummaries(definition, groups)
		return metrics, series, err
	default:
		return nil, 0, fmt.Errorf("unsupported metric type %q", typed.Type)
	}
}

func (c *Collector) seriesWithinBudget(groupCount int, seriesPerGroup int, remaining int) (int, error) {
	if groupCount > remaining/seriesPerGroup {
		return 0, fmt.Errorf("snapshot exceeds maximum of %d series", c.maxSeries)
	}
	return groupCount * seriesPerGroup, nil
}

func buildGauges(definition *metric.Definition, samples []sample) ([]prometheus.Metric, error) {
	seen := make(map[string]struct{}, len(samples))
	result := make([]prometheus.Metric, 0, len(samples))
	for _, configuredSample := range samples {
		key := labelKey(configuredSample.labelValues)
		if _, exists := seen[key]; exists {
			return nil, fmt.Errorf("duplicate gauge label set %v", configuredSample.labelValues)
		}
		seen[key] = struct{}{}
		createdMetric, err := prometheus.NewConstMetric(
			metricDescription(definition, configuredSample.labelNames),
			prometheus.GaugeValue,
			configuredSample.value,
			configuredSample.labelValues...,
		)
		if err != nil {
			return nil, err
		}
		result = append(result, createdMetric)
	}
	return result, nil
}

func buildCounters(definition *metric.Definition, groups map[string]*group) ([]prometheus.Metric, error) {
	result := make([]prometheus.Metric, 0, len(groups))
	for _, grouped := range groups {
		value, err := counterValue(grouped.values)
		if err != nil {
			return nil, err
		}
		createdMetric, err := prometheus.NewConstMetric(
			metricDescription(definition, grouped.labelNames),
			prometheus.CounterValue,
			value,
			grouped.labelValues...,
		)
		if err != nil {
			return nil, err
		}
		result = append(result, createdMetric)
	}
	return result, nil
}

func counterValue(values []float64) (float64, error) {
	result := 0.0
	for _, value := range values {
		if value < 0 {
			return 0, errors.New("counter value must not be negative")
		}
		result += value
		if !isFinite(result) {
			return 0, errors.New("counter sum must be finite")
		}
	}
	return result, nil
}

func buildHistograms(definition *metric.Definition, groups map[string]*group) ([]prometheus.Metric, error) {
	result := make([]prometheus.Metric, 0, len(groups))
	for _, grouped := range groups {
		buckets, sum, err := histogramValues(grouped.values, definition.Buckets)
		if err != nil {
			return nil, err
		}
		createdMetric, err := prometheus.NewConstHistogram(
			metricDescription(definition, grouped.labelNames),
			uint64(len(grouped.values)),
			sum,
			buckets,
			grouped.labelValues...,
		)
		if err != nil {
			return nil, err
		}
		result = append(result, createdMetric)
	}
	return result, nil
}

func histogramValues(values []float64, bounds []float64) (map[float64]uint64, float64, error) {
	buckets := make(map[float64]uint64, len(bounds))
	sum := 0.0
	for _, observation := range values {
		sum += observation
		if !isFinite(sum) {
			return nil, 0, errors.New("histogram sum must be finite")
		}
		for _, upperBound := range bounds {
			if observation <= upperBound {
				buckets[upperBound]++
			}
		}
	}
	return buckets, sum, nil
}

func buildSummaries(definition *metric.Definition, groups map[string]*group) ([]prometheus.Metric, error) {
	result := make([]prometheus.Metric, 0, len(groups))
	for _, grouped := range groups {
		quantiles, sum, err := summaryValues(grouped.values, definition.Objectives)
		if err != nil {
			return nil, err
		}
		createdMetric, err := prometheus.NewConstSummary(
			metricDescription(definition, grouped.labelNames),
			uint64(len(grouped.values)),
			sum,
			quantiles,
			grouped.labelValues...,
		)
		if err != nil {
			return nil, err
		}
		result = append(result, createdMetric)
	}
	return result, nil
}

func summaryValues(
	values []float64,
	objectives []metric.SummaryObjective,
) (map[float64]float64, float64, error) {
	slices.Sort(values)
	quantiles := make(map[float64]float64, len(objectives))
	sum := 0.0
	for _, observation := range values {
		sum += observation
		if !isFinite(sum) {
			return nil, 0, errors.New("summary sum must be finite")
		}
	}
	for _, objective := range objectives {
		value := quantile(values, objective.Quantile)
		if !isFinite(value) {
			return nil, 0, fmt.Errorf("summary quantile %v must be finite", objective.Quantile)
		}
		quantiles[objective.Quantile] = value
	}
	return quantiles, sum, nil
}

func metricDescription(definition *metric.Definition, labelNames []string) *prometheus.Desc {
	return prometheus.NewDesc(definition.FullName(), definition.Help, labelNames, nil)
}

func (c *Collector) samples(definition *metric.Definition, table db.Table) ([]sample, error) {
	layout, err := resultLayout(definition, table.Columns)
	if err != nil {
		return nil, err
	}
	globalNames := definition.Metadata.Labels.LabelNames()
	for _, name := range globalNames {
		if slices.Contains(layout.labelNames, name) {
			return nil, fmt.Errorf("query label %q collides with a global label", name)
		}
	}
	allNames := append(append([]string(nil), layout.labelNames...), globalNames...)
	slices.Sort(allNames)
	result := make([]sample, 0, len(table.Rows))
	for rowIndex, row := range table.Rows {
		configuredSample, conversionErr := c.sampleFromRow(definition, layout, allNames, rowIndex, row)
		if conversionErr != nil {
			return nil, conversionErr
		}
		result = append(result, configuredSample)
	}
	return result, nil
}

func (c *Collector) sampleFromRow(
	definition *metric.Definition,
	layout queryLayout,
	allNames []string,
	rowIndex int,
	row []any,
) (sample, error) {
	if len(row) != layout.columnCount {
		return sample{}, fmt.Errorf(
			"row %d has %d values for %d columns",
			rowIndex,
			len(row),
			layout.columnCount,
		)
	}
	value, err := db.ToFloat64(row[layout.valueIndex])
	if err != nil {
		return sample{}, fmt.Errorf("row %d value: %w", rowIndex, err)
	}
	valuesByName := make(labels.Labels, len(allNames))
	for index, columnIndex := range layout.labelIndexes {
		labelName := layout.labelNames[index]
		labelValue, conversionErr := db.ToLabelValue(row[columnIndex])
		if conversionErr != nil {
			return sample{}, fmt.Errorf("row %d label %q: %w", rowIndex, labelName, conversionErr)
		}
		if len(labelValue) > c.maxLabelLen {
			return sample{}, fmt.Errorf("row %d label %q exceeds maximum length %d", rowIndex, labelName, c.maxLabelLen)
		}
		valuesByName[labelName] = labelValue
	}
	for name, labelValue := range definition.Metadata.Labels {
		if len(labelValue) > c.maxLabelLen {
			return sample{}, fmt.Errorf("global label %q exceeds maximum length %d", name, c.maxLabelLen)
		}
		valuesByName[name] = labelValue
	}
	labelValues := make([]string, len(allNames))
	for index, name := range allNames {
		labelValues[index] = valuesByName[name]
	}
	return sample{labelNames: allNames, labelValues: labelValues, value: value}, nil
}

func resultLayout(definition *metric.Definition, columns []string) (queryLayout, error) {
	if len(columns) == 0 {
		return queryLayout{}, errors.New("query returned no columns")
	}
	columnIndexes, err := indexColumns(columns)
	if err != nil {
		return queryLayout{}, err
	}
	valueIndex, err := findValueColumn(definition, columns, columnIndexes)
	if err != nil {
		return queryLayout{}, err
	}
	labelNames, labelIndexes, err := findLabelColumns(definition, columns, columnIndexes, valueIndex)
	if err != nil {
		return queryLayout{}, err
	}
	return queryLayout{
		valueIndex:   valueIndex,
		labelIndexes: labelIndexes,
		labelNames:   labelNames,
		columnCount:  len(columns),
	}, nil
}

func indexColumns(columns []string) (map[string]int, error) {
	columnIndexes := make(map[string]int, len(columns))
	for index, name := range columns {
		if _, exists := columnIndexes[name]; exists {
			return nil, fmt.Errorf("query repeats column %q", name)
		}
		columnIndexes[name] = index
	}
	return columnIndexes, nil
}

func findValueColumn(
	definition *metric.Definition,
	columns []string,
	columnIndexes map[string]int,
) (int, error) {
	valueIndex := -1
	for _, candidate := range []string{"value", "val", "v"} {
		if index, exists := columnIndexes[candidate]; exists {
			if valueIndex != -1 {
				return -1, errors.New("query has multiple value columns")
			}
			valueIndex = index
		}
	}
	if valueIndex != -1 {
		return valueIndex, nil
	}
	if len(definition.Labels) > 0 {
		return nonLabelValueColumn(definition.Labels, columns)
	}
	if len(columns) == 1 {
		return 0, nil
	}
	return functionValueColumn(columns)
}

func nonLabelValueColumn(labelNames []string, columns []string) (int, error) {
	labelSet := make(map[string]struct{}, len(labelNames))
	for _, name := range labelNames {
		labelSet[name] = struct{}{}
	}
	valueIndex := -1
	for index, name := range columns {
		if _, isLabel := labelSet[name]; isLabel {
			continue
		}
		if valueIndex != -1 {
			return -1, errors.New("query must return exactly one non-label value column")
		}
		valueIndex = index
	}
	if valueIndex == -1 {
		return -1, errors.New("query must return exactly one non-label value column")
	}
	return valueIndex, nil
}

func functionValueColumn(columns []string) (int, error) {
	valueIndex := -1
	for index, name := range columns {
		if !strings.Contains(name, "(") || !strings.Contains(name, ")") {
			continue
		}
		if valueIndex != -1 {
			return -1, errors.New("query has multiple function-like value columns")
		}
		valueIndex = index
	}
	if valueIndex == -1 {
		return -1, errors.New("query must return a numeric column named value")
	}
	return valueIndex, nil
}

func findLabelColumns(
	definition *metric.Definition,
	columns []string,
	columnIndexes map[string]int,
	valueIndex int,
) ([]string, []int, error) {
	labelNames := append([]string(nil), definition.Labels...)
	if len(labelNames) == 0 {
		for index, name := range columns {
			if index != valueIndex {
				labelNames = append(labelNames, name)
			}
		}
	}
	if err := labels.FromNames(labelNames).ValidateNames(); err != nil {
		return nil, nil, err
	}
	labelIndexes := make([]int, len(labelNames))
	for labelPosition, name := range labelNames {
		columnIndex, exists := columnIndexes[name]
		if !exists {
			return nil, nil, fmt.Errorf("query does not return declared label %q", name)
		}
		if columnIndex == valueIndex {
			return nil, nil, fmt.Errorf("label %q is also the value column", name)
		}
		labelIndexes[labelPosition] = columnIndex
	}
	if len(labelNames)+1 != len(columns) {
		return nil, nil, errors.New("query returns undeclared columns")
	}
	return labelNames, labelIndexes, nil
}

func groupSamples(samples []sample) map[string]*group {
	result := make(map[string]*group)
	for _, configuredSample := range samples {
		key := labelKey(configuredSample.labelValues)
		grouped, exists := result[key]
		if !exists {
			grouped = &group{
				labelNames:  append([]string(nil), configuredSample.labelNames...),
				labelValues: append([]string(nil), configuredSample.labelValues...),
			}
			result[key] = grouped
		}
		grouped.values = append(grouped.values, configuredSample.value)
	}
	return result
}

func labelKey(values []string) string {
	var builder strings.Builder
	for _, value := range values {
		builder.WriteString(strconv.Itoa(len(value)))
		builder.WriteByte(':')
		builder.WriteString(value)
		builder.WriteByte('|')
	}
	return builder.String()
}

func quantile(values []float64, target float64) float64 {
	if len(values) == 0 {
		return math.NaN()
	}
	position := target * float64(len(values)-1)
	lower := int(math.Floor(position))
	upper := int(math.Ceil(position))
	if lower == upper {
		return values[lower]
	}
	fraction := position - float64(lower)
	return (1-fraction)*values[lower] + fraction*values[upper]
}

func isFinite(value float64) bool {
	return !math.IsNaN(value) && !math.IsInf(value, 0)
}

func snapshotMetrics(snapshot *Snapshot) []prometheus.Metric {
	if snapshot == nil {
		return nil
	}
	return snapshot.Metrics
}
