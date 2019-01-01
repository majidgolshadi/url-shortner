package url_shortner

import (
	"crypto/md5"
	"fmt"
	"github.com/farmx/goscraper"
	"github.com/marksalpeter/token"
	"github.com/pkg/errors"
)

type tokenGenerator struct {
	counter   counter
	datastore datastore
}

func NewTokenGenerator(counter counter, datastore datastore) *tokenGenerator {
	return &tokenGenerator{
		counter:   counter,
		datastore: datastore,
	}
}

func (tg *tokenGenerator) NewUrl(longUrl string) string {
	title := make(chan string)

	go func() { title <- tg.getUrlTitle(longUrl) }()
	md5str := fmt.Sprintf("%x", md5.Sum([]byte(longUrl)))
	tk := tg.datastore.getToken(md5str)

	if tk == "" {
		tk = token.Token(tg.counter.next()).Encode()
		tg.datastore.persist(&urlMap{
			MD5:   md5str,
			token: tk,
			url:   longUrl,
			title: <-title,
		})
	}

	return tk
}

func (tg *tokenGenerator) NewUrlWithCustomToken(longUrl string, customToken string) (string, error) {
	title := make(chan string)

	go func() { title <- tg.getUrlTitle(longUrl) }()
	md5str := fmt.Sprintf("%x", md5.Sum([]byte(longUrl)))
	tk := tg.datastore.getToken(md5str)

	if tk == "" {
		if !tg.datastore.tokenIsUsed(customToken) {
			return customToken, tg.datastore.persist(&urlMap{
				MD5:   md5str,
				token: customToken,
				url:   longUrl,
				title: <-title,
			})
		}

		return tk, errors.New("token is already in used, please choose another one")

	}

	return tk, errors.New("origin url was registered")
}

func (tg *tokenGenerator) GetLongUrl(token string) string {
	return tg.datastore.getLongUrl(token)
}

func (tg *tokenGenerator) getUrlTitle(longUrl string) string {
	s, err := goscraper.Scrape(longUrl, 1)
	if err != nil {
		println(err)
		return ""
	}

	return s.Preview.Title
}
