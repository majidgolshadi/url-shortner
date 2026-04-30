package mysql

import (
	"context"
	"database/sql"
	"strconv"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/sirupsen/logrus"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	otelmetric "go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"
	"github.com/pkg/errors"

	"github.com/majidgolshadi/url-shortner/internal/domain"
	intErr "github.com/majidgolshadi/url-shortner/internal/infrastructure/errors"
	intLogger "github.com/majidgolshadi/url-shortner/internal/infrastructure/logger"
	"github.com/majidgolshadi/url-shortner/internal/infrastructure/telemetry"
	"github.com/majidgolshadi/url-shortner/internal/storage"
)

// lastReservedIDKey is the single row in nodes_coordination_keys that tracks
// the global high-water mark of allocated ID ranges across all nodes.
const lastReservedIDKey = "last_reserved_id"

type coordinator struct {
	db     *sqlx.DB
	logger *logrus.Entry

	// metrics
	queryDuration otelmetric.Float64Histogram
	queryErrors   otelmetric.Int64Counter
}

type (
	nodesCoordinationKeyRow struct {
		KeyID   string `db:"key_id"`
		Value   string `db:"value"`
		Version int    `db:"version"`
	}
	nodeRangeJournalRow struct {
		NodeID string `db:"node_id"`
		Start  uint   `db:"start"`
		End    uint   `db:"end"`
	}
)

func NewCoordinator(db *sqlx.DB, logger *logrus.Entry) storage.Coordinator {
	meter := telemetry.Meter("url-shortener/storage/mysql/coordinator")

	queryDuration, _ := meter.Float64Histogram("db.coordinator.query.duration_ms",
		otelmetric.WithDescription("Duration of coordinator database queries in milliseconds"),
		otelmetric.WithUnit("ms"))
	queryErrors, _ := meter.Int64Counter("db.coordinator.query.errors",
		otelmetric.WithDescription("Total number of coordinator database query errors"))

	return &coordinator{
		db:            db,
		logger:        logger,
		queryDuration: queryDuration,
		queryErrors:   queryErrors,
	}
}

// GetNodeReservedRange get requested node latest reserved range
func (c *coordinator) GetNodeReservedRange(ctx context.Context, nodeID string) (*domain.Range, error) {
	ctx, span := telemetry.Tracer("url-shortener/storage/mysql/coordinator").Start(ctx, "Coordinator.GetNodeReservedRange")
	defer span.End()

	start := time.Now()
	log := intLogger.WithContext(ctx, c.logger).WithFields(logrus.Fields{
		"operation": "get_node_reserved_range",
		"node_id":   nodeID,
	})

	span.SetAttributes(
		attribute.String("db.system", "mysql"),
		attribute.String("db.operation", "SELECT"),
		attribute.String("db.table", "node_range_journal"),
		attribute.String("node.id", nodeID),
	)

	log.Debug("fetching node reserved range")

	var row nodeRangeJournalRow
	err := c.db.GetContext(ctx, &row, "SELECT start, end FROM node_range_journal WHERE node_id=?;", nodeID)

	duration := float64(time.Since(start).Milliseconds())
	c.queryDuration.Record(ctx, duration, otelmetric.WithAttributes(
		attribute.String("db.operation", "SELECT"),
	))

	if err != nil {
		c.queryErrors.Add(ctx, 1, otelmetric.WithAttributes(
			attribute.String("db.operation", "SELECT"),
		))
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to get node reserved range")
		log.WithError(err).Error("failed to get node reserved range")
		return nil, err
	}

	span.SetStatus(codes.Ok, "node reserved range fetched")
	span.SetAttributes(
		attribute.Int("range.start", int(row.Start)),
		attribute.Int("range.end", int(row.End)),
	)
	log.WithFields(logrus.Fields{
		"range_start": row.Start,
		"range_end":   row.End,
	}).Debug("node reserved range fetched")

	return &domain.Range{
		Start: row.Start,
		End:   row.End,
	}, nil
}

// GetLatestReservedRange returns the  latest reserved range with its version
func (c *coordinator) GetLatestReservedRange(ctx context.Context) (lastReservedNumber uint, version int, err error) {
	ctx, span := telemetry.Tracer("url-shortener/storage/mysql/coordinator").Start(ctx, "Coordinator.GetLatestReservedRange")
	defer span.End()

	start := time.Now()
	log := intLogger.WithContext(ctx, c.logger).WithField("operation", "get_latest_reserved_range")

	span.SetAttributes(
		attribute.String("db.system", "mysql"),
		attribute.String("db.operation", "SELECT"),
		attribute.String("db.table", "nodes_coordination_keys"),
	)

	log.Debug("fetching latest reserved range")

	var row nodesCoordinationKeyRow
	err = c.db.GetContext(ctx, &row, "SELECT value, version FROM nodes_coordination_keys WHERE key_id = ?;", lastReservedIDKey)

	duration := float64(time.Since(start).Milliseconds())
	c.queryDuration.Record(ctx, duration, otelmetric.WithAttributes(
		attribute.String("db.operation", "SELECT"),
	))

	if errors.Is(err, sql.ErrNoRows) {
		span.SetStatus(codes.Ok, "no reserved range found")
		log.Debug("no reserved range found")
		return 0, 0, intErr.CoordinatorNoReservedRangeErr
	}

	if err != nil {
		c.queryErrors.Add(ctx, 1, otelmetric.WithAttributes(
			attribute.String("db.operation", "SELECT"),
		))
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to get latest reserved range")
		log.WithError(err).Error("failed to get latest reserved range")
		return 0, 0, err
	}

	end, strConvErr := strconv.Atoi(row.Value)
	if strConvErr != nil {
		span.RecordError(strConvErr)
		span.SetStatus(codes.Error, "failed to parse reserved range value")
		log.WithError(strConvErr).Error("failed to parse reserved range value")
		return 0, 0, strConvErr
	}

	span.SetStatus(codes.Ok, "latest reserved range fetched")
	span.SetAttributes(
		attribute.Int("range.last_reserved", end),
		attribute.Int("range.version", row.Version),
	)
	log.WithFields(logrus.Fields{
		"last_reserved": end,
		"version":       row.Version,
	}).Debug("latest reserved range fetched")

	return uint(end), row.Version, nil
}

