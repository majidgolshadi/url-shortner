package id

import (
	"context"
	"github.com/pkg/errors"
	"sync"
	"time"

	"github.com/majidgolshadi/url-shortner/internal/domain"
	intErr "github.com/majidgolshadi/url-shortner/internal/infrastructure/errors"
	"github.com/majidgolshadi/url-shortner/internal/storage"
)

const (
	reserveRangeMaxRetry = 3

	reserveRangeWaitingTimeMillisecond = 200
)

type datastoreRangeManager struct {
	nodeID        string
	rangeSize     int
	reservedRange Range
	coordinator   storage.Coordinator

	mux sync.Mutex
}

func NewDataStoreRangeManager(nodeID string, rangeSize int, coordinator storage.Coordinator) RangeManager {
	return &datastoreRangeManager{
		nodeID:      nodeID,
		rangeSize:   rangeSize,
		coordinator: coordinator,
	}
}

func (c *datastoreRangeManager) getCurrentRange(ctx context.Context) (Range, error) {
	if c.reservedRange.End != 0 {
		return Range{
			Start: c.reservedRange.Start,
			End:   c.reservedRange.End,
		}, nil
	}

	c.mux.Lock()
	defer c.mux.Unlock()

	rng, err := c.coordinator.GetNodeReservedRange(ctx, c.nodeID)
	if err != nil {
		return Range{}, err
	}

	return Range{
		Start: rng.Start,
		End:   rng.End,
	}, nil
}

func (c *datastoreRangeManager) getNextIDRange(ctx context.Context) (rng Range, err error) {
	var takenRange *domain.Range
	c.mux.Lock()
	defer c.mux.Unlock()

	for i := 0; i < reserveRangeMaxRetry; i++ {
		takenRange, err = c.coordinator.TakeNextFreeRange(ctx, c.nodeID, c.rangeSize)
		if !errors.Is(err, intErr.CoordinatorTakeNextFreeRangeErr) {
			return Range{}, err
		}

		// TODO: log the error as warning
		time.Sleep(reserveRangeWaitingTimeMillisecond * time.Millisecond)
	}

	c.reservedRange.Start = takenRange.Start
	c.reservedRange.End = takenRange.End

	return c.reservedRange, nil
}

func (c *datastoreRangeManager) updateRemainedRange(lastConsumedID uint) error {
	c.mux.Lock()
	defer c.mux.Unlock()

	return c.coordinator.UpdateRemainedRange(c.nodeID, domain.Range{
		Start: lastConsumedID + 1,
		End:   c.reservedRange.End,
	})
}
