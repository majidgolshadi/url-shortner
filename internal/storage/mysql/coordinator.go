package mysql

import (
	"context"
	"github.com/jmoiron/sqlx"
	"github.com/majidgolshadi/url-shortner/internal/domain"
	"github.com/majidgolshadi/url-shortner/internal/storage"
	"strconv"
)

const lastReservedIDKey = "last_reserved_id"

type coordinator struct {
	db *sqlx.DB
}

type (
	nodesCoordinationKeyRow struct {
		Key     string `db:"key"`
		Value   string `db:"value"`
		Version int    `db:"version"`
	}
	nodeRangeJournalRow struct {
		NodeID string `db:"node_id"`
		Start  uint   `db:"start"`
		End    uint   `db:"end"`
	}
)

func NewCoordinator(db *sqlx.DB) storage.Coordinator {
	return &coordinator{
		db: db,
	}
}

func (c *coordinator) GetNodeReservedRange(ctx context.Context, nodeID string) (*domain.Range, error) {
	var row nodeRangeJournalRow
	err := c.db.GetContext(ctx, &row, "SELECT start, end FROM node_range_journal WHERE node_id=?", nodeID)
	if err != nil {
		return nil, err
	}

	return &domain.Range{
		Start: row.Start,
		End:   row.End,
	}, nil
}

func (c *coordinator) TakeNextFreeRange(ctx context.Context, nodeID string, rangeSize int) (*domain.Range, error) {
	var ncRow nodesCoordinationKeyRow
	err := c.db.GetContext(ctx, &ncRow, "SELECT value, version FROM nodes_coodination_keys WHERE key=?", lastReservedIDKey)
	if err != nil {
		return nil, err
	}

	v, err := strconv.Atoi(ncRow.Value)
	if err != nil {
		return nil, err
	}

	tx, err := c.db.BeginTxx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	latestReservedIDValue := uint(v + rangeSize)
	newVersion := ncRow.Version + 1
	_, err = tx.ExecContext(ctx, "INSERT INTO nodes_coodination_keys(key, value, version) VALUES (?,?,?)",
		lastReservedIDKey, latestReservedIDValue, newVersion)
	if err != nil {
		return nil, err
	}

	startRangeID := latestReservedIDValue + 1
	_, err = tx.ExecContext(ctx, "INSERT INTO node_range_journal(node_id, start, end) VALUES (?,?,?)",
		nodeID, startRangeID, latestReservedIDValue)
	if err != nil {
		return nil, err
	}

	err = tx.Commit()
	// TODO: throw retryable error
	if err != nil {
		return nil, err
	}

	return &domain.Range{
		Start: startRangeID,
		End:   latestReservedIDValue,
	}, nil
}

func (c *coordinator) UpdateRemainedRange(nodeID string, remainedRange domain.Range) error {
	_, err := c.db.Exec("INSERT INTO node_range_journal(node_id, start, end) VALUES (?,?,?)",
		nodeID, remainedRange.Start, remainedRange.End)
	return err
}
