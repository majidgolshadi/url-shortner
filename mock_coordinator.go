package url_shortner

type MockCoordinator struct {
	Offset int
	Max int
	CommitCalled bool
}

func (c *MockCoordinator) getRestoreRange() (offset int, end int, err error) {
	return c.Offset, c.Max, nil
}

func (c *MockCoordinator) getNextRange() (start int, end int, err error) {
	rangNum := 100
	start = (rangNum * 2)+ c.Max + 1
	end = start + rangNum

	return
}

func (c *MockCoordinator) commit(counter int, end int) error {
	c.CommitCalled = true
	return nil
}
