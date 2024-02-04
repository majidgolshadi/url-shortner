package id

import (
	"context"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestGetIDRange(t *testing.T) {
	rangeMng := &inMemory{
		startID: 1,
	}

	rng, _ := rangeMng.getCurrentRange(context.Background())
	assert.Equal(t, uint(1), rng.Start)
	assert.Equal(t, uint(18446744073709551615), rng.End)
}
