package url

import "github.com/majidgolshadi/url-shortner/internal/token"

type SaveDataStore interface {
	Save(token, url string) error
}

type Service struct {
	datastore      SaveDataStore
	tokenGenerator token.TokenGenerator
}

func (s *Service) AddUrl(url string) error {
	t, err := s.tokenGenerator.NewToken()
	if err != nil {
		return err
	}

	return s.datastore.Save(t, url)
}
