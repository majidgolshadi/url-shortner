package url_shortner

import (
	"errors"
	"github.com/gocql/gocql"
	log "github.com/sirupsen/logrus"
)

type CassandraConfig struct {
	Address  []string
	Username string
	Password string
	Keyspace string
}

type cassandra struct {
	conn *gocql.Session
	cnf  *CassandraConfig
}

func (cnf *CassandraConfig) init() error {
	if cnf.Keyspace == "" {
		return errors.New("keyspace name does not set")
	}

	if len(cnf.Address) == 0 {
		cnf.Address[0] = "127.0.0.1:"
	}

	if cnf.Username == "" {
		cnf.Username = "root"
	}

	return nil
}

func NewCassandra(cnf *CassandraConfig) (*cassandra, error) {
	if err := cnf.init(); err != nil {
		return nil, err
	}

	return &cassandra{
		cnf: cnf,
	}, nil
}

func (c *cassandra) Connect() error {
	cluster := gocql.NewCluster(c.cnf.Address...)
	cluster.Keyspace = c.cnf.Keyspace
	cluster.Consistency = gocql.Quorum
	session, err := cluster.CreateSession()

	if err != nil {
		return err
	}

	log.Info("cassandra connect established to ", c.cnf.Address)
	c.conn = session
	return nil
}

func (c *cassandra) getToken(md5 string) string {
	var row urlMap
	err := c.conn.Query("SELECT token FROM url_map WHERE md5=?", md5).Scan(&row.token)
	if err != nil {
		log.Error("MariaDB get token error: ", err.Error())
		return ""
	}

	return row.token
}

func (c *cassandra) persist(row *urlMap) error {
	return c.conn.Query("INSERT INTO url_map (md5, token, url, title) VALUES (?, ?, ?, ?)",
		row.MD5, row.token, row.url, row.title).Exec()
}

func (c *cassandra) tokenIsUsed(token string) bool {
	var row urlMap
	err := c.conn.Query("SELECT md5 FROM url_map WHERE token=?", token).Scan(&row.MD5)

	if err != nil {
		log.Error("MariaDB token is used error: ", err.Error())
	}

	return row.MD5 != ""
}

func (c *cassandra) getLongUrl(token string) string {
	var row urlMap
	err := c.conn.Query("SELECT url FROM url_map WHERE token=?", token).Scan(&row.url)

	if err != nil {
		log.Error("MariaDB get long url error: ", err.Error())
	}

	return row.url
}

func (c *cassandra) authorizedUser(username string, password string) bool {
	var row apiUser
	err := c.conn.Query("SELECT username FROM api_users WHERE username=? AND password=?", username, password).Scan(&row.Username)

	if err != nil {
		log.Error("MariaDB authorized user error: ", err.Error())
	}

	return row.Username != ""
}

func (c *cassandra) Close() {
	c.conn.Close()
}
