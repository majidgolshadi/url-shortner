package url

import (
	"context"
	"github.com/majidgolshadi/url-shortner/internal/domain"
	"github.com/majidgolshadi/url-shortner/internal/id"
	intErr "github.com/majidgolshadi/url-shortner/internal/infrastructure/errors"
	"github.com/majidgolshadi/url-shortner/internal/token"
	"github.com/pkg/errors"
)

const maxGeneratedTokenConflictRetry = 3

type DataStore interface {
	Save(ctx context.Context, url *domain.Url) error
	Delete(ctx context.Context, token string) error
	Fetch(ctx context.Context, token string) (*domain.Url, error)
}

type Service struct {
	idManager      *id.Manager
	tokenGenerator token.Generator
	datastore      DataStore
}

func NewService(idManager *id.Manager, tokenGenerator token.Generator, datastore DataStore) *Service {
	return &Service{
		idManager:      idManager,
		tokenGenerator: tokenGenerator,
		datastore:      datastore,
	}
}

func (s *Service) AddUrl(ctx context.Context, url string) (insertError error) {
	for i := 0; i < maxGeneratedTokenConflictRetry; i++ {
		identifier, err := s.idManager.GetNextID(ctx)
		if err != nil {
			return err
		}

		tk := s.tokenGenerator.GetToken(identifier)

		insertError = s.datastore.Save(ctx, &domain.Url{
			UrlPath: url,
			Token:   tk,
		})

		if errors.Is(insertError, intErr.RepositoryDuplicateTokenErr) {
			// TODO: log as warning
		} else {
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
