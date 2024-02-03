package url

import (
	"github.com/majidgolshadi/url-shortner/internal/domain"
	"github.com/majidgolshadi/url-shortner/internal/id"
	"github.com/majidgolshadi/url-shortner/internal/token"
)

type SaveDataStore interface {
	Save(url *domain.Url) error
}

type SaveConfig struct {
	maxInsert int
}

type Service struct {
	saveConfig     *SaveConfig
	idGenerator    id.Generator
	tokenGenerator token.Generator
	datastore      SaveDataStore
}

func (s *Service) AddUrl(url string) (insertError error) {
	for i := 0; i < s.saveConfig.maxInsert; i++ {
		id := s.idGenerator.NewID()
		token := s.tokenGenerator.GetToken(id)

		insertError = s.datastore.Save(&domain.Url{
			UrlPath: url,
			Token:   token,
		})

		// TODO: retry on duplicate token error
		if insertError == nil {
			return
		}
	}

	return
}
