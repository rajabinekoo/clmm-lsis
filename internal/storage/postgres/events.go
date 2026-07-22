package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/rajabinekoo/clmm-lsis/internal/domain"
	"github.com/rajabinekoo/clmm-lsis/internal/storage"
)

func (r *Repository) LoadOrderedEvents(
	ctx context.Context,
	poolAddress domain.Address,
	fromBlockInclusive uint64,
	toBlockInclusive uint64,
) ([]domain.PoolEvent, error) {
	if poolAddress.IsZero() {
		return nil, fmt.Errorf(
			"load ordered events: pool address is required",
		)
	}

	if fromBlockInclusive == 0 {
		return nil, fmt.Errorf(
			"load ordered events: from block must be greater than zero",
		)
	}

	if fromBlockInclusive > toBlockInclusive {
		return nil, fmt.Errorf(
			"load ordered events: invalid range %d-%d",
			fromBlockInclusive,
			toBlockInclusive,
		)
	}

	swapTableExists, err := r.tableExists(
		ctx,
		"pool_swaps",
	)
	if err != nil {
		return nil, err
	}

	if !swapTableExists {
		return nil, fmt.Errorf(
			"%w: exact event replay requires pool_swaps",
			storage.ErrSwapTableUnavailable,
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
		return nil, fmt.Errorf(
			"load ordered events: begin transaction: %w",
			err,
		)
	}

	defer func() {
		_ = tx.Rollback()
	}()

	actions, err := loadLPActions(
		ctx,
		tx,
		poolAddress,
		fromBlockInclusive,
		toBlockInclusive,
	)
	if err != nil {
		return nil, err
	}

	swaps, err := loadSwaps(
		ctx,
		tx,
		poolAddress,
		fromBlockInclusive,
		toBlockInclusive,
	)
	if err != nil {
		return nil, err
	}

	events, err :=
		storage.BuildOrderedEventStream(
			actions,
			swaps,
		)
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf(
			"load ordered events: commit read-only transaction: %w",
			err,
		)
	}

	return events, nil
}

func loadLPActions(
	ctx context.Context,
	tx *sql.Tx,
	poolAddress domain.Address,
	fromBlock uint64,
	toBlock uint64,
) ([]storage.LegacyLPActionRecord, error) {
	rows, err := tx.QueryContext(
		ctx,
		`
			SELECT
				id,
				BTRIM(pool_address),
				action,
				BTRIM(tx_hash),
				block_number,
				log_index,
				timestamp,
				BTRIM(owner),
				BTRIM(sender),
				BTRIM(origin),
				tick_lower,
				tick_upper,
				liquidity_delta::text
			FROM lp_actions
			WHERE pool_address = $1
			  AND block_number BETWEEN $2 AND $3
			ORDER BY
				block_number,
				log_index
		`,
		poolAddress.String(),
		fromBlock,
		toBlock,
	)
	if err != nil {
		return nil, fmt.Errorf(
			"load lp actions for event stream: %w",
			err,
		)
	}
	defer rows.Close()

	records := make(
		[]storage.LegacyLPActionRecord,
		0,
	)

	for rows.Next() {
		var (
			record storage.LegacyLPActionRecord

			action      int64
			blockNumber int64
			logIndex    int64

			owner  sql.NullString
			sender sql.NullString
			origin sql.NullString

			timestamp time.Time

			tickLower int64
			tickUpper int64
		)

		if err := rows.Scan(
			&record.ID,
			&record.PoolAddress,
			&action,
			&record.TransactionHash,
			&blockNumber,
			&logIndex,
			&timestamp,
			&owner,
			&sender,
			&origin,
			&tickLower,
			&tickUpper,
			&record.LiquidityDelta,
		); err != nil {
			return nil, fmt.Errorf(
				"scan lp action event: %w",
				err,
			)
		}

		convertedBlock, err := checkedUint64(
			"lp action block number",
			blockNumber,
		)
		if err != nil {
			return nil, err
		}

		convertedLog, err := checkedUint32(
			"lp action log index",
			logIndex,
		)
		if err != nil {
			return nil, err
		}

		convertedLower, err := checkedInt32(
			"lp action tick lower",
			tickLower,
		)
		if err != nil {
			return nil, err
		}

		convertedUpper, err := checkedInt32(
			"lp action tick upper",
			tickUpper,
		)
		if err != nil {
			return nil, err
		}

		if action < 0 ||
			action > int64(^uint16(0)) {
			return nil, fmt.Errorf(
				"lp action type is outside int16 domain: %d",
				action,
			)
		}

		record.Action =
			storage.LegacyLPActionType(action)

		record.BlockNumber = convertedBlock
		record.LogIndex = convertedLog
		record.Timestamp = timestamp

		record.Owner =
			nullableStringPointer(owner)

		record.Sender =
			nullableStringPointer(sender)

		record.Origin =
			nullableStringPointer(origin)

		record.TickLower = convertedLower
		record.TickUpper = convertedUpper

		records = append(
			records,
			record,
		)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf(
			"iterate lp action events: %w",
			err,
		)
	}

	return records, nil
}

