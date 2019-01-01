package url_shortner

type coordinator interface {
	getRestoreRange() (offset int, end int, err error)
	getNextRange() (start int, end int, err error)
	commit(offset int, end int) error
}
