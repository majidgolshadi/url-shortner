package id

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

type rangeManagerMock struct {
	initRange Range
	jumpRange uint
}

func (mock *rangeManagerMock) getCurrentRange() (Range, error) {
	return mock.initRange, nil
}

func (mock *rangeManagerMock) getNextIDRange() (Range, error) {
	return Range{
		Min: mock.initRange.Min + mock.jumpRange,
		Max: mock.initRange.Max + mock.jumpRange,
	}, nil
}

func TestIDManagement(t *testing.T) {
	idMng, _ := NewManager(&rangeManagerMock{
		initRange: Range{
			Min: 2,
			Max: 3,
		},
		jumpRange: 100,
	})

	assert.Equal(t, uint(2), idMng.GetLastID())
	nextID, _ := idMng.GetNextID()
	assert.Equal(t, uint(3), nextID)
	nextID, _ = idMng.GetNextID()
	assert.Equal(t, uint(102), nextID)
	nextID, _ = idMng.GetNextID()
	assert.Equal(t, uint(103), nextID)
	assert.Equal(t, uint(103), idMng.GetLastID())
}
