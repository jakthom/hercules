package herculestypes

import (
	"testing"

	"github.com/dbecorp/hercules/pkg/labels"
	"github.com/stretchr/testify/assert"
)

func TestMetricPrefix(t *testing.T) {
	pkg := MetricPrefix("test")
	assert.Equal(t, "test", string(pkg))
}

func TestMetricMetadata(t *testing.T) {
	packageName := PackageName("test")
	metricPrefix := MetricPrefix("herc_")
	pkg := MetricMetadata{
		PackageName:  packageName,
		MetricPrefix: metricPrefix,
		Labels:       labels.Labels{},
	}
	assert.Equal(t, packageName, pkg.PackageName)
	assert.Equal(t, metricPrefix, pkg.MetricPrefix)
	assert.Equal(t, labels.Labels{}, pkg.Labels)
}
