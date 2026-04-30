package errors

const (
	// CoordinatorDataInvalidVersionErr ...
	CoordinatorDataInvalidVersionErr = errorStr("invalid data version")

	// CoordinatorNoReservedRangeErr ...
	CoordinatorNoReservedRangeErr = errorStr("no range has been reserved")

	// CoordinatorRangeFragmentationErr ...
	CoordinatorRangeFragmentationErr = errorStr("range fragmentation error")

	// RepositoryDuplicateTokenErr ...
	RepositoryDuplicateTokenErr = errorStr("duplicate token error")

	// RangeManagerNoReservedRangeErr ...
	RangeManagerNoReservedRangeErr = errorStr("node hasn't reserved any range yet")

	// BudgetExceededErr is returned when a customer has reached their URL creation limit.
	BudgetExceededErr = errorStr("budget exceeded")

	// NotURLOwnerErr is returned when a customer tries to access a URL they do not own.
	NotURLOwnerErr = errorStr("not url owner")
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
