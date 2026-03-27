package id

import (
	"context"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"

	"github.com/majidgolshadi/url-shortner/internal/domain"
)

type rangeManagerMock struct {
	initRange domain.Range
	jumpRange uint
}

func (mock *rangeManagerMock) getCurrentRange(ctx context.Context) (domain.Range, error) {
	return mock.initRange, nil
}

func (mock *rangeManagerMock) getNextIDRange(ctx context.Context) (domain.Range, error) {
	return domain.Range{
		Start: mock.initRange.Start + mock.jumpRange,
		End:   mock.initRange.End + mock.jumpRange,
	}, nil
}

func TestIDManagement(t *testing.T) {
	ctx := context.Background()
	idMng, _ := NewManager(ctx, &rangeManagerMock{
		initRange: domain.Range{
			Start: 2,
			End:   3,
		},
		jumpRange: 100,
	}, logrus.NewEntry(logrus.StandardLogger()))

	assert.Equal(t, uint(2), idMng.GetLastID())
	nextID, _ := idMng.GetNextID(ctx)
	assert.Equal(t, uint(3), nextID)
	nextID, _ = idMng.GetNextID(ctx)
	assert.Equal(t, uint(102), nextID)
	nextID, _ = idMng.GetNextID(ctx)
	assert.Equal(t, uint(103), nextID)
	assert.Equal(t, uint(103), idMng.GetLastID())
}