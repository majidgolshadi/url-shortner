package url_shortner

import (
	"crypto/md5"
	"fmt"
	"github.com/farmx/goscraper"
	"github.com/marksalpeter/token"
	"github.com/pkg/errors"
	"log"
)

type tokenGenerator struct {
	etcd    *etcdDatasource
	mariadb *mariadb
	last    int
	max     int
}

var tg tokenGenerator

func InitTokenGenerator(config *EtcdConfig, dbConfig *MariaDbConfig) error {
	etcd, err := NewEtcd(config)
	if err != nil {
		return err
	}

	tg.etcd = etcd

	tg.last, tg.max, err = tg.etcd.restoreStartPoint()
	if err != nil {
		return err
	}

	mdb, err := dbConnect(dbConfig)
	if err != nil {
		return err
	}

	tg.mariadb = mdb

	return nil
}

func (tg *tokenGenerator) next() int {
	if tg.last+1 > tg.max {
		var err error
		tg.last, tg.max, err = tg.etcd.getNewRange()

		if err != nil {
			log.Fatal(err.Error())
		}
	}

	tg.last += 1

	tg.etcd.save(tg.last, tg.max)
	return tg.last
}

func NewUrl(longUrl string) string {
	title := make(chan string)

	go func() {title <- getUrlTitle(longUrl)}()
	md5str := fmt.Sprintf("%x", md5.Sum([]byte(longUrl)))
	tk := tg.mariadb.getToken(md5str)

	if tk == "" {
		tk = token.Token(tg.next()).Encode()
		tg.mariadb.persist(&urlMap{
			MD5:   md5str,
			token: tk,
			url:   longUrl,
			title: <-title,
		})
	}

	return tk
}

func NewUrlWithCustomToken(longUrl string, customToken string) (string, error) {
	title := make(chan string)

	go func() {title <- getUrlTitle(longUrl)}()
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

func GetLongUrl(token string) string {
	return tg.mariadb.getLongUrl(token)
}

func getUrlTitle(longUrl string) string {
	s, err := goscraper.Scrape(longUrl, 1)
	if err != nil {
		println(err)
		return ""
	}

	return s.Preview.Title
}
