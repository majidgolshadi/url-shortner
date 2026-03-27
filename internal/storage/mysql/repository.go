package mysql

import (
	"context"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/sirupsen/logrus"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	otelmetric "go.opentelemetry.io/otel/metric"

	"github.com/majidgolshadi/url-shortner/internal/domain"
	intLogger "github.com/majidgolshadi/url-shortner/internal/infrastructure/logger"
	"github.com/majidgolshadi/url-shortner/internal/infrastructure/telemetry"
	"github.com/majidgolshadi/url-shortner/internal/storage"
)

type sqlRow struct {
	Token string `db:"token"` // primary key
	URL   string `db:"url"`
}
type repository struct {
	db     *sqlx.DB
	logger *logrus.Entry

	// metrics
	queryDuration otelmetric.Float64Histogram
	queryErrors   otelmetric.Int64Counter
}

func NewRepository(db *sqlx.DB, logger *logrus.Entry) storage.Repository {
	meter := telemetry.Meter("url-shortener/storage/mysql")

	queryDuration, _ := meter.Float64Histogram("db.query.duration_ms",
		otelmetric.WithDescription("Duration of database queries in milliseconds"),
		otelmetric.WithUnit("ms"))
	queryErrors, _ := meter.Int64Counter("db.query.errors",
		otelmetric.WithDescription("Total number of database query errors"))

	return &repository{
		db:            db,
		logger:        logger,
		queryDuration: queryDuration,
		queryErrors:   queryErrors,
	}
}

func (r *repository) Save(ctx context.Context, url *domain.URL) error {
	ctx, span := telemetry.Tracer("url-shortener/storage/mysql").Start(ctx, "Repository.Save")
	defer span.End()

	start := time.Now()
	log := intLogger.WithContext(ctx, r.logger).WithFields(logrus.Fields{
		"operation": "save",
		"token":     url.Token,
	})

	span.SetAttributes(
		attribute.String("db.system", "mysql"),
		attribute.String("db.operation", "INSERT"),
		attribute.String("db.table", "url_token"),
	)

	log.Debug("saving URL to database")

	sql := `INSERT INTO url_token(token, url) VALUES(?, ?);`
	_, err := r.db.ExecContext(ctx, sql, url.Token, url.Path)

	duration := float64(time.Since(start).Milliseconds())
	r.queryDuration.Record(ctx, duration, otelmetric.WithAttributes(
		attribute.String("db.operation", "INSERT"),
	))

	if err != nil {
		translatedErr := translateMysqlError(err)
		r.queryErrors.Add(ctx, 1, otelmetric.WithAttributes(
			attribute.String("db.operation", "INSERT"),
		))
		span.RecordError(translatedErr)
		span.SetStatus(codes.Error, "failed to save URL")
		log.WithError(translatedErr).Error("failed to save URL to database")
		return translatedErr
	}

	span.SetStatus(codes.Ok, "URL saved successfully")
	log.Debug("URL saved to database successfully")
	return nil
}

func (r *repository) Delete(ctx context.Context, token string) error {
	ctx, span := telemetry.Tracer("url-shortener/storage/mysql").Start(ctx, "Repository.Delete")
	defer span.End()

	start := time.Now()
	log := intLogger.WithContext(ctx, r.logger).WithFields(logrus.Fields{
		"operation": "delete",
		"token":     token,
	})

	span.SetAttributes(
		attribute.String("db.system", "mysql"),
		attribute.String("db.operation", "DELETE"),
		attribute.String("db.table", "url_token"),
	)

	log.Debug("deleting URL from database")

	_, err := r.db.ExecContext(ctx, `DELETE FROM url_token WHERE token=?;`, token)

	duration := float64(time.Since(start).Milliseconds())
	r.queryDuration.Record(ctx, duration, otelmetric.WithAttributes(
		attribute.String("db.operation", "DELETE"),
	))

	if err != nil {
		r.queryErrors.Add(ctx, 1, otelmetric.WithAttributes(
			attribute.String("db.operation", "DELETE"),
		))
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to delete URL")
		log.WithError(err).Error("failed to delete URL from database")
		return err
	}

	span.SetStatus(codes.Ok, "URL deleted successfully")
	log.Debug("URL deleted from database successfully")
	return nil
}

func (r *repository) Fetch(ctx context.Context, token string) (*domain.URL, error) {
	ctx, span := telemetry.Tracer("url-shortener/storage/mysql").Start(ctx, "Repository.Fetch")
	defer span.End()

	start := time.Now()
	log := intLogger.WithContext(ctx, r.logger).WithFields(logrus.Fields{
		"operation": "fetch",
		"token":     token,
	})

	span.SetAttributes(
		attribute.String("db.system", "mysql"),
		attribute.String("db.operation", "SELECT"),
		attribute.String("db.table", "url_token"),
	)

	log.Debug("fetching URL from database")

	row := sqlRow{}
	err := r.db.GetContext(ctx, &row, `SELECT token, url FROM url_token WHERE token = ?;`, token)

	duration := float64(time.Since(start).Milliseconds())
	r.queryDuration.Record(ctx, duration, otelmetric.WithAttributes(
		attribute.String("db.operation", "SELECT"),
	))

	if err != nil {
		r.queryErrors.Add(ctx, 1, otelmetric.WithAttributes(
			attribute.String("db.operation", "SELECT"),
		))
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to fetch URL")
		log.WithError(err).Error("failed to fetch URL from database")
		return nil, err
	}

	span.SetStatus(codes.Ok, "URL fetched successfully")
	log.Debug("URL fetched from database successfully")
	return &domain.URL{
		Path:  row.URL,
		Token: row.Token,
	}, nil
}

func (r *repository) HealthCheck(ctx context.Context) (bool, interface{}) {
	err := r.db.PingContext(ctx)

	if err != nil {
		return false, struct {
			Status   bool
			ErrorMsg string
		}{
			Status:   false,
			ErrorMsg: err.Error(),
		}
	}

	return true, struct {
		Status bool
	}{
		Status: true,
	}
}