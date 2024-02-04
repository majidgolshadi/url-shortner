package id

import (
	"context"
	"github.com/stretchr/testify/assert"
	"testing"
)

type rangeManagerMock struct {
	initRange Range
	jumpRange uint
}

func (mock *rangeManagerMock) getCurrentRange(ctx context.Context) (Range, error) {
	return mock.initRange, nil
}

func (mock *rangeManagerMock) getNextIDRange(ctx context.Context) (Range, error) {
	return Range{
		Start: mock.initRange.Start + mock.jumpRange,
		End:   mock.initRange.End + mock.jumpRange,
	}, nil
}

func TestIDManagement(t *testing.T) {
	ctx := context.Background()
	idMng, _ := NewManager(ctx, &rangeManagerMock{
		initRange: Range{
			Start: 2,
			End:   3,
		},
		jumpRange: 100,
	})

	assert.Equal(t, uint(2), idMng.GetLastID())
	nextID, _ := idMng.GetNextID(ctx)
	assert.Equal(t, uint(3), nextID)
	nextID, _ = idMng.GetNextID(ctx)
	assert.Equal(t, uint(102), nextID)
	nextID, _ = idMng.GetNextID(ctx)
	assert.Equal(t, uint(103), nextID)
	assert.Equal(t, uint(103), idMng.GetLastID())
}
