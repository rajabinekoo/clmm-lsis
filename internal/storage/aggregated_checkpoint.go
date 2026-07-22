package storage

import (
	"fmt"
	"math/big"

	"github.com/rajabinekoo/clmm-lsis/internal/domain"
)

// AggregatedPositionRecord represents one positive owner-range position
// calculated by PostgreSQL from the legacy lp_actions table.
type AggregatedPositionRecord struct {
	Owner string

	TickLower int32
	TickUpper int32

	Liquidity string
}

// BuildCheckpointFromAggregatedPositions combines a scalar legacy snapshot
// with the positive owner-range positions that existed at the same block.
//
// PostgreSQL may aggregate the legacy event history before calling this
// function. The resulting PoolSnapshot still validates:
//
//   - position liquidity;
//   - initialized tick gross and net liquidity;
//   - active liquidity at the current tick.
func BuildCheckpointFromAggregatedPositions(
	pool domain.Pool,
	scalar LegacyPoolSnapshotRecord,
	records []AggregatedPositionRecord,
) (domain.PoolSnapshot, error) {
	if err := pool.Validate(); err != nil {
		return domain.PoolSnapshot{}, fmt.Errorf(
			"build aggregated checkpoint: invalid pool: %w",
			err,
		)
	}

	scalarPoolAddress, err :=
		parseRequiredStorageAddress(
			"snapshot pool_address",
			scalar.PoolAddress,
		)
	if err != nil {
		return domain.PoolSnapshot{}, err
	}

	if scalarPoolAddress != pool.Address {
		return domain.PoolSnapshot{}, fmt.Errorf(
			"%w: snapshot pool %s does not match configured pool %s",
			ErrCheckpointMismatch,
			scalarPoolAddress,
			pool.Address,
		)
	}

	if scalar.BlockNumber == 0 {
		return domain.PoolSnapshot{}, fmt.Errorf(
			"%w: snapshot block number must be greater than zero",
			ErrInvalidLegacyRecord,
		)
	}

	if scalar.CurrentTick == nil {
		return domain.PoolSnapshot{}, fmt.Errorf(
			"%w: snapshot block %d has no current tick",
			ErrInvalidLegacyRecord,
			scalar.BlockNumber,
		)
	}

	sqrtPriceX96, err := parseStorageBigInt(
		"snapshot sqrt_price_x96",
		scalar.SqrtPriceX96,
	)
	if err != nil {
		return domain.PoolSnapshot{}, err
	}

	if sqrtPriceX96.Sign() <= 0 {
		return domain.PoolSnapshot{}, fmt.Errorf(
			"%w: snapshot sqrt price must be positive",
			ErrInvalidLegacyRecord,
		)
	}

	activeLiquidity, err :=
		parseStorageBigInt(
			"snapshot active_liquidity",
			scalar.ActiveLiquidity,
		)
	if err != nil {
		return domain.PoolSnapshot{}, err
	}

	if activeLiquidity.Sign() < 0 {
		return domain.PoolSnapshot{}, fmt.Errorf(
			"%w: snapshot active liquidity must not be negative",
			ErrInvalidLegacyRecord,
		)
	}

	positions := make(
		[]domain.CorePosition,
		0,
		len(records),
	)

	seen := make(
		map[domain.CorePositionKey]struct{},
		len(records),
	)

	for index, record := range records {
		owner, err :=
			parseRequiredStorageAddress(
				fmt.Sprintf(
					"aggregated position %d owner",
					index,
				),
				record.Owner,
			)
		if err != nil {
			return domain.PoolSnapshot{}, err
		}

		key := domain.CorePositionKey{
			PoolAddress: pool.Address,
			Owner:       owner,
			TickLower:   record.TickLower,
			TickUpper:   record.TickUpper,
		}

		if err := key.Validate(); err != nil {
			return domain.PoolSnapshot{}, fmt.Errorf(
				"%w: aggregated position %d: %v",
				ErrInvalidLegacyRecord,
				index,
				err,
			)
		}

		if record.TickLower%pool.TickSpacing != 0 {
			return domain.PoolSnapshot{}, fmt.Errorf(
				"%w: position %s lower tick is not aligned with tick spacing %d",
				ErrInvalidLegacyRecord,
				key,
				pool.TickSpacing,
			)
		}

		if record.TickUpper%pool.TickSpacing != 0 {
			return domain.PoolSnapshot{}, fmt.Errorf(
				"%w: position %s upper tick is not aligned with tick spacing %d",
				ErrInvalidLegacyRecord,
				key,
				pool.TickSpacing,
			)
		}

		if _, exists := seen[key]; exists {
			return domain.PoolSnapshot{}, fmt.Errorf(
				"%w: duplicate aggregated position %s",
				ErrInvalidLegacyRecord,
				key,
			)
		}

		seen[key] = struct{}{}

		liquidity, err := parseStorageBigInt(
			fmt.Sprintf(
				"aggregated position %d liquidity",
				index,
			),
			record.Liquidity,
		)
		if err != nil {
			return domain.PoolSnapshot{}, err
		}

		if liquidity.Sign() <= 0 {
			return domain.PoolSnapshot{}, fmt.Errorf(
				"%w: aggregated position %s has non-positive liquidity %s",
				ErrCheckpointMismatch,
				key,
				liquidity,
			)
		}

		position, err := domain.NewCorePosition(
			key,
			new(big.Int).Set(liquidity),
		)
		if err != nil {
			return domain.PoolSnapshot{}, fmt.Errorf(
				"build aggregated position %s: %w",
				key,
				err,
			)
		}

		positions = append(
			positions,
			position,
		)
	}

	ticks, err := deriveTicksFromPositions(
		positions,
	)
	if err != nil {
		return domain.PoolSnapshot{}, err
	}

	reference, err :=
		domain.NewBlockEndSnapshotReference(
			scalar.BlockNumber,
		)
	if err != nil {
		return domain.PoolSnapshot{}, err
	}

	snapshot, err := domain.NewPoolSnapshot(
		pool.Address,
		reference,
		sqrtPriceX96,
		*scalar.CurrentTick,
		activeLiquidity,
		ticks,
		positions,
	)
	if err != nil {
		return domain.PoolSnapshot{}, fmt.Errorf(
			"%w: build aggregated checkpoint at block %d: %v",
			ErrCheckpointMismatch,
			scalar.BlockNumber,
			err,
		)
	}

	return snapshot, nil
}
