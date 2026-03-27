package http

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"
	"go.opentelemetry.io/contrib/instrumentation/github.com/gorilla/mux/otelmux"

	"github.com/majidgolshadi/url-shortner/internal/server/protocol/http/middleware"
	"github.com/majidgolshadi/url-shortner/internal/usecase"
)

const shutdownTimeout = 5 * time.Second

// Server holds HTTP server dependencies.
type Server struct {
	urlService  URLService
	logger      *logrus.Entry
	serviceName string
}

// NewHTTPServer creates a new HTTP server instance.
func NewHTTPServer(urlService URLService, logger *logrus.Entry, serviceName string) *Server {
	return &Server{
		urlService:  urlService,
		logger:      logger,
		serviceName: serviceName,
	}
}

// Run starts the HTTP server and handles graceful shutdown.
func (s *Server) Run(tag string, commit string, httpPort string) error {
	router := s.setupRoutes(tag, commit)

	srv := &http.Server{
		Addr:    ":" + httpPort,
		Handler: router,
	}

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-c
		s.logger.Info("shutting down HTTP/REST server...")
		ctx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
		defer cancel()
		if err := srv.Shutdown(ctx); err != nil {
			s.logger.Errorf("HTTP server shutdown error: %v", err)
		}
	}()

	s.logger.Info("starting HTTP/REST gateway on port ", httpPort)
	return srv.ListenAndServe()
}

func (s *Server) setupRoutes(tag string, commit string) *mux.Router {
	urlHandler := NewURLHandler(s.urlService, s.logger)

	hcs := usecase.NewHealthCheckService()
	hc := NewHealthCheckHandler(tag, commit, s.logger, hcs)

	router := mux.NewRouter()

	// OpenTelemetry middleware for automatic HTTP request tracing
	router.Use(otelmux.Middleware(s.serviceName))
	router.Use(middleware.ContentType)

	urlRoutes := router.PathPrefix("/url").Subrouter()
	urlRoutes.HandleFunc("", urlHandler.addUrlHandle).Methods(http.MethodPost)
	urlRoutes.HandleFunc("/{token}", urlHandler.fetchUrlHandle).Methods(http.MethodGet)
	urlRoutes.HandleFunc("/{token}", urlHandler.deleteUrlHandle).Methods(http.MethodDelete)
	urlRoutes.HandleFunc("/{token}/og", urlHandler.refreshOgHandle).Methods(http.MethodPut)

	router.HandleFunc("/healthcheck", hc.Handle)

	// Redirect route: when a user visits /{token}, redirect to the original URL with custom headers
	router.HandleFunc("/{token}", urlHandler.redirectHandle).Methods(http.MethodGet)

	return router
}