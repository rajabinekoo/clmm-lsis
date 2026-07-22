package storage

import (
	"fmt"
	"math/big"
	"sort"
	"strings"

	"github.com/rajabinekoo/clmm-lsis/internal/domain"
)

// BuildLegacyCheckpoint combines one scalar pool_snapshots row with all
// liquidity actions up to that block.
//
// No network re-indexing is required. Existing lp_actions data is aggregated
// into complete owner-range positions and initialized tick state.
func BuildLegacyCheckpoint(
	pool domain.Pool,
	scalar LegacyPoolSnapshotRecord,
	actions []LegacyLPActionRecord,
) (domain.PoolSnapshot, error) {
	if err := pool.Validate(); err != nil {
		return domain.PoolSnapshot{}, fmt.Errorf(
			"build legacy checkpoint: invalid pool: %w",
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

	sqrtPriceX96, err :=
		parseStorageBigInt(
			"snapshot sqrt_price_x96",
			scalar.SqrtPriceX96,
		)
	if err != nil {
		return domain.PoolSnapshot{}, err
	}

	activeLiquidity, err :=
		parseStorageBigInt(
			"snapshot active_liquidity",
			scalar.ActiveLiquidity,
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

	if activeLiquidity.Sign() < 0 {
		return domain.PoolSnapshot{}, fmt.Errorf(
			"%w: snapshot active liquidity must not be negative",
			ErrInvalidLegacyRecord,
		)
	}

	sortedActions := append(
		[]LegacyLPActionRecord(nil),
		actions...,
	)

	sort.SliceStable(
		sortedActions,
		func(i, j int) bool {
			if sortedActions[i].BlockNumber !=
				sortedActions[j].BlockNumber {
				return sortedActions[i].BlockNumber <
					sortedActions[j].BlockNumber
			}

			return sortedActions[i].LogIndex <
				sortedActions[j].LogIndex
		},
	)

	positionLiquidity := make(
		map[domain.CorePositionKey]*big.Int,
	)

	var previousCursor *domain.EventCursor

	for _, action := range sortedActions {
		if action.BlockNumber >
			scalar.BlockNumber {
			return domain.PoolSnapshot{}, fmt.Errorf(
				"%w: action %s at block %d occurs after snapshot block %d",
				ErrCheckpointMismatch,
				action.ID,
				action.BlockNumber,
				scalar.BlockNumber,
			)
		}

		actionPoolAddress, err :=
			parseRequiredStorageAddress(
				"lp action pool_address",
				action.PoolAddress,
			)
		if err != nil {
			return domain.PoolSnapshot{}, err
		}

		if actionPoolAddress != pool.Address {
			return domain.PoolSnapshot{}, fmt.Errorf(
				"%w: action %s belongs to pool %s",
				ErrCheckpointMismatch,
				action.ID,
				actionPoolAddress,
			)
		}

		cursor := domain.EventCursor{
			BlockNumber: action.BlockNumber,
			LogIndex:    action.LogIndex,
		}

		if previousCursor != nil &&
			cursor.Compare(*previousCursor) <= 0 {
			return domain.PoolSnapshot{}, fmt.Errorf(
				"%w: liquidity actions are not uniquely ordered at %s",
				ErrDuplicateEventCursor,
				cursor,
			)
		}

		copiedCursor := cursor
		previousCursor = &copiedCursor

		owner, exists, err :=
			parseOptionalStorageAddress(
				"lp action owner",
				action.Owner,
			)
		if err != nil {
			return domain.PoolSnapshot{}, err
		}

		if !exists {
			return domain.PoolSnapshot{}, fmt.Errorf(
				"%w: action id=%s",
				ErrMissingPositionOwner,
				action.ID,
			)
		}

		key := domain.CorePositionKey{
			PoolAddress: pool.Address,
			Owner:       owner,
			TickLower:   action.TickLower,
			TickUpper:   action.TickUpper,
		}

		if err := key.Validate(); err != nil {
			return domain.PoolSnapshot{}, fmt.Errorf(
				"%w: action %s position: %v",
				ErrInvalidLegacyRecord,
				action.ID,
				err,
			)
		}

		if action.TickLower%pool.TickSpacing != 0 ||
			action.TickUpper%pool.TickSpacing != 0 {
			return domain.PoolSnapshot{}, fmt.Errorf(
				"%w: action %s range [%d,%d) is not aligned with tick spacing %d",
				ErrInvalidLegacyRecord,
				action.ID,
				action.TickLower,
				action.TickUpper,
				pool.TickSpacing,
			)
		}

		delta, err :=
			parseStorageBigInt(
				"lp action liquidity_delta",
				action.LiquidityDelta,
			)
		if err != nil {
			return domain.PoolSnapshot{}, err
		}

		switch action.Action {
		case LegacyLPActionMint:
			if delta.Sign() <= 0 {
				return domain.PoolSnapshot{}, fmt.Errorf(
					"%w: mint action %s has delta %s",
					ErrInvalidLiquiditySign,
					action.ID,
					delta,
				)
			}

		case LegacyLPActionBurn:
			if delta.Sign() >= 0 {
				return domain.PoolSnapshot{}, fmt.Errorf(
					"%w: burn action %s has delta %s",
					ErrInvalidLiquiditySign,
					action.ID,
					delta,
				)
			}

		default:
			return domain.PoolSnapshot{}, fmt.Errorf(
				"%w: action %s has unsupported type %d",
				ErrInvalidLegacyRecord,
				action.ID,
				action.Action,
			)
		}

		current := positionLiquidity[key]

		if current == nil {
			current = new(big.Int)
		}

		next := new(big.Int).Add(
			current,
			delta,
		)

		if next.Sign() < 0 {
			return domain.PoolSnapshot{}, fmt.Errorf(
				"%w: action %s makes position %s liquidity negative: current=%s delta=%s",
				ErrCheckpointMismatch,
				action.ID,
				key,
				current,
				delta,
			)
		}

		if next.Sign() == 0 {
			delete(positionLiquidity, key)
			continue
		}

		positionLiquidity[key] = next
	}

	positions := make(
		[]domain.CorePosition,
		0,
		len(positionLiquidity),
	)

	for key, liquidity := range positionLiquidity {
		position, err := domain.NewCorePosition(
			key,
			liquidity,
		)
		if err != nil {
			return domain.PoolSnapshot{}, fmt.Errorf(
				"build checkpoint position %s: %w",
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
			"%w: build complete checkpoint at block %d: %v",
			ErrCheckpointMismatch,
			scalar.BlockNumber,
			err,
		)
	}

	return snapshot, nil
}

func deriveTicksFromPositions(
	positions []domain.CorePosition,
) ([]domain.TickState, error) {
	type tickAccounting struct {
		gross *big.Int
		net   *big.Int
	}

	accounting := make(
		map[int32]*tickAccounting,
	)

	for _, position := range positions {
		key := position.Key()
		liquidity := position.Liquidity()

		lower := accounting[key.TickLower]

		if lower == nil {
			lower = &tickAccounting{
				gross: new(big.Int),
				net:   new(big.Int),
			}

			accounting[key.TickLower] = lower
		}

		lower.gross.Add(
			lower.gross,
			liquidity,
		)

		lower.net.Add(
			lower.net,
			liquidity,
		)

		upper := accounting[key.TickUpper]

		if upper == nil {
			upper = &tickAccounting{
				gross: new(big.Int),
				net:   new(big.Int),
			}

			accounting[key.TickUpper] = upper
		}

		upper.gross.Add(
			upper.gross,
			liquidity,
		)

		upper.net.Sub(
			upper.net,
			liquidity,
		)
	}

	indexes := make(
		[]int32,
		0,
		len(accounting),
	)

	for index := range accounting {
		indexes = append(
			indexes,
			index,
		)
	}

	sort.Slice(
		indexes,
		func(i, j int) bool {
			return indexes[i] < indexes[j]
		},
	)

	ticks := make(
		[]domain.TickState,
		0,
		len(indexes),
	)

	for _, index := range indexes {
		values := accounting[index]

		tick, err := domain.NewTickState(
			index,
			values.gross,
			values.net,
		)
		if err != nil {
			return nil, fmt.Errorf(
				"derive initialized tick %d: %w",
				index,
				err,
			)
		}

		ticks = append(
			ticks,
			tick,
		)
	}

	return ticks, nil
}

// normalizeLegacyText is retained for PostgreSQL CHAR columns whose values may
// contain padding spaces.
func normalizeLegacyText(
	value string,
) string {
	return strings.TrimSpace(value)
}
