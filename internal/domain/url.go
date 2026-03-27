package domain

// URL represents a shortened URL entity.
type URL struct {
	Path    string
	Token   string
	Headers map[string]string
}
