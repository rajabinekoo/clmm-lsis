package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/rajabinekoo/clmm-lsis/internal/domain"
	"github.com/rajabinekoo/clmm-lsis/internal/storage"
)

func (r *Repository) LoadLatestCheckpoint(
	ctx context.Context,
	pool domain.Pool,
	atOrBeforeBlock uint64,
) (domain.PoolSnapshot, error) {
	if err := pool.Validate(); err != nil {
		return domain.PoolSnapshot{}, fmt.Errorf(
			"load latest checkpoint: %w",
			err,
		)
	}

	if atOrBeforeBlock == 0 {
		return domain.PoolSnapshot{}, fmt.Errorf(
			"load latest checkpoint: target block must be greater than zero",
		)
	}

	tx, err := r.db.BeginTx(
		ctx,
		&sql.TxOptions{
			ReadOnly:  true,
			Isolation: sql.LevelRepeatableRead,
		},
	)
	if err != nil {
		return domain.PoolSnapshot{}, fmt.Errorf(
			"load latest checkpoint: begin transaction: %w",
			err,
		)
	}

	defer func() {
		_ = tx.Rollback()
	}()

	scalar, err := loadLatestScalarSnapshot(
		ctx,
		tx,
		pool.Address,
		atOrBeforeBlock,
	)
	if err != nil {
		return domain.PoolSnapshot{}, err
	}

	missingOwners, err :=
		countMissingOwners(
			ctx,
			tx,
			pool.Address,
			scalar.BlockNumber,
		)
	if err != nil {
		return domain.PoolSnapshot{}, err
	}

	if missingOwners > 0 {
		return domain.PoolSnapshot{}, fmt.Errorf(
			"%w: pool=%s block<=%d rows=%d",
			storage.ErrMissingPositionOwner,
			pool.Address,
			scalar.BlockNumber,
			missingOwners,
		)
	}

	positions, err :=
		loadAggregatedPositions(
			ctx,
			tx,
			pool.Address,
			scalar.BlockNumber,
		)
	if err != nil {
		return domain.PoolSnapshot{}, err
	}

	snapshot, err :=
		storage.BuildCheckpointFromAggregatedPositions(
			pool,
			scalar,
			positions,
		)
	if err != nil {
		return domain.PoolSnapshot{}, err
	}

	if err := tx.Commit(); err != nil {
		return domain.PoolSnapshot{}, fmt.Errorf(
			"load latest checkpoint: commit read-only transaction: %w",
			err,
		)
	}

	return snapshot, nil
}

func loadLatestScalarSnapshot(
	ctx context.Context,
	tx *sql.Tx,
	poolAddress domain.Address,
	atOrBeforeBlock uint64,
) (storage.LegacyPoolSnapshotRecord, error) {
	var (
		record storage.LegacyPoolSnapshotRecord

		blockNumber int64
		tick        sql.NullInt64
	)

	err := tx.QueryRowContext(
		ctx,
		`
			SELECT
				BTRIM(pool_address),
				block_number,
				sqrt_price_x96::text,
				tick,
				active_liquidity::text
			FROM pool_snapshots
			WHERE pool_address = $1
			  AND block_number <= $2
			ORDER BY block_number DESC
			LIMIT 1
		`,
		poolAddress.String(),
		atOrBeforeBlock,
	).Scan(
		&record.PoolAddress,
		&blockNumber,
		&record.SqrtPriceX96,
		&tick,
		&record.ActiveLiquidity,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return storage.LegacyPoolSnapshotRecord{}, fmt.Errorf(
				"%w: pool snapshot for %s at or before block %d",
				storage.ErrRecordNotFound,
				poolAddress,
				atOrBeforeBlock,
			)
		}

		return storage.LegacyPoolSnapshotRecord{}, fmt.Errorf(
			"load latest scalar snapshot for %s: %w",
			poolAddress,
			err,
		)
	}

	convertedBlock, err := checkedUint64(
		"snapshot block number",
		blockNumber,
	)
	if err != nil {
		return storage.LegacyPoolSnapshotRecord{}, err
	}

	record.BlockNumber = convertedBlock

	if tick.Valid {
		convertedTick, err := checkedInt32(
			"snapshot tick",
			tick.Int64,
		)
		if err != nil {
			return storage.LegacyPoolSnapshotRecord{}, err
		}

		record.CurrentTick = &convertedTick
	}

	return record, nil
}

func countMissingOwners(
	ctx context.Context,
	tx *sql.Tx,
	poolAddress domain.Address,
	atOrBeforeBlock uint64,
) (int64, error) {
	var count int64

	err := tx.QueryRowContext(
		ctx,
		`
			SELECT COUNT(*)
			FROM lp_actions
			WHERE pool_address = $1
			  AND block_number <= $2
			  AND (
				owner IS NULL
				OR BTRIM(owner) = ''
			  )
		`,
		poolAddress.String(),
		atOrBeforeBlock,
	).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf(
			"count missing position owners for %s: %w",
			poolAddress,
			err,
		)
	}

	return count, nil
}

func loadAggregatedPositions(
	ctx context.Context,
	tx *sql.Tx,
	poolAddress domain.Address,
	atOrBeforeBlock uint64,
) ([]storage.AggregatedPositionRecord, error) {
	rows, err := tx.QueryContext(
		ctx,
		`
			SELECT
				BTRIM(owner),
				tick_lower,
				tick_upper,
				SUM(liquidity_delta)::text
			FROM lp_actions
			WHERE pool_address = $1
			  AND block_number <= $2
			  AND owner IS NOT NULL
			  AND BTRIM(owner) <> ''
			GROUP BY
				owner,
				tick_lower,
				tick_upper
			HAVING SUM(liquidity_delta) <> 0
			ORDER BY
				BTRIM(owner),
				tick_lower,
				tick_upper
		`,
		poolAddress.String(),
		atOrBeforeBlock,
	)
	if err != nil {
		return nil, fmt.Errorf(
			"load aggregated positions for %s: %w",
			poolAddress,
			err,
		)
	}
	defer rows.Close()

	records := make(
		[]storage.AggregatedPositionRecord,
		0,
	)

	for rows.Next() {
		var (
			record storage.AggregatedPositionRecord

			tickLower int64
			tickUpper int64
		)

		if err := rows.Scan(
			&record.Owner,
			&tickLower,
			&tickUpper,
			&record.Liquidity,
		); err != nil {
			return nil, fmt.Errorf(
				"scan aggregated position for %s: %w",
				poolAddress,
				err,
			)
		}

		convertedLower, err := checkedInt32(
			"aggregated position tick lower",
			tickLower,
		)
		if err != nil {
			return nil, err
		}

		convertedUpper, err := checkedInt32(
			"aggregated position tick upper",
			tickUpper,
		)
		if err != nil {
			return nil, err
		}

		record.TickLower = convertedLower
		record.TickUpper = convertedUpper

		records = append(
			records,
			record,
		)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf(
			"iterate aggregated positions for %s: %w",
			poolAddress,
			err,
		)
	}

	return records, nil
}
