package storage

import (
	"context"
	"github.com/majidgolshadi/url-shortner/internal/domain"
)

type Repository interface {
	Save(ctx context.Context, url *domain.Url) error
	Delete(ctx context.Context, token string) error
	Fetch(ctx context.Context, token string) (*domain.Url, error)
}
