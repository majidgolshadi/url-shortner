package url_shortner

type datastore interface {
	getToken(md5 string) string
	persist(row *urlMap) error
	tokenIsUsed(token string) bool
	getLongUrl(token string) string
}
