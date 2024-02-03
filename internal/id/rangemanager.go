package id

type Range struct {
	Min uint
	Max uint
}
type RangeManager interface {
	getCurrentRange() (Range, error)
	getNextIDRange() (Range, error)
}
