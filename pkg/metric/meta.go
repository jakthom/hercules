package metric

import (
	"github.com/jakthom/hercules/pkg/labels"
	herculestypes "github.com/jakthom/hercules/pkg/types"
)

type Metadata struct {
	PackageName string                     `json:"packageName"`
	Prefix      herculestypes.MetricPrefix `json:"metricPrefix"`
	Labels      labels.Labels              `json:"labels"`
}
