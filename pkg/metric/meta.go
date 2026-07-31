package metric

import "github.com/jakthom/hercules/pkg/labels"

type Prefix string

type Metadata struct {
	PackageName string        `json:"packageName"`
	Prefix      Prefix        `json:"metricPrefix"`
	Labels      labels.Labels `json:"labels"`
}
