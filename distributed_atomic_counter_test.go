package url_shortner

import "testing"

func TestGetNextNumberInRange(t *testing.T) {

	coordinator := &MockCoordinator{
		Offset: 1,
		Max:    100,
	}

	counter, _ := NewDistributedAtomicCounter(coordinator)

	if counter.next() != 2 {
		t.Fail()
	}

	if !coordinator.CommitCalled {
		t.Fail()
	}
}

func TestGetNextNumberOutOfRange(t *testing.T) {

	coordinator := &MockCoordinator{
		Offset: 100,
		Max:    100,
	}

	counter, _ := NewDistributedAtomicCounter(coordinator)

	if counter.next() != 301 {
		t.Fail()
	}

	if !coordinator.CommitCalled {
		t.Fail()
	}
}
