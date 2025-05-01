// Package herculestypes_test contains tests for the herculestypes package
package herculestypes_test

import (
	"testing"

	herculestypes "github.com/jakthom/hercules/pkg/types"
	"github.com/stretchr/testify/assert"
)

func TestPackageName(t *testing.T) {
	pkg := herculestypes.PackageName("test")
	assert.Equal(t, "test", string(pkg))
}
