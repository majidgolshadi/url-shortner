package url

type DeleteDataStore interface {
	DeleteUrl(token, url string) error
}

type DeleteService struct {
	datastore DeleteDataStore
}

func (s *DeleteService) Delete(token, url string) error {
	return s.datastore.DeleteUrl(token, url)
}
