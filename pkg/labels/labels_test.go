package labels

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestInjectLabelFromEnv(t *testing.T) {
	label := "$HERCULES"
	value := "notprometheus"
	os.Setenv("HERCULES", value)

	labelValue := InjectLabelFromEnv(label)
	assert.Equal(t, value, labelValue)
}

func TestLabelNames(t *testing.T) {
	labelNames := Labels{
		"cell":     "ausw1",
		"fromEnv":  "testing",
		"hercules": "fake",
	}.LabelNames()
	assert.Equal(t, 3, len(labelNames))
	assert.Equal(t, "cell", labelNames[0])
	assert.Equal(t, "fromEnv", labelNames[1])
	assert.Equal(t, "hercules", labelNames[2])
}

func TestMerge(t *testing.T) {
	someLabels := Labels{
		"cell":    "ausw1",
		"fromEnv": "testing",
	}
	moreLabels := Labels{
		"hercules": "testing",
	}

	want := Labels{
		"cell":     "ausw1",
		"fromEnv":  "testing",
		"hercules": "testing",
	}
	got := Merge(someLabels, moreLabels)
	assert.Equal(t, want, got)
}
