package id

import (
	"context"
	"errors"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"

	"github.com/majidgolshadi/url-shortner/internal/domain"
	intErr "github.com/majidgolshadi/url-shortner/internal/infrastructure/errors"
)

type rangeManagerMock struct {
	initRange domain.Range
	jumpRange uint

	// override for error testing
	currentRangeErr error
	nextRangeErr    error
}

func (mock *rangeManagerMock) getCurrentRange(_ context.Context) (domain.Range, error) {
	if mock.currentRangeErr != nil {
		return domain.Range{}, mock.currentRangeErr
	}
	return mock.initRange, nil
}

func (mock *rangeManagerMock) getNextIDRange(_ context.Context) (domain.Range, error) {
	if mock.nextRangeErr != nil {
		return domain.Range{}, mock.nextRangeErr
	}
	return domain.Range{
		Start: mock.initRange.Start + mock.jumpRange,
		End:   mock.initRange.End + mock.jumpRange,
	}, nil
}

func testIDLogger() *logrus.Entry {
	return logrus.NewEntry(logrus.StandardLogger())
}

func TestIDManagement(t *testing.T) {
	ctx := context.Background()
	idMng, _ := NewManager(ctx, &rangeManagerMock{
		initRange: domain.Range{Start: 2, End: 3},
		jumpRange: 100,
	}, testIDLogger())

	assert.Equal(t, uint(2), idMng.GetLastID())
	nextID, _ := idMng.GetNextID(ctx)
	assert.Equal(t, uint(3), nextID)
	nextID, _ = idMng.GetNextID(ctx)
	assert.Equal(t, uint(102), nextID)
	nextID, _ = idMng.GetNextID(ctx)
	assert.Equal(t, uint(103), nextID)
	assert.Equal(t, uint(103), idMng.GetLastID())
}

func TestNewManager_FreshRunSuccess(t *testing.T) {
	ctx := context.Background()
	mock := &rangeManagerMock{
		currentRangeErr: intErr.RangeManagerNoReservedRangeErr,
		initRange:       domain.Range{Start: 10, End: 20},
		jumpRange:       0,
	}
	idMng, err := NewManager(ctx, mock, testIDLogger())
	assert.NoError(t, err)
	assert.NotNil(t, idMng)
}

func TestNewManager_FreshRunError(t *testing.T) {
	ctx := context.Background()
	mock := &rangeManagerMock{
		currentRangeErr: intErr.RangeManagerNoReservedRangeErr,
		nextRangeErr:    errors.New("coordination store unavailable"),
	}
	idMng, err := NewManager(ctx, mock, testIDLogger())
	assert.Error(t, err)
	assert.Nil(t, idMng)
}

func TestNewManager_CurrentRangeUnknownError(t *testing.T) {
	ctx := context.Background()
	mock := &rangeManagerMock{
		currentRangeErr: errors.New("db connection error"),
	}
	idMng, err := NewManager(ctx, mock, testIDLogger())
	assert.Error(t, err)
	assert.Nil(t, idMng)
}

func TestGetNextID_RangeExhaustedSuccess(t *testing.T) {
	ctx := context.Background()
	// Range {Start:0, End:1} gives exactly 1 usable ID (value 1).
	// jumpRange=100 means next range start is 0+100=100.
	mock := &rangeManagerMock{
		initRange: domain.Range{Start: 0, End: 1},
		jumpRange: 100,
	}
	idMng, err := NewManager(ctx, mock, testIDLogger())
	assert.NoError(t, err)

	// first call: lastID++ = 1, within range → returns 1
	id1, err := idMng.GetNextID(ctx)
	assert.NoError(t, err)
	assert.Equal(t, uint(1), id1)

	// second call: lastID++ = 2, exceeds End=1 → acquires new range {Start:100, End:101}, returns 100
	id2, err := idMng.GetNextID(ctx)
	assert.NoError(t, err)
	assert.Equal(t, uint(100), id2)
}

func TestGetNextID_RangeExhaustedError(t *testing.T) {
	ctx := context.Background()
	// Range {Start:0, End:1} gives exactly 1 usable ID.
	mock := &rangeManagerMock{
		initRange: domain.Range{Start: 0, End: 1},
	}
	idMng, err := NewManager(ctx, mock, testIDLogger())
	assert.NoError(t, err)

	// first call: uses the single available ID
	_, err = idMng.GetNextID(ctx)
	assert.NoError(t, err)

	// make the next range reservation fail
	mock.nextRangeErr = errors.New("cannot reserve new range")

	// second call: exhausts range and fails to reserve a new one
	_, err = idMng.GetNextID(ctx)
	assert.Error(t, err)
}