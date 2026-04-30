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

type customerSqlRow struct {
	ID        string `db:"id"`
	AuthToken string `db:"auth_token"`
}

type customerRepository struct {
	db     *sqlx.DB
	logger *logrus.Entry

	queryDuration otelmetric.Float64Histogram
	queryErrors   otelmetric.Int64Counter
}

func NewCustomerRepository(db *sqlx.DB, logger *logrus.Entry) storage.CustomerRepository {
	meter := telemetry.Meter("url-shortener/storage/mysql/customer")

	queryDuration, _ := meter.Float64Histogram("db.customer.query.duration_ms",
		otelmetric.WithDescription("Duration of customer database queries in milliseconds"),
		otelmetric.WithUnit("ms"))
	queryErrors, _ := meter.Int64Counter("db.customer.query.errors",
		otelmetric.WithDescription("Total number of customer database query errors"))

	return &customerRepository{
		db:            db,
		logger:        logger,
		queryDuration: queryDuration,
		queryErrors:   queryErrors,
	}
}

func (r *customerRepository) Save(ctx context.Context, customer *domain.Customer) error {
	ctx, span := telemetry.Tracer("url-shortener/storage/mysql/customer").Start(ctx, "CustomerRepository.Save")
	defer span.End()

	start := time.Now()
	log := intLogger.WithContext(ctx, r.logger).WithField("operation", "save_customer")

	span.SetAttributes(
		attribute.String("db.system", "mysql"),
		attribute.String("db.operation", "INSERT"),
		attribute.String("db.table", "customer"),
	)

	log.Debug("saving customer to database")

	_, err := r.db.ExecContext(ctx, `INSERT INTO customer(id, auth_token) VALUES(?, ?);`, customer.ID, customer.AuthToken)

	duration := float64(time.Since(start).Milliseconds())
	r.queryDuration.Record(ctx, duration, otelmetric.WithAttributes(
		attribute.String("db.operation", "INSERT"),
	))

	if err != nil {
		r.queryErrors.Add(ctx, 1, otelmetric.WithAttributes(
			attribute.String("db.operation", "INSERT"),
		))
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to save customer")
		log.WithError(err).Error("failed to save customer to database")
		return err
	}

	span.SetStatus(codes.Ok, "customer saved successfully")
	log.Debug("customer saved to database successfully")
	return nil
}

func (r *customerRepository) FindByAuthToken(ctx context.Context, authToken string) (*domain.Customer, error) {
	ctx, span := telemetry.Tracer("url-shortener/storage/mysql/customer").Start(ctx, "CustomerRepository.FindByAuthToken")
	defer span.End()

	start := time.Now()
	log := intLogger.WithContext(ctx, r.logger).WithField("operation", "find_customer_by_auth_token")

	span.SetAttributes(
		attribute.String("db.system", "mysql"),
		attribute.String("db.operation", "SELECT"),
		attribute.String("db.table", "customer"),
	)

	log.Debug("finding customer by auth token")

	row := customerSqlRow{}
	err := r.db.GetContext(ctx, &row, `SELECT id, auth_token FROM customer WHERE auth_token = ?;`, authToken)

	duration := float64(time.Since(start).Milliseconds())
	r.queryDuration.Record(ctx, duration, otelmetric.WithAttributes(
		attribute.String("db.operation", "SELECT"),
	))

	if err != nil {
		r.queryErrors.Add(ctx, 1, otelmetric.WithAttributes(
			attribute.String("db.operation", "SELECT"),
		))
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to find customer")
		log.WithError(err).Error("failed to find customer by auth token")
		return nil, err
	}

	span.SetStatus(codes.Ok, "customer found successfully")
	log.Debug("customer found successfully")
	return &domain.Customer{
		ID:        row.ID,
		AuthToken: row.AuthToken,
	}, nil
}
