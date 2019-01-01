package url_shortner

import (
	"database/sql"
	"errors"
	"fmt"
	_ "github.com/go-sql-driver/mysql"
	log "github.com/sirupsen/logrus"
)

type MariaDbConfig struct {
	Address      string
	Username     string
	Password     string
	Database     string
	MaxOpenConn  int
	MaxIdealConn int
}

type mariadb struct {
	conn *sql.DB
}

type urlMap struct {
	MD5   string
	url   string
	token string
	title string
}

type apiUser struct {
	Username string
	Password string
}

func (opt *MariaDbConfig) init() error {
	if opt.Database == "" {
		return errors.New("database name does not set")
	}

	if opt.Address == "" {
		opt.Address = "127.0.0.1:3306"
	}

	if opt.Username == "" {
		opt.Username = "root"
	}

	if opt.MaxOpenConn == 0 {
		opt.MaxOpenConn = 10
	}

	if opt.MaxIdealConn == 0 {
		opt.MaxIdealConn = 10
	}

	return nil
}

func DbConnect(cnf *MariaDbConfig) (*mariadb, error) {
	if err := cnf.init(); err != nil {
		return nil, err
	}

	datasourceName := fmt.Sprintf("%s:%s@tcp(%s)/%s", cnf.Username, cnf.Password, cnf.Address, cnf.Database)
	db, err := sql.Open("mysql", datasourceName)
	if err != nil {
		return nil, err
	}

	db.SetMaxIdleConns(cnf.MaxIdealConn)
	db.SetMaxOpenConns(cnf.MaxOpenConn)

	log.Info("mysql connect established to ", cnf.Address)

	return &mariadb{
		conn: db,
	}, nil
}

func (m *mariadb) getToken(md5 string) string {
	var row urlMap
	err := m.conn.QueryRow("SELECT token FROM url_map WHERE md5=?", md5).Scan(&row.token)
	if err != nil {
		log.Error("MariaDB get token error: ", err.Error())
		return ""
	}

	return row.token
}

func (m *mariadb) persist(row *urlMap) (err error) {
	_, err = m.conn.Exec("INSERT INTO url_map (md5, token, url, title) VALUES (?, ?, ?, ?)", row.MD5, row.token, row.url, row.title)
	return
}

func (m *mariadb) tokenIsUsed(token string) bool {
	var row urlMap
	err := m.conn.QueryRow("SELECT md5 FROM url_map WHERE token=?", token).Scan(&row.MD5)

	if err != nil {
		log.Error("MariaDB token is used error: ", err.Error())
	}

	return row.MD5 != ""
}

func (m *mariadb) getLongUrl(token string) string {
	var row urlMap
	err := m.conn.QueryRow("SELECT url FROM url_map WHERE token=?", token).Scan(&row.url)

	if err != nil {
		log.Error("MariaDB get long url error: ", err.Error())
	}

	return row.url
}

func (m *mariadb) authorizedUser(username string, password string) bool {
	var row apiUser
	err := m.conn.QueryRow("SELECT username FROM api_users WHERE username=? AND password=?", username, password).Scan(&row.Username)

	if err != nil {
		log.Error("MariaDB authorized user error: ", err.Error())
	}

	return row.Username != ""
}

func (m *mariadb) Close() {
	m.conn.Close()
}
