package id

import "context"

type Range struct {
	Start uint
	End   uint
}

type RangeManager interface {
	getCurrentRange(ctx context.Context) (Range, error)
	getNextIDRange(ctx context.Context) (Range, error)
}
