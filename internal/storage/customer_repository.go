package storage

import (
	"context"

	"github.com/majidgolshadi/url-shortner/internal/domain"
)

type CustomerRepository interface {
	Save(ctx context.Context, customer *domain.Customer) error
	FindByAuthToken(ctx context.Context, authToken string) (*domain.Customer, error)
}
