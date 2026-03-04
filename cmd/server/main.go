package main

import (
	"context"
	"log"
	"time"

	"github.com/majidgolshadi/url-shortner/cmd/server/config"
	"github.com/majidgolshadi/url-shortner/internal/id"
	intLogger "github.com/majidgolshadi/url-shortner/internal/infrastructure/logger"
	"github.com/majidgolshadi/url-shortner/internal/infrastructure/sql"
	"github.com/majidgolshadi/url-shortner/internal/infrastructure/sql/mysql"
	"github.com/majidgolshadi/url-shortner/internal/server/protocol/http"
	mysqlRepo "github.com/majidgolshadi/url-shortner/internal/storage/mysql"
	"github.com/majidgolshadi/url-shortner/internal/token"
	"github.com/majidgolshadi/url-shortner/internal/usecase/url"
)

var (
	// CommitHash will be set at compile time with current git commit.
	CommitHash string
	// Tag will be set at compile time with current branch or tag.
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

	coordinationDB, err := sql.NewDBFactory(newDBConfig(cfg.ServiceName, cfg.Coordination.DataStore)).CreateDB()
	if err != nil {
		return err
	}

	coordinationStorage := mysqlRepo.NewCoordinator(coordinationDB)
	rangeMng := id.NewDataStoreRangeManager(cfg.Coordination.NodeID, cfg.Coordination.RangeSize, coordinationStorage)

	startupCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	idMng, err := id.NewManager(startupCtx, rangeMng)
	if err != nil {
		return err
	}

	appDB, err := sql.NewDBFactory(newDBConfig(cfg.ServiceName, cfg.AppDB)).CreateDB()
	if err != nil {
		return err
	}

	repo := mysqlRepo.NewRepository(appDB)
	urlSrv := url.NewService(idMng, &token.Base62TokenGenerator{}, repo)
	logger := intLogger.NewLogger(cfg.LogLevel)

	httpSrv := http.NewHTTPServer(urlSrv, logger)
	return httpSrv.Run(Tag, CommitHash, cfg.HTTPAddr)
}

func newDBConfig(serviceName string, dbCfg config.MySqlDB) *sql.DBConfig {
	return &sql.DBConfig{
		DSN: mysql.CreateDSN(
			dbCfg.Credential.Host,
			dbCfg.Credential.DBName,
			dbCfg.Credential.Username,
			dbCfg.Credential.Password,
			dbCfg.ReadTimeoutSec,
			dbCfg.WriteTimeoutSec,
		),
		ConnMaxLifetime: time.Duration(dbCfg.ConnLifetimeSec) * time.Second,
		MaxOpenConns:    dbCfg.MaxOpenConn,
		ServiceName:     serviceName + "-mysql",
	}
}