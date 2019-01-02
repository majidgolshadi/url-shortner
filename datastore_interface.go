package url_shortner

type Datastore interface {
	Connect() error
	getToken(md5 string) string
	persist(row *urlMap) error
	tokenIsUsed(token string) bool
	getLongUrl(token string) string
	authorizedUser(username string, password string) bool
}
