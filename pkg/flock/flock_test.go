package flock

import (
	"testing"

	"github.com/jakthom/hercules/pkg/config"
	"github.com/stretchr/testify/assert"
)

func TestInitializeDB(t *testing.T) {
	db, conn := InitializeDB(config.Config{})
	assert.NotNil(t, db)
	assert.NotNil(t, conn)
}
