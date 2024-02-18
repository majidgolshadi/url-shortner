package storage

import (
	"context"
	"github.com/majidgolshadi/url-shortner/internal/domain"
)

type Coordinator interface {
	GetNodeReservedRange(ctx context.Context, nodeID string) (*domain.Range, error)
	GetLatestReservedRange(ctx context.Context) (lastReservedNumber uint, version int, err error)
	TakeFreeRange(ctx context.Context, nodeID string, requestedRange domain.Range, version int) error
}
