package id

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestGenerateIntegerID(t *testing.T) {
	const startID = 2
	idGen := &IntegerIdGenerator{
		id: startID,
	}

	assert.Equal(t, uint(startID), idGen.GetLastID())
	assert.Equal(t, uint(startID+1), idGen.NewID())
}
