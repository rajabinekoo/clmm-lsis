package domain

import (
	"fmt"
	"math/big"
	"sort"
)

// PoolSnapshot is one complete reconstructed pool state.
//
// The snapshot contains sufficient information for counterfactual liquidity
// removal and exact-input swap simulation. Every stored tick and position is
// cloned during construction so callers cannot mutate the snapshot indirectly.
type PoolSnapshot struct {
	poolAddress     Address
	reference       SnapshotReference
	sqrtPriceX96    *big.Int
	currentTick     int32
	activeLiquidity *big.Int
	ticks           map[int32]TickState
	positions       map[CorePositionKey]CorePosition
}

func NewPoolSnapshot(
	poolAddress Address,
	reference SnapshotReference,
	sqrtPriceX96 *big.Int,
	currentTick int32,
	activeLiquidity *big.Int,
	ticks []TickState,
	positions []CorePosition,
) (PoolSnapshot, error) {
	snapshot := PoolSnapshot{
		poolAddress:     poolAddress,
		reference:       reference,
		sqrtPriceX96:    cloneBigInt(sqrtPriceX96),
		currentTick:     currentTick,
		activeLiquidity: cloneBigInt(activeLiquidity),
		ticks:           make(map[int32]TickState, len(ticks)),
		positions:       make(map[CorePositionKey]CorePosition, len(positions)),
	}

	for _, tick := range ticks {
		if _, exists := snapshot.ticks[tick.Index()]; exists {
			return PoolSnapshot{}, fmt.Errorf(
				"duplicate tick state %d",
				tick.Index(),
			)
		}

		clonedTick, err := NewTickState(
			tick.Index(),
			tick.LiquidityGross(),
			tick.LiquidityNet(),
		)
		if err != nil {
			return PoolSnapshot{}, fmt.Errorf(
				"copy tick %d: %w",
				tick.Index(),
				err,
			)
		}

		snapshot.ticks[tick.Index()] = clonedTick
	}

	for _, position := range positions {
		key := position.Key()

		if _, exists := snapshot.positions[key]; exists {
			return PoolSnapshot{}, fmt.Errorf(
				"duplicate position %s",
				key,
			)
		}

		clonedPosition, err := NewCorePosition(
			key,
			position.Liquidity(),
		)
		if err != nil {
			return PoolSnapshot{}, fmt.Errorf(
				"copy position %s: %w",
				key,
				err,
			)
		}

		snapshot.positions[key] = clonedPosition
	}

	if err := snapshot.Validate(); err != nil {
		return PoolSnapshot{}, err
	}

	return snapshot, nil
}

func (s PoolSnapshot) Validate() error {
	if s.poolAddress.IsZero() {
		return fmt.Errorf("snapshot pool address is required")
	}

	if err := s.reference.Validate(); err != nil {
		return fmt.Errorf("snapshot reference: %w", err)
	}

	if err := requirePositiveBigInt(
		"snapshot sqrt price X96",
		s.sqrtPriceX96,
	); err != nil {
		return err
	}

	if err := requireNonNegativeBigInt(
		"snapshot active liquidity",
		s.activeLiquidity,
	); err != nil {
		return err
	}

	for index, tick := range s.ticks {
		if index != tick.Index() {
			return fmt.Errorf(
				"tick map key %d does not match tick index %d",
				index,
				tick.Index(),
			)
		}

		if err := tick.Validate(); err != nil {
			return fmt.Errorf("snapshot tick %d: %w", index, err)
		}

		if !tick.Initialized() {
			return fmt.Errorf(
				"snapshot must not store uninitialized tick %d",
				index,
			)
		}
	}

	for key, position := range s.positions {
		if key != position.Key() {
			return fmt.Errorf(
				"position map key %s does not match position key %s",
				key,
				position.Key(),
			)
		}

		if key.PoolAddress != s.poolAddress {
			return fmt.Errorf(
				"position %s belongs to pool %s, snapshot pool is %s",
				key,
				key.PoolAddress,
				s.poolAddress,
			)
		}

		if err := position.Validate(); err != nil {
			return fmt.Errorf(
				"snapshot position %s: %w",
				key,
				err,
			)
		}
	}

	if err := s.validateDerivedLiquidityAccounting(); err != nil {
		return err
	}

	return nil
}

