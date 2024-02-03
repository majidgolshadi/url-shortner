package id

import "sync"

type Generator interface {
	GetLastID() int
	NewID() int
}

type IntegerIdGenerator struct {
	mux sync.Mutex
	id  int
}

func (idg *IntegerIdGenerator) NewID() int {
	idg.mux.Lock()
	defer idg.mux.Unlock()
	idg.id++

	return idg.id
}

func (idg *IntegerIdGenerator) GetLastID() int {
	idg.mux.Lock()
	defer idg.mux.Unlock()

	return idg.id
}
