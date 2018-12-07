package url_shortner

type CoordinatorMock struct {
	Offset int
	Max int
	CommitCalled bool
}

func (c *CoordinatorMock) getRestoreRange() (offset int, end int, err error) {
	return c.Offset, c.Max, nil
}

func (c *CoordinatorMock) getNextRange() (start int, end int, err error) {
	rangNum := 100
	start = (rangNum * 2)+ c.Max + 1
	end = start + rangNum

	return
}

func (c *CoordinatorMock) commit(counter int, end int) error {
	c.CommitCalled = true
	return nil
}
