package domain

// URL represents a shortened URL entity.
type URL struct {
	Path    string
	Token   string
	Headers map[string]string
	// OgHTML contains pre-rendered Open Graph HTML meta tags fetched from the
	// original URL. This is served directly to social media bots for link previews.
	OgHTML     string
	CustomerID string
}
