package mysql

import (
	"github.com/go-sql-driver/mysql"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"testing"

	intErr "github.com/majidgolshadi/url-shortner/internal/infrastructure/errors"
)

func TestTranslateMysqlError(t *testing.T) {
	unknownErr := errors.New("unknown error")

	tests := map[string]struct {
		error         error
		expectedError error
	}{
		"data invalid version error": {
			error: &mysql.MySQLError{
				Number: 45000,
			},
			expectedError: intErr.CoordinatorDataInvalidVersionErr,
		},
		"duplicate token error": {
			error: &mysql.MySQLError{
				Number: 1062,
			},
			expectedError: intErr.RepositoryDuplicateTokenErr,
		},
		"unknown mysql error": {
			error: &mysql.MySQLError{
				Number: 12,
			},
			expectedError: &mysql.MySQLError{
				Number: 12,
			},
		},
		"unknown error": {
			error:         unknownErr,
			expectedError: unknownErr,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			result := translateMysqlError(test.error)
			assert.Equal(t, test.expectedError, result)
		})
	}
}
