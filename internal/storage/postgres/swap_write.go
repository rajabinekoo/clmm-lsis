package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5/pgconn"

	"github.com/rajabinekoo/clmm-lsis/internal/domain"
	"github.com/rajabinekoo/clmm-lsis/internal/storage"
)

// CommitSwapBatch stores all Swap events and advances range progress inside a
// single transaction.
//
// Either all swaps and the new NextBlock are committed, or none of them are.
func (r *Repository) CommitSwapBatch(
	ctx context.Context,
	commit storage.SwapBatchCommit,
) (storage.SwapIndexRange, error) {
	if err := commit.Validate(); err != nil {
		return storage.SwapIndexRange{}, err
	}

	if err := r.RequireSwapSchema(ctx); err != nil {
		return storage.SwapIndexRange{}, err
	}

	tx, err := r.db.BeginTx(
		ctx,
		&sql.TxOptions{
			Isolation: sql.LevelReadCommitted,
		},
	)
	if err != nil {
		return storage.SwapIndexRange{}, fmt.Errorf(
			"commit swap batch: begin transaction: %w",
			err,
		)
	}

	defer func() {
		_ = tx.Rollback()
	}()

	currentRange, err :=
		loadSwapIndexRangeTx(
			ctx,
			tx,
			commit.RangeKey,
			true,
		)
	if err != nil {
		return storage.SwapIndexRange{}, err
	}

	nextRange, err :=
		advanceSwapIndexRange(
			currentRange,
			commit,
		)
	if err != nil {
		return storage.SwapIndexRange{}, err
	}

	for index, swap := range commit.CopySwaps() {
		if err := insertOrVerifySwap(
			ctx,
			tx,
			swap,
		); err != nil {
			return storage.SwapIndexRange{}, fmt.Errorf(
				"commit swap batch record %d: %w",
				index,
				err,
			)
		}
	}

	updatedRange, err :=
		updateSwapIndexRangeProgress(
			ctx,
			tx,
			nextRange,
			commit.ExpectedNextBlock,
		)
	if err != nil {
		return storage.SwapIndexRange{}, err
	}

	if err := tx.Commit(); err != nil {
		return storage.SwapIndexRange{}, fmt.Errorf(
			"commit swap batch transaction: %w",
			err,
		)
	}

	return updatedRange, nil
}

// MarkSwapIndexRangeFailed records a failure without changing NextBlock.
//
// Retrying therefore resumes from the first block that was never successfully
// committed.
func (r *Repository) MarkSwapIndexRangeFailed(
	ctx context.Context,
	key storage.SwapIndexRangeKey,
	expectedNextBlock uint64,
	reason string,
) (storage.SwapIndexRange, error) {
	if err := key.Validate(); err != nil {
		return storage.SwapIndexRange{}, err
	}

	normalizedReason :=
		strings.TrimSpace(reason)

	if normalizedReason == "" {
		return storage.SwapIndexRange{}, fmt.Errorf(
			"mark swap index range failed: reason is required",
		)
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return storage.SwapIndexRange{}, fmt.Errorf(
			"mark swap index range failed: begin transaction: %w",
			err,
		)
	}

	defer func() {
		_ = tx.Rollback()
	}()

	current, err :=
		loadSwapIndexRangeTx(
			ctx,
			tx,
			key,
			true,
		)
	if err != nil {
		return storage.SwapIndexRange{}, err
	}

	next, err := failSwapIndexRange(
		current,
		expectedNextBlock,
		normalizedReason,
	)
	if err != nil {
		return storage.SwapIndexRange{}, err
	}

	row := tx.QueryRowContext(
		ctx,
		`
			UPDATE public.pool_swap_index_ranges
			SET
				status = $4,
				last_error = $5,
				updated_at = NOW()
			WHERE pool_address = $1
			  AND from_block = $2
			  AND to_block = $3
			RETURNING `+swapIndexRangeColumns,
		next.PoolAddress.String(),
		int64(next.FromBlock),
		int64(next.ToBlock),
		string(next.Status),
		next.LastError,
	)

	updated, err :=
		scanSwapIndexRange(row)
	if err != nil {
		return storage.SwapIndexRange{}, fmt.Errorf(
			"mark swap index range %s failed: %w",
			key,
			err,
		)
	}

	if err := tx.Commit(); err != nil {
		return storage.SwapIndexRange{}, fmt.Errorf(
			"mark swap index range failed: commit: %w",
			err,
		)
	}

	return updated, nil
}

