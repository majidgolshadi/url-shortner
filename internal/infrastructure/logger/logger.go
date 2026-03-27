package logger

import (
	"context"
	"os"

	"github.com/sirupsen/logrus"
	"go.opentelemetry.io/otel/trace"
)

// TimestampFormat common timestamp format for the package
const TimestampFormat = "1990-02-04T13:01:01.000"

// NewLogger creates a new logrus entry
func NewLogger(logLevel string) *logrus.Entry {
	logger := newLogger()
	level, err := logrus.ParseLevel(logLevel)
	if err == nil {
		logger.Level = level
	}
	return logrus.NewEntry(logger)
}

func newLogger() *logrus.Logger {
	logger := logrus.New()
	logger.Out = os.Stdout
	logger.Level = logrus.InfoLevel
	logger.SetFormatter(&logrus.JSONFormatter{
		TimestampFormat: TimestampFormat,
		FieldMap: logrus.FieldMap{
			logrus.FieldKeyTime:  "timestamp",
			logrus.FieldKeyLevel: "level",
			logrus.FieldKeyMsg:   "message",
		},
	})
	return logger
}

// WithContext enriches the log entry with trace context (trace_id and span_id)
// extracted from the provided context. This enables log-trace correlation
// in observability backends like Grafana, Jaeger, or Datadog.
func WithContext(ctx context.Context, entry *logrus.Entry) *logrus.Entry {
	spanCtx := trace.SpanContextFromContext(ctx)
	if !spanCtx.IsValid() {
		return entry
	}

	return entry.WithFields(logrus.Fields{
		"trace_id": spanCtx.TraceID().String(),
		"span_id":  spanCtx.SpanID().String(),
	})
}