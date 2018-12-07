package url_shortner

import (
	"crypto/md5"
	"fmt"
	"github.com/farmx/goscraper"
	"github.com/marksalpeter/token"
	"github.com/pkg/errors"
)

type tokenGenerator struct {
	counter    counter
	mariadb *mariadb
}

func NewTokenGenerator(counter counter, dbConfig *MariaDbConfig) (*tokenGenerator, error) {
	mdb, err := dbConnect(dbConfig)
	if err != nil {
		return nil, err
	}

	return &tokenGenerator{
		counter: counter,
		mariadb: mdb,
	},nil
}

func (tg *tokenGenerator) NewUrl(longUrl string) string {
	title := make(chan string)

	go func() {title <- tg.getUrlTitle(longUrl)}()
	md5str := fmt.Sprintf("%x", md5.Sum([]byte(longUrl)))
	tk := tg.mariadb.getToken(md5str)

	if tk == "" {
		tk = token.Token(tg.counter.next()).Encode()
		tg.mariadb.persist(&urlMap{
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

	go func() {title <- tg.getUrlTitle(longUrl)}()
	md5str := fmt.Sprintf("%x", md5.Sum([]byte(longUrl)))
	tk := tg.mariadb.getToken(md5str)

	if tk == "" {
		if !tg.mariadb.tokenIsUsed(customToken) {
			return customToken, tg.mariadb.persist(&urlMap{
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
	return tg.mariadb.getLongUrl(token)
}

func (tg *tokenGenerator) getUrlTitle(longUrl string) string {
	s, err := goscraper.Scrape(longUrl, 1)
	if err != nil {
		println(err)
		return ""
	}

	return s.Preview.Title
}
