package url

import (
	"github.com/majidgolshadi/url-shortner/internal/domain"
	"github.com/majidgolshadi/url-shortner/internal/id"
	"github.com/majidgolshadi/url-shortner/internal/token"
)

const maxGeneratedTokenConflictRetry = 3

type DataStore interface {
	Save(url *domain.Url) error
	Delete(token string) error
	Fetch(token string) (*domain.Url, error)
}

type Service struct {
	idManager      id.Manager
	tokenGenerator token.Generator
	datastore      DataStore
}

func (s *Service) AddUrl(url string) (insertError error) {
	for i := 0; i < maxGeneratedTokenConflictRetry; i++ {
		identifier, err := s.idManager.GetNextID()
		if err != nil {
			return err
		}

		tk := s.tokenGenerator.GetToken(identifier)

		insertError = s.datastore.Save(&domain.Url{
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

func (s *Service) Delete(token string) error {
	return s.Delete(token)
}

func (s *Service) Fetch(token string) (*domain.Url, error) {
	return s.datastore.Fetch(token)
}
