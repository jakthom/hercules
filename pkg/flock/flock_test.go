// Package flock_test contains tests for the flock package
package flock_test

import (
	"testing"

	"github.com/jakthom/hercules/pkg/config"
	"github.com/jakthom/hercules/pkg/flock"
	"github.com/stretchr/testify/assert"
)

func TestInitializeDB(t *testing.T) {
	db, conn := flock.InitializeDB(config.Config{})
	assert.NotNil(t, db)
	assert.NotNil(t, conn)
}
