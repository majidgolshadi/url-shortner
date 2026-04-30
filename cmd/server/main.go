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
	"github.com/majidgolshadi/url-shortner/internal/infrastructure/telemetry"
	"github.com/majidgolshadi/url-shortner/internal/opengraph"
	"github.com/majidgolshadi/url-shortner/internal/server/protocol/http"
	mysqlRepo "github.com/majidgolshadi/url-shortner/internal/storage/mysql"
	"github.com/majidgolshadi/url-shortner/internal/token"
	"github.com/majidgolshadi/url-shortner/internal/usecase/customer"
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

	logger := intLogger.NewLogger(cfg.LogLevel)

	// Initialize OpenTelemetry
	telemetryCtx, telemetryCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer telemetryCancel()

	tp, err := telemetry.Setup(telemetryCtx, telemetry.Config{
		ServiceName:    cfg.ServiceName,
		ServiceVersion: Tag,
		Environment:    cfg.Environment,
		ExporterType:   cfg.Telemetry.ExporterType,
		OTLPEndpoint:   cfg.Telemetry.OTLPEndpoint,
		Enabled:        cfg.Telemetry.Enabled,
	})
	if err != nil {
		logger.WithError(err).Error("failed to initialize telemetry")
		return err
	}
	defer func() {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if shutdownErr := tp.Shutdown(shutdownCtx); shutdownErr != nil {
			logger.WithError(shutdownErr).Error("failed to shutdown telemetry")
		}
	}()

	logger.Info("telemetry initialized")

	// Coordination DB is separate from the app DB so range reservation can be
	// scaled or failed over independently from URL data storage.
	coordinationDB, err := sql.NewDBFactory(newDBConfig(cfg.ServiceName, cfg.Coordination.DataStore)).CreateDB()
	if err != nil {
		return err
	}

	coordinationStorage := mysqlRepo.NewCoordinator(coordinationDB, logger.WithField("component", "coordinator"))
	rangeMng := id.NewDataStoreRangeManager(cfg.Coordination.NodeID, cfg.Coordination.RangeSize, coordinationStorage)

	// ID manager must claim a range before the server starts accepting traffic;
	// 3s is enough for a healthy DB but short enough to fail fast on startup.
	startupCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	idMng, err := id.NewManager(startupCtx, rangeMng, logger.WithField("component", "id_manager"))
	if err != nil {
		return err
	}

	appDB, err := sql.NewDBFactory(newDBConfig(cfg.ServiceName, cfg.AppDB)).CreateDB()
	if err != nil {
		return err
	}

	repo := mysqlRepo.NewRepository(appDB, logger.WithField("component", "repository"))

	customerRepo := mysqlRepo.NewCustomerRepository(appDB, logger.WithField("component", "customer_repository"))
	customerSrv := customer.NewService(customerRepo, logger.WithField("component", "customer_service"))

	ogFetchTimeout := cfg.OpenGraph.FetchTimeoutSec
	// A zero value means the field was omitted from config; fall back to a safe default.
	if ogFetchTimeout <= 0 {
		ogFetchTimeout = 10
	}
	ogFetcher := opengraph.NewFetcher(ogFetchTimeout)

	urlSrv := url.NewService(idMng, &token.Base62TokenGenerator{}, repo, ogFetcher, cfg.Customer.DefaultBudget, logger.WithField("component", "url_service"))

	httpSrv := http.NewHTTPServer(urlSrv, customerSrv, logger, cfg.ServiceName)
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