func (s PoolSnapshot) validateDerivedLiquidityAccounting() error {
	expectedActiveLiquidity := new(big.Int)
	expectedTicks := make(map[int32]*derivedTickState)

	for _, position := range s.positions {
		key := position.Key()
		liquidity := position.Liquidity()

		if position.IsActiveAt(s.currentTick) {
			expectedActiveLiquidity.Add(
				expectedActiveLiquidity,
				liquidity,
			)
		}

		lower := expectedTicks[key.TickLower]
		if lower == nil {
			lower = newDerivedTickState()
			expectedTicks[key.TickLower] = lower
		}

		lower.gross.Add(lower.gross, liquidity)
		lower.net.Add(lower.net, liquidity)

		upper := expectedTicks[key.TickUpper]
		if upper == nil {
			upper = newDerivedTickState()
			expectedTicks[key.TickUpper] = upper
		}

		upper.gross.Add(upper.gross, liquidity)
		upper.net.Sub(upper.net, liquidity)
	}

	if expectedActiveLiquidity.Cmp(s.activeLiquidity) != 0 {
		return fmt.Errorf(
			"snapshot active liquidity mismatch: stored=%s derived=%s",
			s.activeLiquidity,
			expectedActiveLiquidity,
		)
	}

	if len(expectedTicks) != len(s.ticks) {
		return fmt.Errorf(
			"snapshot initialized tick count mismatch: stored=%d derived=%d",
			len(s.ticks),
			len(expectedTicks),
		)
	}

	for index, expected := range expectedTicks {
		actual, exists := s.ticks[index]
		if !exists {
			return fmt.Errorf(
				"snapshot is missing derived initialized tick %d",
				index,
			)
		}

		if actual.LiquidityGross().Cmp(expected.gross) != 0 {
			return fmt.Errorf(
				"tick %d liquidity gross mismatch: stored=%s derived=%s",
				index,
				actual.LiquidityGross(),
				expected.gross,
			)
		}

		if actual.LiquidityNet().Cmp(expected.net) != 0 {
			return fmt.Errorf(
				"tick %d liquidity net mismatch: stored=%s derived=%s",
				index,
				actual.LiquidityNet(),
				expected.net,
			)
		}
	}

	return nil
}

func (s PoolSnapshot) PoolAddress() Address {
	return s.poolAddress
}

func (s PoolSnapshot) Reference() SnapshotReference {
	return s.reference
}

func (s PoolSnapshot) SqrtPriceX96() *big.Int {
	return cloneBigInt(s.sqrtPriceX96)
}

func (s PoolSnapshot) CurrentTick() int32 {
	return s.currentTick
}

func (s PoolSnapshot) ActiveLiquidity() *big.Int {
	return cloneBigInt(s.activeLiquidity)
}

func (s PoolSnapshot) Tick(
	index int32,
) (TickState, bool) {
	tick, exists := s.ticks[index]

	if !exists {
		return TickState{}, false
	}

	cloned, err := NewTickState(
		tick.Index(),
		tick.LiquidityGross(),
		tick.LiquidityNet(),
	)
	if err != nil {
		panic(fmt.Sprintf(
			"clone previously validated tick %d: %v",
			index,
			err,
		))
	}

	return cloned, true
}

func (s PoolSnapshot) Position(
	key CorePositionKey,
) (CorePosition, bool) {
	position, exists := s.positions[key]

	if !exists {
		return CorePosition{}, false
	}

	cloned, err := NewCorePosition(
		position.Key(),
		position.Liquidity(),
	)
	if err != nil {
		panic(fmt.Sprintf(
			"clone previously validated position %s: %v",
			key,
			err,
		))
	}

	return cloned, true
}

// Ticks returns initialized ticks in ascending tick order.
func (s PoolSnapshot) Ticks() []TickState {
	ticks := make([]TickState, 0, len(s.ticks))

	for _, tick := range s.ticks {
		cloned, err := NewTickState(
			tick.Index(),
			tick.LiquidityGross(),
			tick.LiquidityNet(),
		)
		if err != nil {
			panic(fmt.Sprintf(
				"clone previously validated tick %d: %v",
				tick.Index(),
				err,
			))
		}

		ticks = append(ticks, cloned)
	}

	sort.Slice(ticks, func(i, j int) bool {
		return ticks[i].Index() < ticks[j].Index()
	})

	return ticks
}

// Positions returns positions in deterministic key order.
func (s PoolSnapshot) Positions() []CorePosition {
	positions := make([]CorePosition, 0, len(s.positions))

	for _, position := range s.positions {
		cloned, err := NewCorePosition(
			position.Key(),
			position.Liquidity(),
		)
		if err != nil {
			panic(fmt.Sprintf(
				"clone previously validated position %s: %v",
				position.Key(),
				err,
			))
		}

		positions = append(positions, cloned)
	}

	sort.Slice(positions, func(i, j int) bool {
		left := positions[i].Key()
		right := positions[j].Key()

		if left.Owner != right.Owner {
			return left.Owner < right.Owner
		}

		if left.TickLower != right.TickLower {
			return left.TickLower < right.TickLower
		}

		return left.TickUpper < right.TickUpper
	})

	return positions
}

func (s PoolSnapshot) ActivePositions() []CorePosition {
	positions := s.Positions()
	active := make([]CorePosition, 0, len(positions))

	for _, position := range positions {
		if position.IsActiveAt(s.currentTick) {
			active = append(active, position)
		}
	}

	return active
}

func (s PoolSnapshot) Clone() PoolSnapshot {
	cloned, err := NewPoolSnapshot(
		s.poolAddress,
		s.reference,
		s.sqrtPriceX96,
		s.currentTick,
		s.activeLiquidity,
		s.Ticks(),
		s.Positions(),
	)
	if err != nil {
		panic(fmt.Sprintf(
			"clone previously validated pool snapshot: %v",
			err,
		))
	}

	return cloned
}

type derivedTickState struct {
	gross *big.Int
	net   *big.Int
}

func newDerivedTickState() *derivedTickState {
	return &derivedTickState{
		gross: new(big.Int),
		net:   new(big.Int),
	}
}
