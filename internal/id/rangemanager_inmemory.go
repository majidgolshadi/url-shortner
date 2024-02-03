package id

type inMemory struct {
	startID uint
}

func NewInMemoryRangeManager(startID uint) RangeManager {
	return &inMemory{
		startID: startID,
	}
}

func (c *inMemory) getCurrentRange() (Range, error) {
	return Range{
		Min: c.startID,
		Max: ^uint(0),
	}, nil
}

func (c *inMemory) getNextIDRange() (Range, error) {
	return Range{
		Min: c.startID,
		Max: ^uint(0),
	}, nil
}
