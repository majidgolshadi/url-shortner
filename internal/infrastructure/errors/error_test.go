package errors

import (
	"errors"
	"fmt"
	"testing"

	pkgerrors "github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
)

func TestErrorType_Is(t *testing.T) {
	tests := map[string]struct {
		err    func() error
		target func() error
		equal  bool
	}{
		"simple equal error type": {
			err: func() error {
				return CoordinatorDataInvalidVersionErr
			},
			target: func() error {
				return CoordinatorDataInvalidVersionErr
			},
			equal: true,
		},
		"target error wrapped the same error type": {
			err: func() error {
				return CoordinatorDataInvalidVersionErr
			},
			target: func() error {
				return pkgerrors.Wrap(CoordinatorDataInvalidVersionErr, "test message")
			},
			equal: true,
		},
		"target error twice wrapped the same error type": {
			err: func() error {
				return CoordinatorDataInvalidVersionErr
			},
			target: func() error {
				return pkgerrors.Wrap(pkgerrors.Wrap(CoordinatorDataInvalidVersionErr, "first wrapper"), "second wrapper")
			},
			equal: true,
		},
		"target error wrapped in an message via fmt.errorf method": {
			err: func() error {
				return CoordinatorDataInvalidVersionErr
			},
			target: func() error {
				return fmt.Errorf("%w sample message", CoordinatorDataInvalidVersionErr)
			},
			equal: true,
		},
		"target error wrapped in an message via fmt.errorf method twice": {
			err: func() error {
				return CoordinatorDataInvalidVersionErr
			},
			target: func() error {
				return fmt.Errorf("%w second", fmt.Errorf("%w first", CoordinatorDataInvalidVersionErr))
			},
			equal: true,
		},
		"same error type wrapped differently": {
			err: func() error {
				return pkgerrors.Wrap(CoordinatorDataInvalidVersionErr, "wrapper msg one")
			},
			target: func() error {
				return pkgerrors.Wrap(pkgerrors.Wrap(CoordinatorDataInvalidVersionErr, "wrapper msg two"), "wrapper msg three")
			},
			equal: true,
		},
		"nil target error": {
			err: func() error {
				return CoordinatorDataInvalidVersionErr
			},
			target: func() error {
				return nil
			},
			equal: false,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			err := test.err()
			target := test.target()
			assert.Equal(t, test.equal, errors.Is(err, target))
		})
	}
}