// TakeFreeRange take requested range otherwise throw an error
// errors:
// CoordinatorRangeFragmentationErr in case there is a gap between requested range start point with the latest reserved end point
// CoordinatorDataInvalidVersionErr invalid version number
// mysql errors
func (c *coordinator) TakeFreeRange(ctx context.Context, nodeID string, requestedRange domain.Range, version int) error {
	ctx, span := telemetry.Tracer("url-shortener/storage/mysql/coordinator").Start(ctx, "Coordinator.TakeFreeRange")
	defer span.End()

	log := intLogger.WithContext(ctx, c.logger).WithFields(logrus.Fields{
		"operation":   "take_free_range",
		"node_id":     nodeID,
		"range_start": requestedRange.Start,
		"range_end":   requestedRange.End,
		"version":     version,
	})

	span.SetAttributes(
		attribute.String("db.system", "mysql"),
		attribute.String("node.id", nodeID),
		attribute.Int("range.start", int(requestedRange.Start)),
		attribute.Int("range.end", int(requestedRange.End)),
		attribute.Int("range.version", version),
	)

	log.Info("taking free range")

	var row nodesCoordinationKeyRow
	err := c.db.GetContext(ctx, &row, "SELECT value, version FROM nodes_coordination_keys WHERE key_id = ?;", lastReservedIDKey)
	if errors.Is(err, sql.ErrNoRows) {
		// no need to check the initial version
		return c.setRange(ctx, nodeID, requestedRange, version, log)
	}

	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to get coordination keys")
		log.WithError(err).Error("failed to get coordination keys")
		return err
	}

	latestReservedRange, strConvErr := strconv.Atoi(row.Value)
	if strConvErr != nil {
		span.RecordError(strConvErr)
		span.SetStatus(codes.Error, "failed to parse latest reserved range")
		log.WithError(strConvErr).Error("failed to parse latest reserved range")
		return strConvErr
	}

	// ensure no gap exists between reserved ranges
	if requestedRange.Start != uint(latestReservedRange+1) {
		span.RecordError(intErr.CoordinatorRangeFragmentationErr)
		span.SetStatus(codes.Error, "range fragmentation detected")
		log.WithFields(logrus.Fields{
			"latest_reserved": latestReservedRange,
			"requested_start": requestedRange.Start,
		}).Warn("range fragmentation detected")
		return intErr.CoordinatorRangeFragmentationErr
	}

	return c.setRange(ctx, nodeID, requestedRange, version, log)
}

// setRange uses a transaction to atomically update the high-water mark and append a journal entry.
// The journal (node_range_journal) provides an audit trail of which node held which range.
// The version field in nodes_coordination_keys acts as an optimistic lock:
// only the node that read version N can write version N+1, preventing split-brain range allocation.
func (c *coordinator) setRange(ctx context.Context, nodeID string, requestedRange domain.Range, version int, log *logrus.Entry) error {
	span := trace.SpanFromContext(ctx)

	start := time.Now()

	tx, err := c.db.BeginTxx(ctx, nil)
	if err != nil {
		c.queryErrors.Add(ctx, 1, otelmetric.WithAttributes(
			attribute.String("db.operation", "BEGIN_TX"),
		))
		if span != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, "failed to begin transaction")
		}
		log.WithError(err).Error("failed to begin transaction")
		return err
	}
	// nolint:errcheck
	defer tx.Rollback()

	query := `INSERT INTO nodes_coordination_keys(key_id, value, version) VALUES (?,?,?)
		ON DUPLICATE KEY UPDATE value = VALUES(value), version = VALUES(version);`
	if _, err = tx.ExecContext(ctx, query, lastReservedIDKey, requestedRange.End, version); err != nil {
		translatedErr := translateMysqlError(err)
		c.queryErrors.Add(ctx, 1, otelmetric.WithAttributes(
			attribute.String("db.operation", "UPSERT"),
		))
		if span != nil {
			span.RecordError(translatedErr)
			span.SetStatus(codes.Error, "failed to upsert coordination key")
		}
		log.WithError(translatedErr).Error("failed to upsert coordination key")
		return translatedErr
	}

	_, err = tx.ExecContext(ctx, "INSERT INTO node_range_journal(node_id, start, end) VALUES (?,?,?);",
		nodeID, requestedRange.Start, requestedRange.End)
	if err != nil {
		c.queryErrors.Add(ctx, 1, otelmetric.WithAttributes(
			attribute.String("db.operation", "INSERT"),
		))
		if span != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, "failed to insert range journal")
		}
		log.WithError(err).Error("failed to insert range journal")
		return err
	}

	if err = tx.Commit(); err != nil {
		c.queryErrors.Add(ctx, 1, otelmetric.WithAttributes(
			attribute.String("db.operation", "COMMIT"),
		))
		if span != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, "failed to commit transaction")
		}
		log.WithError(err).Error("failed to commit transaction")
		return err
	}

	duration := float64(time.Since(start).Milliseconds())
	c.queryDuration.Record(ctx, duration, otelmetric.WithAttributes(
		attribute.String("db.operation", "SET_RANGE"),
	))

	if span != nil {
		span.SetStatus(codes.Ok, "range set successfully")
	}
	log.Info("range set successfully")
	return nil
}