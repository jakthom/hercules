// Package labels_test contains tests for the labels package
package labels_test

import (
	"testing"

	"github.com/jakthom/hercules/pkg/labels"
	"github.com/stretchr/testify/assert"
)

func TestInjectLabelFromEnv(t *testing.T) {
	label := "$HERCULES"
	value := "notprometheus"
	t.Setenv("HERCULES", value)

	labelValue := labels.InjectLabelFromEnv(label)
	assert.Equal(t, value, labelValue)
}

// func TestLabelNames(t *testing.T) {
// 	labelNames := labels.Labels{
// 		"cell":     "ausw1",
// 		"fromEnv":  "testing",
// 		"hercules": "fake",
// 	}.LabelNames()
// 	assert.Equal(t, 3, len(labelNames))
// 	assert.Equal(t, "cell", labelNames[0])
// 	assert.Equal(t, "fromEnv", labelNames[1])
// 	assert.Equal(t, "hercules", labelNames[2])
// }

func TestMerge(t *testing.T) {
	someLabels := labels.Labels{
		"cell":    "ausw1",
		"fromEnv": "testing",
	}
	moreLabels := labels.Labels{
		"hercules": "testing",
	}

	want := labels.Labels{
		"cell":     "ausw1",
		"fromEnv":  "testing",
		"hercules": "testing",
	}
	got := labels.Merge(someLabels, moreLabels)
	assert.Equal(t, want, got)
}
