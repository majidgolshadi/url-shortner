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

// errorStr is a typed string constant so errors.Is() comparisons work without allocation
// and cannot accidentally match unrelated string errors.
type errorStr string

func (err errorStr) Error() string {
	return string(err)
}

// Is unwraps both pkg/errors and fmt.Errorf chains so callers can use errors.Is()
// regardless of how the error was wrapped.
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
