package url_shortner

type MockDataStore struct {
	CalledPersist bool
}

func (d *MockDataStore) getToken(md5 string) string {
	if md5 == "f52b1ad2fb0c02e979a197185721b720" {
		return "testToken"
	}

	return ""
}

func (d *MockDataStore) persist(row *urlMap) error {
	d.CalledPersist = true
	return nil
}

func (d *MockDataStore) tokenIsUsed(token string) bool {
	if token == "usedToken" {
		return true
	}

	return false
}

func (d *MockDataStore) getLongUrl(token string) string {
	if token == "testToken" {
		return "http://test.domain.com"
	}

	return "http://real-domain.com"
}

func (d *MockDataStore) authorizedUser(username string, password string) bool {
	return true
}