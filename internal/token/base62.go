package token

type TokenGenerator interface {
	NewToken() string
}

type base64TokenGenerator struct {
}
