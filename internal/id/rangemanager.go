package id

import (
	"context"

	"github.com/majidgolshadi/url-shortner/internal/domain"
)

type RangeManager interface {
	getCurrentRange(ctx context.Context) (domain.Range, error)
	getNextIDRange(ctx context.Context) (domain.Range, error)
}
