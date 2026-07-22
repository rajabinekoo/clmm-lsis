package postgres

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/rajabinekoo/clmm-lsis/internal/domain"
)

// LegacyPoolStats provides a lightweight overview without loading complete
// event rows into memory.
type LegacyPoolStats struct {
	LPActionCount     int64
	MissingOwnerCount int64

	FirstLPActionBlock *uint64
	LastLPActionBlock  *uint64

	SnapshotCount       int64
	LatestSnapshotBlock *uint64
}

func (r *Repository) LoadLegacyPoolStats(
	ctx context.Context,
	poolAddress domain.Address,
	atOrBeforeBlock uint64,
) (LegacyPoolStats, error) {
	if poolAddress.IsZero() {
		return LegacyPoolStats{}, fmt.Errorf(
			"load legacy pool stats: pool address is required",
		)
	}

	if atOrBeforeBlock == 0 {
		return LegacyPoolStats{}, fmt.Errorf(
			"load legacy pool stats: target block must be greater than zero",
		)
	}

	var (
		stats LegacyPoolStats

		firstAction sql.NullInt64
		lastAction  sql.NullInt64

		latestSnapshot sql.NullInt64
	)

	err := r.db.QueryRowContext(
		ctx,
		`
			SELECT
				COUNT(*),
				COUNT(*) FILTER (
					WHERE owner IS NULL
					   OR BTRIM(owner) = ''
				),
				MIN(block_number),
				MAX(block_number)
			FROM lp_actions
			WHERE pool_address = $1
			  AND block_number <= $2
		`,
		poolAddress.String(),
		atOrBeforeBlock,
	).Scan(
		&stats.LPActionCount,
		&stats.MissingOwnerCount,
		&firstAction,
		&lastAction,
	)
	if err != nil {
		return LegacyPoolStats{}, fmt.Errorf(
			"load lp action stats for %s: %w",
			poolAddress,
			err,
		)
	}

	err = r.db.QueryRowContext(
		ctx,
		`
			SELECT
				COUNT(*),
				MAX(block_number)
			FROM pool_snapshots
			WHERE pool_address = $1
			  AND block_number <= $2
		`,
		poolAddress.String(),
		atOrBeforeBlock,
	).Scan(
		&stats.SnapshotCount,
		&latestSnapshot,
	)
	if err != nil {
		return LegacyPoolStats{}, fmt.Errorf(
			"load snapshot stats for %s: %w",
			poolAddress,
			err,
		)
	}

	if firstAction.Valid {
		value, err := checkedUint64(
			"first lp action block",
			firstAction.Int64,
		)
		if err != nil {
			return LegacyPoolStats{}, err
		}

		stats.FirstLPActionBlock = &value
	}

	if lastAction.Valid {
		value, err := checkedUint64(
			"last lp action block",
			lastAction.Int64,
		)
		if err != nil {
			return LegacyPoolStats{}, err
		}

		stats.LastLPActionBlock = &value
	}

	if latestSnapshot.Valid {
		value, err := checkedUint64(
			"latest snapshot block",
			latestSnapshot.Int64,
		)
		if err != nil {
			return LegacyPoolStats{}, err
		}

		stats.LatestSnapshotBlock = &value
	}

	return stats, nil
}
