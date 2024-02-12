package logger

import (
	"os"

	"github.com/sirupsen/logrus"
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
