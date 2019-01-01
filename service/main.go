package main

import (
	"github.com/BurntSushi/toml"
	"github.com/majidgolshadi/url-shortner"
	log "github.com/sirupsen/logrus"
	"net/http"
	_ "net/http/pprof"
	"os"
)

type config struct {
	HttpPort     string `toml:"rest_api_port"`
	DebugPort    string `toml:"debug_port"`
	ApiSecretKey string `toml:"api_secret_key"`

	Log   Log
	Mysql Mysql
	Etcd  Etcd
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
	CheckInterval      int    `toml:"check_interval"`
	MaxIdealConnection int    `toml:"max_ideal_conn"`
	MaxOpenConnection  int    `toml:"max_open_conn"`
}

type Etcd struct {
	Address string `toml:"address"`
	RootKey string `toml:"root_key"`
	NodeId  string `toml:"node_id"`
}

func main() {
	var cnf config
	var err error

	if _, err := toml.DecodeFile("config.toml", &cnf); err != nil {
		log.Fatal("read configuration file error ", err.Error())
	}

	initLogService(cnf.Log)

	go func() {
		log.Info("debugging server listening on port ", cnf.DebugPort)
		log.Println(http.ListenAndServe(cnf.DebugPort, nil))
	}()

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

	db, err := url_shortner.DbConnect(&url_shortner.MariaDbConfig{
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
