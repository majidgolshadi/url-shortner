package url_shortner

import (
	"database/sql"
	"errors"
	"fmt"
	_ "github.com/go-sql-driver/mysql"
)

type MariaDbConfig struct {
	Host     string
	Username string
	Password string
	Database string
}

type mariadb struct {
	conn *sql.DB
}

type urlMap struct {
	MD5   string
	url   string
	token string
}

func (cnf *MariaDbConfig) init() error {
	if cnf.Host == "" {
		return errors.New("host does not set")
	}

	if cnf.Database == "" {
		return errors.New("database name does not set")
	}

	if cnf.Username == "" {
		return errors.New("username does not set")
	}

	return nil
}

func DbConnect(cnf *MariaDbConfig) (*mariadb, error) {
	if err := cnf.init(); err != nil {
		return nil, err
	}

	datasourceName := fmt.Sprintf("%s:%s@tcp(%s)/%s", cnf.Username, cnf.Password, cnf.Host, cnf.Database)
	db, err := sql.Open("mysql", datasourceName)
	if err != nil {
		return nil, err
	}

	return &mariadb{
		conn: db,
	}, nil
}

func (m *mariadb) getToken(md5 string) string {
	var row urlMap
	err := m.conn.QueryRow("select token from url_map where md5=?", md5).Scan(&row.token)
	if err != nil {
		return ""
	}

	return row.token
}

func (m *mariadb) persist(row *urlMap) (err error) {
	_,err = m.conn.Exec("INSERT INTO url_map (md5, token, url) VALUES (?, ?, ?)", row.MD5, row.token, row.url)
	return
}

func (m *mariadb) tokenIsUsed(token string) bool {
	var row urlMap
	err := m.conn.QueryRow("select md5 from url_map where token=?", token).Scan(&row.MD5)

	if err != nil {
		println(err.Error())
	}

	return row.MD5 != ""
}

func (m *mariadb) getTokenLogUrl(token string) string {
	var row urlMap
	err := m.conn.QueryRow("select url from url_map where token=?", token).Scan(&row.url)

	if err != nil {
		println(err.Error())
	}

	return row.url
}

func (m *mariadb) Close() {
	m.conn.Close()
}
