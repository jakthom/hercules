package herculestypes

import "github.com/dbecorp/hercules/pkg/labels"

type MetricPrefix string

type MetricMetadata struct {
	PackageName
	MetricPrefix
	Labels labels.Labels
}