func advanceSwapIndexRange(
	current storage.SwapIndexRange,
	commit storage.SwapBatchCommit,
) (storage.SwapIndexRange, error) {
	if err := current.Validate(); err != nil {
		return storage.SwapIndexRange{}, fmt.Errorf(
			"current swap range: %w",
			err,
		)
	}

	if err := commit.Validate(); err != nil {
		return storage.SwapIndexRange{}, err
	}

	if current.Key() != commit.RangeKey {
		return storage.SwapIndexRange{}, fmt.Errorf(
			"%w: stored range %s does not match batch range %s",
			storage.ErrSwapIndexProgressConflict,
			current.Key(),
			commit.RangeKey,
		)
	}

	if current.Complete() {
		return storage.SwapIndexRange{}, fmt.Errorf(
			"%w: range %s is already complete",
			storage.ErrSwapIndexProgressConflict,
			current.Key(),
		)
	}

	if current.NextBlock !=
		commit.ExpectedNextBlock {
		return storage.SwapIndexRange{}, fmt.Errorf(
			"%w: range %s expected next block stored=%d caller=%d",
			storage.ErrSwapIndexProgressConflict,
			current.Key(),
			current.NextBlock,
			commit.ExpectedNextBlock,
		)
	}

	next := current

	next.NextBlock =
		commit.ProcessedThroughBlock + 1

	next.LastProcessedBlockHash =
		commit.ProcessedThroughBlockHash

	next.LastError = ""

	if next.NextBlock ==
		next.ToBlock+1 {
		next.Status =
			storage.SwapIndexComplete
	} else {
		next.Status =
			storage.SwapIndexRunning
	}

	if err := next.Validate(); err != nil {
		return storage.SwapIndexRange{}, fmt.Errorf(
			"advanced swap range is invalid: %w",
			err,
		)
	}

	return next, nil
}

func failSwapIndexRange(
	current storage.SwapIndexRange,
	expectedNextBlock uint64,
	reason string,
) (storage.SwapIndexRange, error) {
	if err := current.Validate(); err != nil {
		return storage.SwapIndexRange{}, err
	}

	if current.Complete() {
		return storage.SwapIndexRange{}, fmt.Errorf(
			"%w: completed range %s cannot be marked failed",
			storage.ErrSwapIndexProgressConflict,
			current.Key(),
		)
	}

	if current.NextBlock != expectedNextBlock {
		return storage.SwapIndexRange{}, fmt.Errorf(
			"%w: range %s expected next block stored=%d caller=%d",
			storage.ErrSwapIndexProgressConflict,
			current.Key(),
			current.NextBlock,
			expectedNextBlock,
		)
	}

	next := current
	next.Status = storage.SwapIndexFailed
	next.LastError = reason

	if err := next.Validate(); err != nil {
		return storage.SwapIndexRange{}, err
	}

	return next, nil
}

func updateSwapIndexRangeProgress(
	ctx context.Context,
	tx *sql.Tx,
	next storage.SwapIndexRange,
	expectedNextBlock uint64,
) (storage.SwapIndexRange, error) {
	row := tx.QueryRowContext(
		ctx,
		`
			UPDATE public.pool_swap_index_ranges
			SET
				next_block = $4,
				status = $5,
				last_processed_block_hash = $6,
				last_error = NULL,
				updated_at = NOW()
			WHERE pool_address = $1
			  AND from_block = $2
			  AND to_block = $3
			  AND next_block = $7
			  AND status <> 'complete'
			RETURNING `+swapIndexRangeColumns,
		next.PoolAddress.String(),
		int64(next.FromBlock),
		int64(next.ToBlock),
		int64(next.NextBlock),
		string(next.Status),
		next.LastProcessedBlockHash.String(),
		int64(expectedNextBlock),
	)

	updated, err :=
		scanSwapIndexRange(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return storage.SwapIndexRange{}, fmt.Errorf(
				"%w: range %s changed before progress update",
				storage.ErrSwapIndexProgressConflict,
				next.Key(),
			)
		}

		return storage.SwapIndexRange{}, fmt.Errorf(
			"update swap index range %s: %w",
			next.Key(),
			err,
		)
	}

	return updated, nil
}

