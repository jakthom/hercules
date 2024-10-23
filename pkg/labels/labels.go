package labels

import (
	"maps"
	"os"
)

type GlobalLabels map[string]string

func InjectLabelFromEnv(labelVal string) string {
	if string(labelVal[0]) == "$" {
		return os.Getenv(string(labelVal[1:]))
	}
	return labelVal
}

func (l GlobalLabels) LabelNames() []string {
	var labelNames []string
	for k := range l {
		labelNames = append(labelNames, k)
	}
	return labelNames
}

func Merge(labels map[string]string, moreLabels map[string]string) map[string]string {
	var merged = make(map[string]string)
	maps.Copy(merged, labels)
	maps.Copy(merged, moreLabels)
	return merged
}
