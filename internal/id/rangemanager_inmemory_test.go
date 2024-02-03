package id

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestGetIDRange(t *testing.T) {
	rangeMng := &inMemory{
		startID: 1,
	}
	rng, _ := rangeMng.getCurrentRange()
	assert.Equal(t, uint(1), rng.Min)
	assert.Equal(t, uint(18446744073709551615), rng.Max)
}
