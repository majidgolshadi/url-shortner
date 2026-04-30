package storage

import (
	"context"

	"github.com/majidgolshadi/url-shortner/internal/domain"
)

type Repository interface {
	Save(ctx context.Context, url *domain.URL) error
	Delete(ctx context.Context, token string) error
	Fetch(ctx context.Context, token string) (*domain.URL, error)
	UpdateOgHTML(ctx context.Context, token string, ogHTML string) error
	CountByCustomer(ctx context.Context, customerID string) (int, error)
}
