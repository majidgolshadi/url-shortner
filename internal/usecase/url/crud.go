package url

import (
	"context"
	"fmt"
	"time"

	"github.com/sirupsen/logrus"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	otelmetric "go.opentelemetry.io/otel/metric"

	"github.com/majidgolshadi/url-shortner/internal/domain"
	intErr "github.com/majidgolshadi/url-shortner/internal/infrastructure/errors"
	intLogger "github.com/majidgolshadi/url-shortner/internal/infrastructure/logger"
	"github.com/majidgolshadi/url-shortner/internal/infrastructure/telemetry"
	"github.com/majidgolshadi/url-shortner/internal/storage"
	"github.com/majidgolshadi/url-shortner/internal/token"
	"github.com/pkg/errors"
)

const maxGeneratedTokenConflictRetry = 3

// IDProvider abstracts ID generation for testability.
type IDProvider interface {
	GetNextID(ctx context.Context) (uint, error)
}

// Service handles URL shortening business logic.
type Service struct {
	idProvider     IDProvider
	tokenGenerator token.Generator
	repository     storage.Repository
	logger         *logrus.Entry

	// metrics
	addCounter    otelmetric.Int64Counter
	addErrCounter otelmetric.Int64Counter
	addDuration   otelmetric.Float64Histogram

	fetchCounter    otelmetric.Int64Counter
	fetchErrCounter otelmetric.Int64Counter
	fetchDuration   otelmetric.Float64Histogram

	deleteCounter    otelmetric.Int64Counter
	deleteErrCounter otelmetric.Int64Counter
	deleteDuration   otelmetric.Float64Histogram
}

// NewService creates a new URL service.
func NewService(idProvider IDProvider, tokenGenerator token.Generator, repository storage.Repository, logger *logrus.Entry) *Service {
	meter := telemetry.Meter("url-shortener/usecase/url")

	addCounter, _ := meter.Int64Counter("url.add.total",
		otelmetric.WithDescription("Total number of URL shorten operations"))
	addErrCounter, _ := meter.Int64Counter("url.add.errors",
		otelmetric.WithDescription("Total number of URL shorten errors"))
	addDuration, _ := meter.Float64Histogram("url.add.duration_ms",
		otelmetric.WithDescription("Duration of URL shorten operations in milliseconds"),
		otelmetric.WithUnit("ms"))

	fetchCounter, _ := meter.Int64Counter("url.fetch.total",
		otelmetric.WithDescription("Total number of URL fetch operations"))
	fetchErrCounter, _ := meter.Int64Counter("url.fetch.errors",
		otelmetric.WithDescription("Total number of URL fetch errors"))
	fetchDuration, _ := meter.Float64Histogram("url.fetch.duration_ms",
		otelmetric.WithDescription("Duration of URL fetch operations in milliseconds"),
		otelmetric.WithUnit("ms"))

	deleteCounter, _ := meter.Int64Counter("url.delete.total",
		otelmetric.WithDescription("Total number of URL delete operations"))
	deleteErrCounter, _ := meter.Int64Counter("url.delete.errors",
		otelmetric.WithDescription("Total number of URL delete errors"))
	deleteDuration, _ := meter.Float64Histogram("url.delete.duration_ms",
		otelmetric.WithDescription("Duration of URL delete operations in milliseconds"),
		otelmetric.WithUnit("ms"))

	return &Service{
		idProvider:       idProvider,
		tokenGenerator:   tokenGenerator,
		repository:       repository,
		logger:           logger,
		addCounter:       addCounter,
		addErrCounter:    addErrCounter,
		addDuration:      addDuration,
		fetchCounter:     fetchCounter,
		fetchErrCounter:  fetchErrCounter,
		fetchDuration:    fetchDuration,
		deleteCounter:    deleteCounter,
		deleteErrCounter: deleteErrCounter,
		deleteDuration:   deleteDuration,
	}
}

