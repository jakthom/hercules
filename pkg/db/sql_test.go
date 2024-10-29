package db

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSql(t *testing.T) {
	sql := Sql("test")
	assert.Equal(t, "test", string(sql))
}
