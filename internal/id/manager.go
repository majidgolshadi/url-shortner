package id

import (
	"context"
	"sync"
)

type Manager struct {
	rangeManager RangeManager

	lastID        uint
	reservedRange Range
	mux           sync.Mutex
}

func NewManager(ctx context.Context, rangeMng RangeManager) (*Manager, error) {
	rng, err := rangeMng.getCurrentRange(ctx)
	if err != nil {
		return nil, err
	}

	return &Manager{
		rangeManager: rangeMng,

		lastID:        rng.Start,
		reservedRange: rng,
	}, nil
}

// GetLastID returns the latest used ID
func (m *Manager) GetLastID() uint {
	m.mux.Lock()
	defer m.mux.Unlock()

	return m.lastID
}

// GetNextID retrieves the subsequent integer ID.
// In case the reserved range is entirely consumed, it prompts the range manager to reserve a new range, which is then put into use.
func (m *Manager) GetNextID(ctx context.Context) (id uint, err error) {
	m.mux.Lock()
	defer m.mux.Unlock()

	m.lastID++
	if m.lastID > m.reservedRange.End {
		m.reservedRange, err = m.rangeManager.getNextIDRange(ctx)
		if err != nil {
			return 0, err
		}

		m.lastID = m.reservedRange.Start
	}

	return m.lastID, nil
}
