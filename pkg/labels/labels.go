package labels

import "maps"

type GlobalLabels map[string]string

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
