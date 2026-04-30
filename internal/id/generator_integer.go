package id

import "sync"

// IntegerIdGenerator is a simple monotonic counter.
// Mutex is required because multiple goroutines may call NewID concurrently.
type IntegerIdGenerator struct {
	mux sync.Mutex
	id  uint
}

func (idg *IntegerIdGenerator) NewID() uint {
	idg.mux.Lock()
	defer idg.mux.Unlock()
	idg.id++

	return idg.id
}

func (idg *IntegerIdGenerator) GetLastID() uint {
	idg.mux.Lock()
	defer idg.mux.Unlock()

	return idg.id
}
