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

	"github.com/majidgolshadi/url-shortner/internal/domain"
	"github.com/majidgolshadi/url-shortner/internal/server/protocol/http/middleware"
	"github.com/majidgolshadi/url-shortner/internal/usecase"
)

const shutdownTimeout = 5 * time.Second

// customerService is a combined interface used by the server to satisfy both
// the customer handler and the auth middleware.
type customerService interface {
	Register(ctx context.Context) (*domain.Customer, error)
	FindByAuthToken(ctx context.Context, authToken string) (*domain.Customer, error)
}

// Server holds HTTP server dependencies.
type Server struct {
	urlService  URLService
	custService customerService
	logger      *logrus.Entry
	serviceName string
}

// NewHTTPServer creates a new HTTP server instance.
func NewHTTPServer(urlService URLService, custService customerService, logger *logrus.Entry, serviceName string) *Server {
	return &Server{
		urlService:  urlService,
		custService: custService,
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
	customerHandler := NewCustomerHandler(s.custService, s.logger)

	hcs := usecase.NewHealthCheckService()
	hc := NewHealthCheckHandler(tag, commit, s.logger, hcs)

	router := mux.NewRouter()

	// OpenTelemetry middleware for automatic HTTP request tracing
	router.Use(otelmux.Middleware(s.serviceName))
	router.Use(middleware.ContentType)

	router.HandleFunc("/customer", customerHandler.registerHandle).Methods(http.MethodPost)

	// Subrouter scopes auth middleware to /url/* only; the /{token} redirect must remain public.
	authMiddleware := middleware.Auth(s.custService)
	urlRoutes := router.PathPrefix("/url").Subrouter()
	urlRoutes.Use(authMiddleware)
	urlRoutes.HandleFunc("", urlHandler.addUrlHandle).Methods(http.MethodPost)
	urlRoutes.HandleFunc("/{token}", urlHandler.fetchUrlHandle).Methods(http.MethodGet)
	urlRoutes.HandleFunc("/{token}", urlHandler.deleteUrlHandle).Methods(http.MethodDelete)
	urlRoutes.HandleFunc("/{token}/og", urlHandler.refreshOgHandle).Methods(http.MethodPut)

	router.HandleFunc("/healthcheck", hc.Handle)

	// Registered last so more specific /url/* and /customer routes take precedence.
	router.HandleFunc("/{token}", urlHandler.redirectHandle).Methods(http.MethodGet)

	return router
}