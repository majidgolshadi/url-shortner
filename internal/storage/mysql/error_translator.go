package mysql

import (
	"github.com/go-sql-driver/mysql"

	"github.com/majidgolshadi/url-shortner/internal/infrastructure/errors"
)

var mysqlErrCodes = map[uint16]error{
	1062: errors.RepositoryDuplicateTokenErr,
}

// influenced by https://github.com/go-gorm/mysql/blob/master/error_translator.go
// MySQL error number list reference https://dev.mysql.com/doc/mysql-errors/8.0/en/server-error-reference.html
func translateMysqlError(err error) error {
	if mysqlErr, ok := err.(*mysql.MySQLError); ok {
		if translatedError, found := mysqlErrCodes[mysqlErr.Number]; found {
			return translatedError
		}
	}

	return err
}
