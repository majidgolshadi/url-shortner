package mysql

import (
	"context"
	"database/sql"
	"fmt"
	"github.com/jmoiron/sqlx"
	"github.com/majidgolshadi/url-shortner/internal/domain"
	intErr "github.com/majidgolshadi/url-shortner/internal/infrastructure/errors"
	"github.com/majidgolshadi/url-shortner/internal/storage"
	"github.com/pkg/errors"
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
	var row nodesCoordinationKeyRow
	dataVersion := 1
	latestReservedIDValue := uint(rangeSize)

	err := c.db.GetContext(ctx, &row, "SELECT value, version FROM nodes_coodination_keys WHERE key=?", lastReservedIDKey)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return nil, err
	}

	// lastReservedIDKey row has been initiated
	if err == nil {
		lastReservedID, strConvErr := strconv.Atoi(row.Value)
		if strConvErr != nil {
			return nil, strConvErr
		}

		latestReservedIDValue = uint(lastReservedID + rangeSize)
		dataVersion = row.Version + 1
	}

	tx, err := c.db.BeginTxx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	query := `
		INSERT INTO nodes_coodination_keys(key, value, version) VALUES (?,?,?)
		ON DUPLICATE KEY UPDATE value = VALUES(value), version = VALUES(version);`
	if _, err = tx.ExecContext(ctx, query, lastReservedIDKey, latestReservedIDValue, dataVersion); err != nil {
		return nil, err
	}

	startRangeID := latestReservedIDValue + 1
	_, err = tx.ExecContext(ctx, "INSERT INTO node_range_journal(node_id, start, end) VALUES (?,?,?);",
		nodeID, startRangeID, latestReservedIDValue)
	if err != nil {
		return nil, err
	}

	err = tx.Commit()
	if err != nil {
		return nil, fmt.Errorf("%w error %s", intErr.CoordinatorTakeNextFreeRangeErr, err.Error())
	}

	return &domain.Range{
		Start: startRangeID,
		End:   latestReservedIDValue,
	}, nil
}

func (c *coordinator) getNodesCoordinationKeyData(ctx context.Context, key string) (*nodesCoordinationKeyRow, error) {
	var row nodesCoordinationKeyRow

	err := c.db.GetContext(ctx, &row, "SELECT value, version FROM nodes_coodination_keys WHERE key=?", key)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return nil, err
	}

	if errors.Is(err, sql.ErrNoRows) {
		return &nodesCoordinationKeyRow{
			Value:   "",
			Version: 0,
		}, nil
	}

	return &row, nil
}

func (c *coordinator) UpdateRemainedRange(nodeID string, remainedRange domain.Range) error {
	_, err := c.db.Exec("INSERT INTO node_range_journal(node_id, start, end) VALUES (?,?,?)",
		nodeID, remainedRange.Start, remainedRange.End)
	return err
}
