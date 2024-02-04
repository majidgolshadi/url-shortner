package sql

import (
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/pkg/errors"
)

// DBConfig db config
type (
	DBConfig struct {
		DSN             string
		MaxOpenConns    int
		ConnMaxLifetime time.Duration
		// ServiceName is a service name for trace
		ServiceName string
	}

	DBFactory struct {
		config *DBConfig
	}
)

func NewDBFactory(cfg *DBConfig) *DBFactory {
	return &DBFactory{
		config: cfg,
	}
}

// CreateDB creates db instance
func (f *DBFactory) CreateDB() (*sqlx.DB, error) {
	if f.config == nil {
		return nil, errors.New("db config is empty")
	}

	// TODO: wrap the connection for tracing purpose
	db, err := sqlx.Open("mysql", f.config.DSN)
	if err != nil {
		return nil, err
	}

	db.SetMaxOpenConns(f.config.MaxOpenConns)
	db.SetConnMaxLifetime(f.config.ConnMaxLifetime)

	return db, nil
}
