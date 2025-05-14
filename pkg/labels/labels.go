package labels

import (
	"maps"
	"os"
)

type Labels map[string]string

func InjectLabelFromEnv(labelVal string) string {
	if labelVal[0] == '$' {
		return os.Getenv(labelVal[1:])
	}
	return labelVal
}

func (l Labels) LabelNames() []string {
	var labelNames []string
	for k := range l {
		labelNames = append(labelNames, k)
	}
	return labelNames
}

func Merge(labels Labels, moreLabels Labels) Labels {
	var merged = make(map[string]string)
	maps.Copy(merged, labels)
	maps.Copy(merged, moreLabels)
	return merged
}