func insertOrVerifySwap(
	ctx context.Context,
	tx *sql.Tx,
	record storage.SwapRecord,
) error {
	if err := record.Validate(); err != nil {
		return err
	}

	event, err := record.DomainEvent()
	if err != nil {
		return err
	}

	payload, ok :=
		event.Payload().(domain.SwapEvent)

	if !ok {
		return fmt.Errorf(
			"swap record produced payload %T",
			event.Payload(),
		)
	}

	result, err := tx.ExecContext(
		ctx,
		`
			INSERT INTO public.pool_swaps
			(
				pool_address,
				block_number,
				block_hash,
				transaction_hash,
				transaction_index,
				log_index,
				timestamp,
				sender,
				recipient,
				amount0_raw,
				amount1_raw,
				sqrt_price_x96,
				active_liquidity,
				tick
			)
			VALUES
			(
				$1, $2, $3, $4, $5, $6, $7,
				$8, $9, $10, $11, $12, $13, $14
			)
			ON CONFLICT
			(
				pool_address,
				block_number,
				log_index
			)
			DO NOTHING
		`,
		event.PoolAddress().String(),
		int64(event.Cursor().BlockNumber),
		event.BlockHash().String(),
		event.TransactionHash().String(),
		int64(event.Cursor().TransactionIndex),
		int64(event.Cursor().LogIndex),
		record.Timestamp.UTC(),
		payload.Sender().String(),
		payload.Recipient().String(),
		payload.Amount0().String(),
		payload.Amount1().String(),
		payload.SqrtPriceX96().String(),
		payload.ActiveLiquidity().String(),
		int64(payload.Tick()),
	)
	if err != nil {
		var postgresError *pgconn.PgError

		if errors.As(err, &postgresError) &&
			postgresError.Code == "23505" {
			return fmt.Errorf(
				"%w: unique constraint=%s detail=%s",
				storage.ErrStoredSwapConflict,
				postgresError.ConstraintName,
				postgresError.Detail,
			)
		}

		return fmt.Errorf(
			"insert swap at %s: %w",
			event.Cursor(),
			err,
		)
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf(
			"read inserted swap row count: %w",
			err,
		)
	}

	if affected == 1 {
		return nil
	}

	if affected != 0 {
		return fmt.Errorf(
			"insert swap affected unexpected row count %d",
			affected,
		)
	}

	stored, err :=
		loadStoredSwapTx(
			ctx,
			tx,
			event.PoolAddress(),
			event.Cursor(),
		)
	if err != nil {
		return err
	}

	equivalent, err :=
		stored.Equivalent(record)
	if err != nil {
		return fmt.Errorf(
			"compare stored swap at %s: %w",
			event.Cursor(),
			err,
		)
	}

	if !equivalent {
		return fmt.Errorf(
			"%w: pool=%s cursor=%s transaction=%s",
			storage.ErrStoredSwapConflict,
			event.PoolAddress(),
			event.Cursor(),
			event.TransactionHash(),
		)
	}

	return nil
}

func loadStoredSwapTx(
	ctx context.Context,
	tx *sql.Tx,
	poolAddress domain.Address,
	cursor domain.EventCursor,
) (storage.SwapRecord, error) {
	row := tx.QueryRowContext(
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
			FROM public.pool_swaps
			WHERE pool_address = $1
			  AND block_number = $2
			  AND log_index = $3
		`,
		poolAddress.String(),
		int64(cursor.BlockNumber),
		int64(cursor.LogIndex),
	)

	return scanStoredSwap(row)
}

func scanStoredSwap(
	scanner rowScanner,
) (storage.SwapRecord, error) {
	var (
		record storage.SwapRecord

		blockNumber      int64
		transactionIndex int64
		logIndex         int64
		tick             int64
	)

	if err := scanner.Scan(
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
		if errors.Is(err, sql.ErrNoRows) {
			return storage.SwapRecord{}, fmt.Errorf(
				"%w: stored swap was not found after conflict",
				storage.ErrStoredSwapConflict,
			)
		}

		return storage.SwapRecord{}, fmt.Errorf(
			"scan stored swap: %w",
			err,
		)
	}

	convertedBlock, err := checkedUint64(
		"stored swap block number",
		blockNumber,
	)
	if err != nil {
		return storage.SwapRecord{}, err
	}

	convertedTransactionIndex, err :=
		checkedUint32(
			"stored swap transaction index",
			transactionIndex,
		)
	if err != nil {
		return storage.SwapRecord{}, err
	}

	convertedLogIndex, err :=
		checkedUint32(
			"stored swap log index",
			logIndex,
		)
	if err != nil {
		return storage.SwapRecord{}, err
	}

	convertedTick, err := checkedInt32(
		"stored swap tick",
		tick,
	)
	if err != nil {
		return storage.SwapRecord{}, err
	}

	record.BlockNumber =
		convertedBlock

	record.TransactionIndex =
		convertedTransactionIndex

	record.LogIndex =
		convertedLogIndex

	record.Tick =
		convertedTick

	return record, nil
}
