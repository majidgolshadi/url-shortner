package url

import "github.com/majidgolshadi/url-shortner/internal/domain"

type FetchDataStore interface {
	FetchUrl(token string) (*domain.Url, error)
}

type FetchService struct {
	datastore FetchDataStore
}

func (s *FetchService) Fetch(token string) (*domain.Url, error) {
	return s.datastore.FetchUrl(token)
}
