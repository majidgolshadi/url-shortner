package id

import "sync"

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
