package herculestypes

import (
	"strings"

	"github.com/jakthom/hercules/pkg/labels"
)

type MetricPrefix string

type MetricMetadata struct {
	PackageName
	MetricPrefix
	Labels labels.Labels
}

func (m *MetricMetadata) Prefix() string {
	// TODO -> Further sanitization to handle special characters in the package name.
	// KISS for now.
	return string(m.MetricPrefix) + string(strings.ReplaceAll(string(m.PackageName), "-", "_")) + "_"
}
