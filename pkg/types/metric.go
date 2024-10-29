package herculestypes

import "github.com/jakthom/hercules/pkg/labels"

type MetricPrefix string

type MetricMetadata struct {
	PackageName
	MetricPrefix
	Labels labels.Labels
}
