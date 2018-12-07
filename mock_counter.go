package url_shortner

type MockCounter struct {
	Offset int
}

func (c *MockCounter) next() int {
	return c.Offset + 1
}
