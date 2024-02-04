package id

import "context"

type inMemory struct {
	startID uint
}

func NewInMemoryRangeManager(startID uint) RangeManager {
	return &inMemory{
		startID: startID,
	}
}

func (c *inMemory) getCurrentRange(ctx context.Context) (Range, error) {
	return Range{
		Start: c.startID,
		End:   ^uint(0),
	}, nil
}

func (c *inMemory) getNextIDRange(ctx context.Context) (Range, error) {
	return Range{
		Start: c.startID,
		End:   ^uint(0),
	}, nil
}
