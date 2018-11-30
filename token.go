package url_shortner

import (
	"crypto/md5"
	"fmt"
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

	mdb, err := DbConnect(dbConfig)
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

func NewUrl(originUrl string) string {
	md5str := fmt.Sprintf("%x", md5.Sum([]byte(originUrl)))
	tk := tg.mariadb.getToken(md5str)

	if tk == "" {
		tk = token.Token(tg.next()).Encode()
		tg.mariadb.persist(&urlMap{
			MD5:   md5str,
			token: tk,
			url:   originUrl,
		})
	}

	return tk
}

func NewUrlWithCustomToken(originUrl string, customToken string) (string, error) {
	md5str := fmt.Sprintf("%x", md5.Sum([]byte(originUrl)))
	tk := tg.mariadb.getToken(md5str)

	if tk == "" {
		return customToken, tg.mariadb.persist(&urlMap{
			MD5:   md5str,
			token: customToken,
			url:   originUrl,
		})
	}

	return tk, errors.New("Origin url was registered")
}
