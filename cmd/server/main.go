package main

import (
	"context"
	"github.com/majidgolshadi/url-shortner/internal/id"
	"github.com/majidgolshadi/url-shortner/internal/token"
	"github.com/majidgolshadi/url-shortner/internal/usecase/url"
	"log"
	"time"

	"github.com/majidgolshadi/url-shortner/cmd/server/config"
	"github.com/majidgolshadi/url-shortner/internal/infrastructure/sql"
	"github.com/majidgolshadi/url-shortner/internal/infrastructure/sql/mysql"
	mysqlRepo "github.com/majidgolshadi/url-shortner/internal/storage/mysql"
)

var (
	// CommitHash will be set at compile time with current git commit
	CommitHash string
	// Tag will be set at compile time with current branch or tag
	Tag string
)

func main() {
	if err := runServer(); err != nil {
		log.Fatal(err)
	}
}

func runServer() error {
	cfg, err := config.Load(CommitHash, Tag)
	if err != nil {
		return err
	}

	coordinationDbFactory := sql.NewDBFactory(getSqlFactoryConfig(cfg.ServiceName, cfg.Coordination.DataStore))
	coordinationDb, err := coordinationDbFactory.CreateDB()
	if err != nil {
		return err
	}

	coordinationStorage := mysqlRepo.NewCoordinator(coordinationDb)
	rangeMng := id.NewDataStoreRangeManager(cfg.Coordination.NodeID, cfg.Coordination.RangeSize, coordinationStorage)
	idManagementStartUpContext, cancel := context.WithTimeout(context.Background(), time.Second*3)
	defer cancel()
	idMng, err := id.NewManager(idManagementStartUpContext, rangeMng)
	if err != nil {
		return err
	}

	dbFactory := sql.NewDBFactory(getSqlFactoryConfig(cfg.ServiceName, cfg.AppDB))
	db, err := dbFactory.CreateDB()
	if err != nil {
		return err
	}

	repo := mysqlRepo.NewRepository(db)
	urlSrv := url.NewService(idMng, &token.Base64TokenGenerator{}, repo)
}

// dbConfigBy generates dbConfig by given config.
func getSqlFactoryConfig(serviceName string, config config.MySqlDB) *sql.DBConfig {
	return &sql.DBConfig{
		DSN: mysql.CreateDns(
			config.Credential.Host,
			config.Credential.DBName,
			config.Credential.Username,
			config.Credential.Password,
			config.ReadTimeoutSec,
			config.WriteTimeoutSec,
		),
		ConnMaxLifetime: time.Duration(config.ConnLifetimeSec) * time.Second,
		MaxOpenConns:    config.MaxOpenConn,
		ServiceName:     serviceName + "-mysql",
	}
}
