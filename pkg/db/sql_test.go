// Package db_test contains tests for the db package
package db_test

import (
	"testing"

	"github.com/jakthom/hercules/pkg/db"
	"github.com/stretchr/testify/assert"
)

func TestSql(t *testing.T) {
	sql := db.SQL("test")
	assert.Equal(t, "test", string(sql))
}
