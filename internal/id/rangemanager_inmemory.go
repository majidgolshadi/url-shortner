package id

import (
	"context"
	"github.com/majidgolshadi/url-shortner/internal/domain"
)

type inMemory struct {
	startID uint
}

func NewInMemoryRangeManager(startID uint) RangeManager {
	return &inMemory{
		startID: startID,
	}
}

func (c *inMemory) getCurrentRange(ctx context.Context) (domain.Range, error) {
	return domain.Range{
		Start: c.startID,
		End:   ^uint(0),
	}, nil
}

func (c *inMemory) getNextIDRange(ctx context.Context) (domain.Range, error) {
	return domain.Range{
		Start: c.startID,
		End:   ^uint(0),
	}, nil
}
