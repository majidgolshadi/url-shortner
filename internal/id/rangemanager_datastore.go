package id

import (
	"context"
	"sync"

	"github.com/majidgolshadi/url-shortner/internal/domain"
	"github.com/majidgolshadi/url-shortner/internal/storage"
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

func (c *datastoreRangeManager) getNextIDRange(ctx context.Context) (Range, error) {
	c.mux.Lock()
	defer c.mux.Unlock()

	takenRange, err := c.coordinator.TakeNextFreeRange(ctx, c.nodeID, c.rangeSize)
	// TODO: retry multiple times to take a range
	if err != nil {
		return Range{}, err
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
