package id

import (
	"context"
	"github.com/majidgolshadi/url-shortner/internal/domain"
	intErr "github.com/majidgolshadi/url-shortner/internal/infrastructure/errors"
	"github.com/pkg/errors"
	"sync"
)

type Manager struct {
	rangeManager RangeManager

	lastID        uint
	reservedRange domain.Range
	mux           sync.Mutex
}

func NewManager(ctx context.Context, rangeMng RangeManager) (*Manager, error) {
	rng, err := rangeMng.getCurrentRange(ctx)

	// fresh run
	if errors.Is(err, intErr.RangeManagerNoReservedRangeErr) {
		rng, err = rangeMng.getNextIDRange(ctx)
	}

	if err != nil {
		return nil, err
	}

	return &Manager{
		rangeManager: rangeMng,
		lastID:       rng.Start,
		reservedRange: domain.Range{
			Start: rng.Start,
			End:   rng.End,
		},
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
func (m *Manager) GetNextID(ctx context.Context) (uint, error) {
	m.mux.Lock()
	defer m.mux.Unlock()

	m.lastID++
	if m.lastID > m.reservedRange.End {
		takenRange, err := m.rangeManager.getNextIDRange(ctx)
		if err != nil {
			return 0, err
		}

		m.reservedRange = takenRange
		m.lastID = m.reservedRange.Start
	}

	return m.lastID, nil
}
