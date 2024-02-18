package mysql

import (
	"context"
	"database/sql"
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
		KeyID   string `db:"key_id"`
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

// GetNodeReservedRange get requested node latest reserved range
func (c *coordinator) GetNodeReservedRange(ctx context.Context, nodeID string) (*domain.Range, error) {
	var row nodeRangeJournalRow
	err := c.db.GetContext(ctx, &row, "SELECT start, end FROM node_range_journal WHERE node_id=?;", nodeID)
	if err != nil {
		return nil, err
	}

	return &domain.Range{
		Start: row.Start,
		End:   row.End,
	}, nil
}

// GetLatestReservedRange returns the  latest reserved range with its version
func (c *coordinator) GetLatestReservedRange(ctx context.Context) (lastReservedNumber uint, version int, err error) {
	var row nodesCoordinationKeyRow

	err = c.db.GetContext(ctx, &row, "SELECT value, version FROM nodes_coordination_keys WHERE key_id = ?;", lastReservedIDKey)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, 0, intErr.CoordinatorNoReservedRangeErr
	}

	if err != nil {
		return 0, 0, err
	}

	end, strConvErr := strconv.Atoi(row.Value)
	if strConvErr != nil {
		return 0, 0, strConvErr
	}

	return uint(end), row.Version, nil
}

// TakeFreeRange take requested range otherwise throw an error
// errors:
// CoordinatorRangeFragmentationErr in case there is a gap between requested range start point with the latest reserved end point
// CoordinatorDataInvalidVersionErr invalid version number
// mysql errors
func (c *coordinator) TakeFreeRange(ctx context.Context, nodeID string, requestedRange domain.Range, version int) error {
	var row nodesCoordinationKeyRow

	err := c.db.GetContext(ctx, &row, "SELECT value, version FROM nodes_coordination_keys WHERE key_id = ?;", lastReservedIDKey)
	if errors.Is(err, sql.ErrNoRows) {
		// no need to check the initial version
		return c.setRange(ctx, nodeID, requestedRange, version)
	}

	if err != nil {
		return err
	}

	latestReservedRange, strConvErr := strconv.Atoi(row.Value)
	if strConvErr != nil {
		return strConvErr
	}

	// ensure no gap exists between reserved ranges
	if requestedRange.Start != uint(latestReservedRange+1) {
		return intErr.CoordinatorRangeFragmentationErr
	}

	return c.setRange(ctx, nodeID, requestedRange, version)
}

// setRange updates the current coordination with the latest reserved range.
// It also journals the request for traceability and potential future debugging.
func (c *coordinator) setRange(ctx context.Context, nodeID string, requestedRange domain.Range, version int) error {
	tx, err := c.db.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}
	// nolint:errcheck
	defer tx.Rollback()

	query := `INSERT INTO nodes_coordination_keys(key_id, value, version) VALUES (?,?,?)
		ON DUPLICATE KEY UPDATE value = VALUES(value), version = VALUES(version);`
	if _, err = tx.ExecContext(ctx, query, lastReservedIDKey, requestedRange.End, version); err != nil {
		return translateMysqlError(err)
	}

	_, err = tx.ExecContext(ctx, "INSERT INTO node_range_journal(node_id, start, end) VALUES (?,?,?);",
		nodeID, requestedRange.Start, requestedRange.End)
	if err != nil {
		return err
	}

	return tx.Commit()
}
