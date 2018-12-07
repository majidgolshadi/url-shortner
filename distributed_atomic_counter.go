package url_shortner

import "log"

type distributedAtomicCounter struct {
	offset    int
	max     int
	coordinator coordinator
}

func NewDistributedAtomicCounter(coordinator coordinator) (*distributedAtomicCounter, error) {
	offset, max, err := coordinator.getRestoreRange()

	return &distributedAtomicCounter{
		coordinator: coordinator,
		offset:        offset,
		max:         max,
	}, err
}

func (d* distributedAtomicCounter) next() int {
	d.offset++

	if d.offset > d.max {
		var err error
		d.offset, d.max, err = d.coordinator.getNextRange()

		if err != nil {
			log.Fatal(err.Error())
		}
	}

	d.coordinator.commit(d.offset, d.max)
	return d.offset
}
