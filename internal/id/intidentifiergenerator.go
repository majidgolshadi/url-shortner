package id

import "sync"

type Generator interface {
	GetLastID() uint64
	NewID() uint64
}

type IntegerIdGenerator struct {
	mux sync.Mutex
	id  uint64
}

func (idg *IntegerIdGenerator) NewID() uint64 {
	idg.mux.Lock()
	defer idg.mux.Unlock()
	idg.id++

	return idg.id
}

func (idg *IntegerIdGenerator) GetLastID() uint64 {
	idg.mux.Lock()
	defer idg.mux.Unlock()

	return idg.id
}
