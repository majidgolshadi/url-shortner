package url

import (
	"github.com/majidgolshadi/url-shortner/internal/domain"
	"github.com/majidgolshadi/url-shortner/internal/id"
	"github.com/majidgolshadi/url-shortner/internal/token"
)

type DataStore interface {
	Save(url *domain.Url) error
	Delete(token string) error
	Fetch(token string) (*domain.Url, error)
}

type Config struct {
	maxInsert int
}

type Service struct {
	config      *Config
	idGenerator id.Generator
	tokenGenerator token.Generator
	datastore      DataStore
}

func (s *Service) AddUrl(url string) (insertError error) {
	for i := 0; i < s.config.maxInsert; i++ {
		identifier := s.idGenerator.NewID()
		tk := s.tokenGenerator.GetToken(identifier)

		insertError = s.datastore.Save(&domain.Url{
			UrlPath: url,
			Token:   tk,
		})

		// TODO: retry on duplicate token error
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
