package url

import (
	"context"
	"fmt"

	"github.com/majidgolshadi/url-shortner/internal/domain"
	intErr "github.com/majidgolshadi/url-shortner/internal/infrastructure/errors"
	"github.com/majidgolshadi/url-shortner/internal/storage"
	"github.com/majidgolshadi/url-shortner/internal/token"
	"github.com/pkg/errors"
)

const maxGeneratedTokenConflictRetry = 3

// IDProvider abstracts ID generation for testability.
type IDProvider interface {
	GetNextID(ctx context.Context) (uint, error)
}

// Service handles URL shortening business logic.
type Service struct {
	idProvider     IDProvider
	tokenGenerator token.Generator
	repository     storage.Repository
}

// NewService creates a new URL service.
func NewService(idProvider IDProvider, tokenGenerator token.Generator, repository storage.Repository) *Service {
	return &Service{
		idProvider:     idProvider,
		tokenGenerator: tokenGenerator,
		repository:     repository,
	}
}

// Add creates a shortened URL. It retries on token conflicts up to maxGeneratedTokenConflictRetry times.
func (s *Service) Add(ctx context.Context, url string) (string, error) {
	var lastErr error
	for i := 0; i < maxGeneratedTokenConflictRetry; i++ {
		identifier, err := s.idProvider.GetNextID(ctx)
		if err != nil {
			return "", fmt.Errorf("generating next ID: %w", err)
		}

		tok := s.tokenGenerator.GetToken(identifier)

		lastErr = s.repository.Save(ctx, &domain.URL{
			Path:  url,
			Token: tok,
		})

		if lastErr == nil {
			return tok, nil
		}

		if !errors.Is(lastErr, intErr.RepositoryDuplicateTokenErr) {
			return "", lastErr
		}
		// duplicate token — retry with a new ID
	}

	return "", fmt.Errorf("failed to add URL after %d retries: %w", maxGeneratedTokenConflictRetry, lastErr)
}

// Delete removes a shortened URL by token.
func (s *Service) Delete(ctx context.Context, token string) error {
	return s.repository.Delete(ctx, token)
}

// Fetch retrieves a shortened URL by token.
func (s *Service) Fetch(ctx context.Context, token string) (*domain.URL, error) {
	return s.repository.Fetch(ctx, token)
}