func loadSwaps(
	ctx context.Context,
	tx *sql.Tx,
	poolAddress domain.Address,
	fromBlock uint64,
	toBlock uint64,
) ([]storage.SwapRecord, error) {
	rows, err := tx.QueryContext(
		ctx,
		`
			SELECT
				BTRIM(pool_address),
				block_number,
				BTRIM(block_hash),
				BTRIM(transaction_hash),
				transaction_index,
				log_index,
				timestamp,
				BTRIM(sender),
				BTRIM(recipient),
				amount0_raw::text,
				amount1_raw::text,
				sqrt_price_x96::text,
				active_liquidity::text,
				tick
			FROM pool_swaps
			WHERE pool_address = $1
			  AND block_number BETWEEN $2 AND $3
			ORDER BY
				block_number,
				log_index
		`,
		poolAddress.String(),
		fromBlock,
		toBlock,
	)
	if err != nil {
		return nil, fmt.Errorf(
			"load swaps for event stream: %w",
			err,
		)
	}
	defer rows.Close()

	records := make(
		[]storage.SwapRecord,
		0,
	)

	for rows.Next() {
		var (
			record storage.SwapRecord

			blockNumber      int64
			transactionIndex int64
			logIndex         int64
			tick             int64
		)

		if err := rows.Scan(
			&record.PoolAddress,
			&blockNumber,
			&record.BlockHash,
			&record.TransactionHash,
			&transactionIndex,
			&logIndex,
			&record.Timestamp,
			&record.Sender,
			&record.Recipient,
			&record.Amount0Raw,
			&record.Amount1Raw,
			&record.SqrtPriceX96,
			&record.ActiveLiquidity,
			&tick,
		); err != nil {
			return nil, fmt.Errorf(
				"scan stored swap: %w",
				err,
			)
		}

		convertedBlock, err := checkedUint64(
			"swap block number",
			blockNumber,
		)
		if err != nil {
			return nil, err
		}

		convertedTransactionIndex, err :=
			checkedUint32(
				"swap transaction index",
				transactionIndex,
			)
		if err != nil {
			return nil, err
		}

		convertedLog, err := checkedUint32(
			"swap log index",
			logIndex,
		)
		if err != nil {
			return nil, err
		}

		convertedTick, err := checkedInt32(
			"swap tick",
			tick,
		)
		if err != nil {
			return nil, err
		}

		record.BlockNumber =
			convertedBlock

		record.TransactionIndex =
			convertedTransactionIndex

		record.LogIndex = convertedLog
		record.Tick = convertedTick

		records = append(
			records,
			record,
		)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf(
			"iterate stored swaps: %w",
			err,
		)
	}

	return records, nil
}
