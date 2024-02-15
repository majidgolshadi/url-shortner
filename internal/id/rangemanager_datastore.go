package id

import (
	"context"
	"database/sql"
	"errors"
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
	rangeSize     uint
	reservedRange domain.Range
	coordinator   storage.Coordinator

	mux sync.Mutex
}

func NewDataStoreRangeManager(nodeID string, rangeSize int, coordinator storage.Coordinator) RangeManager {
	return &datastoreRangeManager{
		nodeID:      nodeID,
		rangeSize:   uint(rangeSize),
		coordinator: coordinator,
	}
}

func (c *datastoreRangeManager) getCurrentRange(ctx context.Context) (domain.Range, error) {
	if c.reservedRange.Start != c.reservedRange.End {
		return c.reservedRange, nil
	}

	c.mux.Lock()
	defer c.mux.Unlock()

	rng, err := c.coordinator.GetNodeReservedRange(ctx, c.nodeID)
	if errors.Is(err, sql.ErrNoRows) {
		return domain.Range{}, intErr.RangeManagerNoReservedRangeErr
	}

	if err != nil {
		return domain.Range{}, err
	}

	c.reservedRange.Start = rng.Start
	c.reservedRange.End = rng.End

	return c.reservedRange, nil
}

func (c *datastoreRangeManager) getNextIDRange(ctx context.Context) (domain.Range, error) {
	c.mux.Lock()
	defer c.mux.Unlock()

	for i := 0; i < reserveRangeMaxRetry; i++ {
		err := c.takeRange(ctx)

		// range has been taken successfully
		if err == nil {
			break
		}

		if !errors.Is(err, intErr.CoordinatorDataInvalidVersionErr) {
			return domain.Range{}, err
		}

		// TODO: log the error as warning

		// wait and then retry
		time.Sleep(reserveRangeWaitingTimeMillisecond * time.Millisecond)
	}

	return c.reservedRange, nil
}

func (c *datastoreRangeManager) takeRange(ctx context.Context) error {
	end, version, err := c.coordinator.GetLatestReservedRange(ctx)
	if errors.Is(err, intErr.CoordinatorNoReservedRangeErr) {
		c.reservedRange.Start = 1
		c.reservedRange.End = c.rangeSize

		return c.coordinator.TakeFreeRange(ctx, c.nodeID, c.reservedRange, 1)
	}

	// set the range according the latest taken range
	c.reservedRange.Start = end + 1
	c.reservedRange.End = end + 1 + c.rangeSize

	return c.coordinator.TakeFreeRange(ctx, c.nodeID, c.reservedRange, version+1)
}
