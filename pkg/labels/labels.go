package labels

type Labels map[string]string

func (l Labels) LabelNames() []string {
	var labelNames []string
	for k := range l {
		labelNames = append(labelNames, k)
	}
	return labelNames
}
