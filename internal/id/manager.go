package id

import (
	"context"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	otelmetric "go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"

	"github.com/majidgolshadi/url-shortner/internal/domain"
	intErr "github.com/majidgolshadi/url-shortner/internal/infrastructure/errors"
	intLogger "github.com/majidgolshadi/url-shortner/internal/infrastructure/logger"
	"github.com/majidgolshadi/url-shortner/internal/infrastructure/telemetry"
	"github.com/pkg/errors"
)

type Manager struct {
	rangeManager RangeManager
	logger       *logrus.Entry

	lastID        uint
	reservedRange domain.Range
	mux           sync.Mutex

	// metrics
	idGeneratedCounter otelmetric.Int64Counter
	idGenerateDuration otelmetric.Float64Histogram
	rangeRemaining     otelmetric.Int64UpDownCounter
}

func NewManager(ctx context.Context, rangeMng RangeManager, logger *logrus.Entry) (*Manager, error) {
	log := intLogger.WithContext(ctx, logger).WithField("component", "id_manager")
	log.Info("initializing ID manager")

	rng, err := rangeMng.getCurrentRange(ctx)

	// fresh run
	if errors.Is(err, intErr.RangeManagerNoReservedRangeErr) {
		log.Info("no reserved range found, requesting new range")
		rng, err = rangeMng.getNextIDRange(ctx)
	}

	if err != nil {
		log.WithError(err).Error("failed to initialize ID manager")
		return nil, err
	}

	meter := telemetry.Meter("url-shortener/id")

	idGeneratedCounter, _ := meter.Int64Counter("id.generated.total",
		otelmetric.WithDescription("Total number of IDs generated"))
	idGenerateDuration, _ := meter.Float64Histogram("id.generate.duration_ms",
		otelmetric.WithDescription("Duration of ID generation in milliseconds"),
		otelmetric.WithUnit("ms"))
	rangeRemaining, _ := meter.Int64UpDownCounter("id.range.remaining",
		otelmetric.WithDescription("Remaining IDs in the current range"))

	m := &Manager{
		rangeManager: rangeMng,
		logger:       logger,
		lastID:       rng.Start,
		reservedRange: domain.Range{
			Start: rng.Start,
			End:   rng.End,
		},
		idGeneratedCounter: idGeneratedCounter,
		idGenerateDuration: idGenerateDuration,
		rangeRemaining:     rangeRemaining,
	}

	// Set initial remaining count
	remaining := int64(rng.End - rng.Start)
	m.rangeRemaining.Add(ctx, remaining)

	log.WithFields(logrus.Fields{
		"range_start": rng.Start,
		"range_end":   rng.End,
		"remaining":   remaining,
	}).Info("ID manager initialized")

	return m, nil
}

// GetLastID returns the latest used ID
func (m *Manager) GetLastID() uint {
	m.mux.Lock()
	defer m.mux.Unlock()

	return m.lastID
}

// GetNextID retrieves the subsequent integer ID.
// In case the reserved range is entirely consumed, it prompts the range manager to reserve a new range, which is then put into use.
func (m *Manager) GetNextID(ctx context.Context) (uint, error) {
	ctx, span := telemetry.Tracer("url-shortener/id").Start(ctx, "Manager.GetNextID")
	defer span.End()

	start := time.Now()

	m.mux.Lock()
	defer m.mux.Unlock()

	log := intLogger.WithContext(ctx, m.logger).WithField("component", "id_manager")

	m.lastID++
	if m.lastID > m.reservedRange.End {
		log.WithFields(logrus.Fields{
			"last_id":   m.lastID,
			"range_end": m.reservedRange.End,
		}).Info("range exhausted, requesting new range")

		span.AddEvent("range_exhausted", trace.WithAttributes(
			attribute.Int("last_id", int(m.lastID)),
			attribute.Int("range_end", int(m.reservedRange.End)),
		))

		takenRange, err := m.rangeManager.getNextIDRange(ctx)
		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, "failed to get next ID range")
			log.WithError(err).Error("failed to get next ID range")
			m.idGenerateDuration.Record(ctx, float64(time.Since(start).Milliseconds()))
			return 0, err
		}

		// Reset remaining counter for the new range
		newRemaining := int64(takenRange.End - takenRange.Start)
		m.rangeRemaining.Add(ctx, newRemaining)

		m.reservedRange = takenRange
		m.lastID = m.reservedRange.Start

		log.WithFields(logrus.Fields{
			"new_range_start": takenRange.Start,
			"new_range_end":   takenRange.End,
			"remaining":       newRemaining,
		}).Info("new range acquired")
	}

	m.idGeneratedCounter.Add(ctx, 1)
	m.rangeRemaining.Add(ctx, -1)

	span.SetAttributes(attribute.Int("id.value", int(m.lastID)))
	span.SetStatus(codes.Ok, "ID generated")
	m.idGenerateDuration.Record(ctx, float64(time.Since(start).Milliseconds()))

	return m.lastID, nil
}