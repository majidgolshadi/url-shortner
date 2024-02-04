package errors

const (
	// CoordinatorDataInvalidVersionErr ...
	CoordinatorDataInvalidVersionErr = errorStr("coordinator: invalid data version")

	// CoordinatorTakeNextFreeRangeErr ...
	CoordinatorTakeNextFreeRangeErr = errorStr("coordinator: take next free range error")

	// RepositoryDuplicateTokenErr ...
	RepositoryDuplicateTokenErr = errorStr("repository: duplicate token error")
)

type errorStr string

func (err errorStr) Error() string {
	return string(err)
}

func (err errorStr) Is(target error) bool {
	targetError, ok := target.(errorStr)
	if ok {
		if err == targetError {
			return true
		}
	}
	// in case error has been encapsulated by github.com/pkg/errors package
	type causer interface {
		Cause() error
	}
	cause, ok := target.(causer)
	if ok {
		return err.Is(cause.Cause())
	}

	// in case error has been encapsulated by fmt.Errorf
	type unwrap interface {
		Unwrap() error
	}
	wrapped, ok := target.(unwrap)
	if ok {
		return err.Is(wrapped.Unwrap())
	}
	return false
}