// Add creates a shortened URL. It retries on token conflicts up to maxGeneratedTokenConflictRetry times.
func (s *Service) Add(ctx context.Context, url string) (string, error) {
	ctx, span := telemetry.Tracer("url-shortener/usecase/url").Start(ctx, "Service.Add")
	defer span.End()

	start := time.Now()
	s.addCounter.Add(ctx, 1)

	log := intLogger.WithContext(ctx, s.logger).WithField("url", url)
	log.Info("shortening URL")

	span.SetAttributes(attribute.String("url.original", url))

	var lastErr error
	for i := 0; i < maxGeneratedTokenConflictRetry; i++ {
		identifier, err := s.idProvider.GetNextID(ctx)
		if err != nil {
			s.addErrCounter.Add(ctx, 1)
			span.RecordError(err)
			span.SetStatus(codes.Error, "failed to generate ID")
			log.WithError(err).Error("failed to generate next ID")
			s.addDuration.Record(ctx, float64(time.Since(start).Milliseconds()))
			return "", fmt.Errorf("generating next ID: %w", err)
		}

		tok := s.tokenGenerator.GetToken(identifier)
		span.SetAttributes(attribute.String("url.token", tok))

		lastErr = s.repository.Save(ctx, &domain.URL{
			Path:  url,
			Token: tok,
		})

		if lastErr == nil {
			span.SetStatus(codes.Ok, "URL shortened successfully")
			log.WithField("token", tok).Info("URL shortened successfully")
			s.addDuration.Record(ctx, float64(time.Since(start).Milliseconds()))
			return tok, nil
		}

		if !errors.Is(lastErr, intErr.RepositoryDuplicateTokenErr) {
			s.addErrCounter.Add(ctx, 1)
			span.RecordError(lastErr)
			span.SetStatus(codes.Error, "failed to save URL")
			log.WithError(lastErr).Error("failed to save URL")
			s.addDuration.Record(ctx, float64(time.Since(start).Milliseconds()))
			return "", lastErr
		}

		// duplicate token — retry with a new ID
		log.WithField("retry", i+1).Warn("duplicate token conflict, retrying")
	}

	s.addErrCounter.Add(ctx, 1)
	span.RecordError(lastErr)
	span.SetStatus(codes.Error, "max retries exceeded")
	log.WithError(lastErr).Error("failed to add URL after max retries")
	s.addDuration.Record(ctx, float64(time.Since(start).Milliseconds()))
	return "", fmt.Errorf("failed to add URL after %d retries: %w", maxGeneratedTokenConflictRetry, lastErr)
}

// Delete removes a shortened URL by token.
func (s *Service) Delete(ctx context.Context, token string) error {
	ctx, span := telemetry.Tracer("url-shortener/usecase/url").Start(ctx, "Service.Delete")
	defer span.End()

	start := time.Now()
	s.deleteCounter.Add(ctx, 1)

	log := intLogger.WithContext(ctx, s.logger).WithField("token", token)
	log.Info("deleting URL")

	span.SetAttributes(attribute.String("url.token", token))

	err := s.repository.Delete(ctx, token)
	if err != nil {
		s.deleteErrCounter.Add(ctx, 1)
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to delete URL")
		log.WithError(err).Error("failed to delete URL")
		s.deleteDuration.Record(ctx, float64(time.Since(start).Milliseconds()))
		return err
	}

	span.SetStatus(codes.Ok, "URL deleted successfully")
	log.Info("URL deleted successfully")
	s.deleteDuration.Record(ctx, float64(time.Since(start).Milliseconds()))
	return nil
}

// Fetch retrieves a shortened URL by token.
func (s *Service) Fetch(ctx context.Context, token string) (*domain.URL, error) {
	ctx, span := telemetry.Tracer("url-shortener/usecase/url").Start(ctx, "Service.Fetch")
	defer span.End()

	start := time.Now()
	s.fetchCounter.Add(ctx, 1)

	log := intLogger.WithContext(ctx, s.logger).WithField("token", token)
	log.Debug("fetching URL")

	span.SetAttributes(attribute.String("url.token", token))

	result, err := s.repository.Fetch(ctx, token)
	if err != nil {
		s.fetchErrCounter.Add(ctx, 1)
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to fetch URL")
		log.WithError(err).Error("failed to fetch URL")
		s.fetchDuration.Record(ctx, float64(time.Since(start).Milliseconds()))
		return nil, err
	}

	span.SetStatus(codes.Ok, "URL fetched successfully")
	span.SetAttributes(attribute.String("url.original", result.Path))
	log.WithField("url", result.Path).Debug("URL fetched successfully")
	s.fetchDuration.Record(ctx, float64(time.Since(start).Milliseconds()))
	return result, nil
}