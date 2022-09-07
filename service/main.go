package main

import (
	"flag"
	"net/http"
	_ "net/http/pprof"
	"os"
	"strings"

	"github.com/BurntSushi/toml"
	log "github.com/sirupsen/logrus"
)

type config struct {
	HttpPort     string `toml:"rest_api_port"`
	DebugPort    string `toml:"debug_port"`
	ApiSecretKey string `toml:"api_secret_key"`

	Log       Log
	Mysql     Mysql
	Cassandra Cassandra
	Etcd      Etcd
}

type Log struct {
	Format   string `toml:"format"`
	LogLevel string `toml:"log_level"`
	LogDst   string `toml:"log_dst"`
}

type Mysql struct {
	Address            string `toml:"address"`
	Username           string `toml:"username"`
	Password           string `toml:"password"`
	DB                 string `toml:"db"`
	MaxIdealConnection int    `toml:"max_ideal_conn"`
	MaxOpenConnection  int    `toml:"max_open_conn"`
}

type Cassandra struct {
	Address  string `toml:"address"`
	Username string `toml:"username"`
	Password string `toml:"password"`
	Keyspace string `toml:"keyspace"`
}

type Etcd struct {
	Address string `toml:"address"`
	RootKey string `toml:"root_key"`
	NodeId  string `toml:"node_id"`
}

func main() {
	var cnf config
	var err error

	configFilePath := flag.String("config", "config.toml", "specify config file path")
	flag.Parse()

	if _, err := toml.DecodeFile(*configFilePath, &cnf); err != nil {
		log.Fatal("read configuration file error ", err.Error())
	}

	initLogService(cnf.Log)

	go func() {
		log.Info("debugging server listening on port ", cnf.DebugPort)
		log.Println(http.ListenAndServe(cnf.DebugPort, nil))
	}()

	///////////////////////////////////////////////////////////
	// Etcd
	///////////////////////////////////////////////////////////
	coordinator, err := url_shortner.NewEtcd(&url_shortner.EtcdConfig{
		Hosts:   []string{cnf.Etcd.Address},
		RootKey: cnf.Etcd.RootKey,
		NodeId:  cnf.Etcd.NodeId,
	})

	if err != nil {
		log.Fatal(err.Error())
	}

	counter, err := url_shortner.NewDistributedAtomicCounter(coordinator)
	if err != nil {
		log.Fatal(err.Error())
	}

	///////////////////////////////////////////////////////////
	///////////////////////////////////////////////////////////
	// DataStore
	///////////////////////////////////////////////////////////
	///////////////////////////////////////////////////////////

	var db url_shortner.Datastore

	///////////////////////////////////////////////////////////
	// MariaDB
	///////////////////////////////////////////////////////////
	if cnf.Mysql.DB != "" {
		db, err := url_shortner.NewMariadb(&url_shortner.MariaDbConfig{
			Address:      cnf.Mysql.Address,
			Username:     cnf.Mysql.Username,
			Password:     cnf.Mysql.Password,
			Database:     cnf.Mysql.DB,
			MaxIdealConn: cnf.Mysql.MaxIdealConnection,
			MaxOpenConn:  cnf.Mysql.MaxOpenConnection,
		})

		if err != nil {
			log.Fatal(err.Error())
		}

		if err := db.Connect(); err != nil {
			log.Fatal(err.Error())
		}

	} else {
		///////////////////////////////////////////////////////////
		// Cassandra
		///////////////////////////////////////////////////////////
		db, err := url_shortner.NewCassandra(&url_shortner.CassandraConfig{
			Address:  strings.Split(cnf.Cassandra.Address, ","),
			Username: cnf.Cassandra.Username,
			Password: cnf.Cassandra.Password,
			Keyspace: cnf.Cassandra.Keyspace,
		})

		if err != nil {
			log.Fatal(err.Error())
		}

		if err := db.Connect(); err != nil {
			log.Fatal(err.Error())
		}
	}

	///////////////////////////////////////////////////////////
	// Rest API
	///////////////////////////////////////////////////////////
	url_shortner.RunRestApi(
		url_shortner.NewTokenGenerator(counter, db),
		db, cnf.ApiSecretKey, cnf.HttpPort)
}

func initLogService(logConfig Log) {
	switch logConfig.LogLevel {
	case "debug":
		log.SetLevel(log.DebugLevel)
	case "info":
		log.SetLevel(log.InfoLevel)
	case "error":
		log.SetLevel(log.ErrorLevel)
	default:
		log.SetLevel(log.WarnLevel)
	}

	switch logConfig.Format {
	case "json":
		log.SetFormatter(&log.JSONFormatter{})
	default:
		log.SetFormatter(&log.TextFormatter{})
	}

	if logConfig.LogDst != "" {
		f, err := os.Create(logConfig.LogDst)
		if err != nil {
			log.Fatal("create log file error: ", err.Error())
		}

		log.SetOutput(f)
	}
}
