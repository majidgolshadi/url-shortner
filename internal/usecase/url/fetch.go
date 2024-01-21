package url

type FetchDataStore interface {
	// TODO: return domain url model instead of string
	FetchUrl(token string) (string, error)
}

type FetchService struct {
	datastore FetchDataStore
}

func (s *FetchService) Fetch(token string) (string, error) {
	return s.datastore.FetchUrl(token)
}
