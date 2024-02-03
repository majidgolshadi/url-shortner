package url

import (
	"context"
	"github.com/majidgolshadi/url-shortner/internal/domain"
	"github.com/majidgolshadi/url-shortner/internal/id"
	"github.com/majidgolshadi/url-shortner/internal/token"
)

const maxGeneratedTokenConflictRetry = 3

type DataStore interface {
	Save(ctx context.Context, url *domain.Url) error
	Delete(ctx context.Context, token string) error
	Fetch(ctx context.Context, token string) (*domain.Url, error)
}

type Service struct {
	idManager      id.Manager
	tokenGenerator token.Generator
	datastore      DataStore
}

func (s *Service) AddUrl(ctx context.Context, url string) (insertError error) {
	for i := 0; i < maxGeneratedTokenConflictRetry; i++ {
		identifier, err := s.idManager.GetNextID()
		if err != nil {
			return err
		}

		tk := s.tokenGenerator.GetToken(identifier)

		insertError = s.datastore.Save(ctx, &domain.Url{
			UrlPath: url,
			Token:   tk,
		})

		// TODO: retry on duplicate token error only
		if insertError == nil {
			return
		}
	}

	return
}

func (s *Service) Delete(ctx context.Context, token string) error {
	return s.Delete(ctx, token)
}

func (s *Service) Fetch(ctx context.Context, token string) (*domain.Url, error) {
	return s.datastore.Fetch(ctx, token)
}
