package http

import (
	"context"
	"github.com/gorilla/mux"
	"github.com/majidgolshadi/url-shortner/internal/server/protocol/http/middleware"
	"github.com/majidgolshadi/url-shortner/internal/usecase"
	"github.com/sirupsen/logrus"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

const shutdownTimeout = 5 * time.Second

type server struct {
	urlService UrlService
	logger     *logrus.Entry
}

func InitHttpServer(urlService UrlService, logger *logrus.Entry) *server {
	return &server{
		urlService: urlService,
		logger: logger,
	}
}

func (s *server) RunServer(tag string, commit string, httpPort string) error {
	urlHandler := NewUrlHandler(s.urlService)

	hcs := usecase.NewHealthCheckService()
	hc := NewHealthCheckHandler(tag, commit, s.logger, hcs)

	router := mux.NewRouter()

	router.Use(middleware.ContentType)

	urlRoutes := router.StrictSlash(true).Path("/url").Subrouter()

	urlRoutes.Methods(http.MethodPost).HandlerFunc(urlHandler.addUrlHandle)
	urlRoutes.Methods(http.MethodGet).Path("/{token}").HandlerFunc(urlHandler.fetchUrlHandle)
	urlRoutes.Methods(http.MethodDelete).Path("/{token}").HandlerFunc(urlHandler.deleteUrlHandle)

	router.HandleFunc("/healthcheck", hc.Handle)

	srv := &http.Server{
		Addr:    ":" + httpPort,
		Handler: router,
	}

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		// sig is a ^C, handle it
		<-c
		s.logger.Info("shutting down HTTP/REST server...")
		ctx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
		defer cancel()
		_ = srv.Shutdown(ctx)
	}()

	// Start HTTP server
	s.logger.Info("starting HTTP/REST gateway on port ", httpPort)
	return srv.ListenAndServe()
}