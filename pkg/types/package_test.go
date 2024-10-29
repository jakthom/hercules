package herculestypes

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPackageName(t *testing.T) {
	pkg := PackageName("test")
	assert.Equal(t, "test", string(pkg))
}
