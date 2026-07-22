package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/rajabinekoo/clmm-lsis/internal/domain"
	"github.com/rajabinekoo/clmm-lsis/internal/storage"
)

const swapIndexRangeColumns = `
	BTRIM(pool_address),
	from_block,
	to_block,
	next_block,
	status,
	last_processed_block_hash,
	last_error,
	created_at,
	updated_at
`

type rowScanner interface {
	Scan(dest ...any) error
}

// EnsureSwapIndexRange creates a pending range when it does not already exist.
//
// Calling this method repeatedly with the same key is idempotent.
func (r *Repository) EnsureSwapIndexRange(
	ctx context.Context,
	key storage.SwapIndexRangeKey,
) (storage.SwapIndexRange, error) {
	if err := key.Validate(); err != nil {
		return storage.SwapIndexRange{}, fmt.Errorf(
			"ensure swap index range: %w",
			err,
		)
	}

	if err := r.RequireSwapSchema(ctx); err != nil {
		return storage.SwapIndexRange{}, err
	}

	indexRange, err :=
		storage.NewSwapIndexRange(
			key.PoolAddress,
			key.FromBlock,
			key.ToBlock,
		)
	if err != nil {
		return storage.SwapIndexRange{}, err
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return storage.SwapIndexRange{}, fmt.Errorf(
			"ensure swap index range: begin transaction: %w",
			err,
		)
	}

	defer func() {
		_ = tx.Rollback()
	}()

	_, err = tx.ExecContext(
		ctx,
		`
			INSERT INTO public.pool_swap_index_ranges
			(
				pool_address,
				from_block,
				to_block,
				next_block,
				status
			)
			VALUES ($1, $2, $3, $4, $5)
			ON CONFLICT
			(
				pool_address,
				from_block,
				to_block
			)
			DO NOTHING
		`,
		indexRange.PoolAddress.String(),
		int64(indexRange.FromBlock),
		int64(indexRange.ToBlock),
		int64(indexRange.NextBlock),
		string(indexRange.Status),
	)
	if err != nil {
		return storage.SwapIndexRange{}, fmt.Errorf(
			"ensure swap index range %s: insert: %w",
			key,
			err,
		)
	}

	stored, err := loadSwapIndexRangeTx(
		ctx,
		tx,
		key,
		true,
	)
	if err != nil {
		return storage.SwapIndexRange{}, err
	}

	if err := tx.Commit(); err != nil {
		return storage.SwapIndexRange{}, fmt.Errorf(
			"ensure swap index range %s: commit: %w",
			key,
			err,
		)
	}

	return stored, nil
}

func (r *Repository) LoadSwapIndexRange(
	ctx context.Context,
	key storage.SwapIndexRangeKey,
) (storage.SwapIndexRange, error) {
	if err := key.Validate(); err != nil {
		return storage.SwapIndexRange{}, fmt.Errorf(
			"load swap index range: %w",
			err,
		)
	}

	row := r.db.QueryRowContext(
		ctx,
		`
			SELECT `+swapIndexRangeColumns+`
			FROM public.pool_swap_index_ranges
			WHERE pool_address = $1
			  AND from_block = $2
			  AND to_block = $3
		`,
		key.PoolAddress.String(),
		int64(key.FromBlock),
		int64(key.ToBlock),
	)

	indexRange, err :=
		scanSwapIndexRange(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return storage.SwapIndexRange{}, fmt.Errorf(
				"%w: %s",
				storage.ErrSwapIndexRangeNotFound,
				key,
			)
		}

		return storage.SwapIndexRange{}, fmt.Errorf(
			"load swap index range %s: %w",
			key,
			err,
		)
	}

	return indexRange, nil
}

func loadSwapIndexRangeTx(
	ctx context.Context,
	tx *sql.Tx,
	key storage.SwapIndexRangeKey,
	forUpdate bool,
) (storage.SwapIndexRange, error) {
	query := `
		SELECT ` + swapIndexRangeColumns + `
		FROM public.pool_swap_index_ranges
		WHERE pool_address = $1
		  AND from_block = $2
		  AND to_block = $3
	`

	if forUpdate {
		query += ` FOR UPDATE`
	}

	row := tx.QueryRowContext(
		ctx,
		query,
		key.PoolAddress.String(),
		int64(key.FromBlock),
		int64(key.ToBlock),
	)

	indexRange, err :=
		scanSwapIndexRange(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return storage.SwapIndexRange{}, fmt.Errorf(
				"%w: %s",
				storage.ErrSwapIndexRangeNotFound,
				key,
			)
		}

		return storage.SwapIndexRange{}, fmt.Errorf(
			"load swap index range %s: %w",
			key,
			err,
		)
	}

	return indexRange, nil
}

func scanSwapIndexRange(
	scanner rowScanner,
) (storage.SwapIndexRange, error) {
	var (
		poolAddressText string

		fromBlock int64
		toBlock   int64
		nextBlock int64

		status string

		lastProcessedBlockHash sql.NullString
		lastError              sql.NullString

		createdAt time.Time
		updatedAt time.Time
	)

	if err := scanner.Scan(
		&poolAddressText,
		&fromBlock,
		&toBlock,
		&nextBlock,
		&status,
		&lastProcessedBlockHash,
		&lastError,
		&createdAt,
		&updatedAt,
	); err != nil {
		return storage.SwapIndexRange{}, err
	}

	poolAddress, err :=
		domain.ParseAddress(poolAddressText)
	if err != nil {
		return storage.SwapIndexRange{}, fmt.Errorf(
			"parse stored swap range pool address: %w",
			err,
		)
	}

	convertedFrom, err :=
		checkedUint64(
			"swap range from block",
			fromBlock,
		)
	if err != nil {
		return storage.SwapIndexRange{}, err
	}

	convertedTo, err :=
		checkedUint64(
			"swap range to block",
			toBlock,
		)
	if err != nil {
		return storage.SwapIndexRange{}, err
	}

	convertedNext, err :=
		checkedUint64(
			"swap range next block",
			nextBlock,
		)
	if err != nil {
		return storage.SwapIndexRange{}, err
	}

	var parsedLastHash domain.Hash

	if lastProcessedBlockHash.Valid {
		parsedLastHash, err =
			domain.ParseHash(
				lastProcessedBlockHash.String,
			)
		if err != nil {
			return storage.SwapIndexRange{}, fmt.Errorf(
				"parse stored last processed block hash: %w",
				err,
			)
		}
	}

	indexRange := storage.SwapIndexRange{
		PoolAddress: poolAddress,

		FromBlock: convertedFrom,
		ToBlock:   convertedTo,
		NextBlock: convertedNext,

		Status: storage.SwapIndexStatus(status),

		LastProcessedBlockHash: parsedLastHash,

		CreatedAt: createdAt,

		UpdatedAt: updatedAt,
	}

	if lastError.Valid {
		indexRange.LastError =
			lastError.String
	}

	if err := indexRange.Validate(); err != nil {
		return storage.SwapIndexRange{}, fmt.Errorf(
			"stored swap index range is invalid: %w",
			err,
		)
	}

	return indexRange, nil
}
