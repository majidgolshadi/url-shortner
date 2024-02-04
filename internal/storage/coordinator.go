package storage

import (
	"context"
	"github.com/majidgolshadi/url-shortner/internal/domain"
)

type Coordinator interface {
	GetNodeReservedRange(ctx context.Context, nodeID string) (*domain.Range, error)
	TakeNextFreeRange(ctx context.Context, nodeID string, rangeSize int) (*domain.Range, error)
	UpdateRemainedRange(nodeID string, remainedRange domain.Range) error
}
