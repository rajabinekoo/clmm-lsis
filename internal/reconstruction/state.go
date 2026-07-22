package reconstruction

import (
	"fmt"
	"math/big"
	"sort"

	"github.com/rajabinekoo/clmm-lsis/internal/domain"
)

// MutablePoolState is the event-replay representation of one Uniswap v3 pool.
//
// Unlike domain.PoolSnapshot, this type is intentionally mutable. It exists
// only inside the reconstruction layer and is never exposed as an empirical
// output.
//
// The state can start from:
//
//   - an empty pool before Initialize; or
//   - an existing historical PoolSnapshot checkpoint.
//
// This allows the new implementation to reuse previously indexed snapshots
// instead of replaying every event from pool creation.
type MutablePoolState struct {
	pool domain.Pool

	initialized bool

	sqrtPriceX96    *big.Int
	currentTick     int32
	activeLiquidity *big.Int

	ticks       map[int32]domain.TickState
	tickIndexes []int32

	positions map[domain.CorePositionKey]domain.CorePosition

	baseReference    domain.SnapshotReference
	hasBaseReference bool

	lastAppliedCursor *domain.EventCursor
}

// NewMutablePoolState creates an empty, uninitialized pool state.
func NewMutablePoolState(
	pool domain.Pool,
) (*MutablePoolState, error) {
	if err := pool.Validate(); err != nil {
		return nil, fmt.Errorf(
			"create mutable pool state: %w",
			err,
		)
	}

	return &MutablePoolState{
		pool: pool,

		initialized: false,

		sqrtPriceX96:    nil,
		currentTick:     0,
		activeLiquidity: new(big.Int),

		ticks:       make(map[int32]domain.TickState),
		tickIndexes: make([]int32, 0),

		positions: make(
			map[domain.CorePositionKey]domain.CorePosition,
		),
	}, nil
}

// NewMutablePoolStateFromSnapshot creates a replay state from an existing
// immutable checkpoint.
//
// The supplied snapshot is copied completely. Mutating the returned state
// cannot change the original snapshot.
func NewMutablePoolStateFromSnapshot(
	pool domain.Pool,
	snapshot domain.PoolSnapshot,
) (*MutablePoolState, error) {
	if err := pool.Validate(); err != nil {
		return nil, fmt.Errorf(
			"create mutable pool state from snapshot: %w",
			err,
		)
	}

	if err := snapshot.Validate(); err != nil {
		return nil, fmt.Errorf(
			"create mutable pool state from invalid snapshot: %w",
			err,
		)
	}

	if snapshot.PoolAddress() != pool.Address {
		return nil, fmt.Errorf(
			"%w: snapshot pool %s does not match requested pool %s",
			ErrPoolMismatch,
			snapshot.PoolAddress(),
			pool.Address,
		)
	}

	state := &MutablePoolState{
		pool: pool,

		initialized: true,

		sqrtPriceX96:    snapshot.SqrtPriceX96(),
		currentTick:     snapshot.CurrentTick(),
		activeLiquidity: snapshot.ActiveLiquidity(),

		ticks:       make(map[int32]domain.TickState),
		tickIndexes: make([]int32, 0),

		positions: make(
			map[domain.CorePositionKey]domain.CorePosition,
		),

		baseReference:    snapshot.Reference(),
		hasBaseReference: true,
	}

	for _, tick := range snapshot.Ticks() {
		state.ticks[tick.Index()] = tick
		state.tickIndexes = append(
			state.tickIndexes,
			tick.Index(),
		)
	}

	for _, position := range snapshot.Positions() {
		state.positions[position.Key()] = position
	}

	sort.Slice(
		state.tickIndexes,
		func(i, j int) bool {
			return state.tickIndexes[i] <
				state.tickIndexes[j]
		},
	)

	if state.baseReference.Boundary() ==
		domain.SnapshotAfterEvent {
		cursor, exists :=
			state.baseReference.Cursor()

		if !exists {
			return nil, fmt.Errorf(
				"after-event base snapshot has no cursor",
			)
		}

		state.setLastAppliedCursor(cursor)
	}

	return state, nil
}

func (s *MutablePoolState) Pool() domain.Pool {
	return s.pool
}

func (s *MutablePoolState) Initialized() bool {
	return s.initialized
}

func (s *MutablePoolState) SqrtPriceX96() (
	*big.Int,
	bool,
) {
	if !s.initialized {
		return nil, false
	}

	return cloneBigInt(s.sqrtPriceX96), true
}

func (s *MutablePoolState) CurrentTick() (
	int32,
	bool,
) {
	if !s.initialized {
		return 0, false
	}

	return s.currentTick, true
}

func (s *MutablePoolState) ActiveLiquidity() (
	*big.Int,
	bool,
) {
	if !s.initialized {
		return nil, false
	}

	return cloneBigInt(s.activeLiquidity), true
}

func (s *MutablePoolState) LastAppliedCursor() (
	domain.EventCursor,
	bool,
) {
	if s.lastAppliedCursor == nil {
		return domain.EventCursor{}, false
	}

	return *s.lastAppliedCursor, true
}

func (s *MutablePoolState) Position(
	key domain.CorePositionKey,
) (
	domain.CorePosition,
	bool,
) {
	position, exists := s.positions[key]
	if !exists {
		return domain.CorePosition{}, false
	}

	cloned, err := domain.NewCorePosition(
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

func (s *MutablePoolState) Tick(
	index int32,
) (
	domain.TickState,
	bool,
) {
	tick, exists := s.ticks[index]
	if !exists {
		return domain.TickState{}, false
	}

	cloned, err := domain.NewTickState(
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

func (s *MutablePoolState) validateEventOrder(
	cursor domain.EventCursor,
) error {
	if err := cursor.Validate(); err != nil {
		return err
	}

	if s.lastAppliedCursor != nil {
		if cursor.Compare(*s.lastAppliedCursor) <= 0 {
			return fmt.Errorf(
				"%w: event cursor %s is not after last applied cursor %s",
				ErrOutOfOrder,
				cursor,
				*s.lastAppliedCursor,
			)
		}

		return nil
	}

	if !s.hasBaseReference {
		return nil
	}

	switch s.baseReference.Boundary() {
	case domain.SnapshotBeforeEvent:
		baseCursor, exists :=
			s.baseReference.Cursor()

		if !exists {
			return fmt.Errorf(
				"before-event base snapshot has no cursor",
			)
		}

		if cursor.Compare(baseCursor) < 0 {
			return fmt.Errorf(
				"%w: event cursor %s occurs before base snapshot cursor %s",
				ErrOutOfOrder,
				cursor,
				baseCursor,
			)
		}

	case domain.SnapshotAfterEvent:
		baseCursor, exists :=
			s.baseReference.Cursor()

		if !exists {
			return fmt.Errorf(
				"after-event base snapshot has no cursor",
			)
		}

		if cursor.Compare(baseCursor) <= 0 {
			return fmt.Errorf(
				"%w: event cursor %s is not after base snapshot cursor %s",
				ErrOutOfOrder,
				cursor,
				baseCursor,
			)
		}

	case domain.SnapshotBlockEnd:
		if cursor.BlockNumber <=
			s.baseReference.BlockNumber() {
			return fmt.Errorf(
				"%w: event block %d must be after block-end checkpoint %d",
				ErrOutOfOrder,
				cursor.BlockNumber,
				s.baseReference.BlockNumber(),
			)
		}

	default:
		return fmt.Errorf(
			"unsupported base snapshot boundary %s",
			s.baseReference.Boundary(),
		)
	}

	return nil
}

func (s *MutablePoolState) setLastAppliedCursor(
	cursor domain.EventCursor,
) {
	copied := cursor
	s.lastAppliedCursor = &copied
}

func (s *MutablePoolState) tickOrEmpty(
	index int32,
) domain.TickState {
	tick, exists := s.ticks[index]
	if exists {
		return tick
	}

	return domain.EmptyTickState(index)
}

// setTick inserts, replaces or removes one tick while keeping tickIndexes
// sorted.
//
// A zero-gross tick is no longer initialized and must not remain in the state.
func (s *MutablePoolState) setTick(
	tick domain.TickState,
) {
	index := tick.Index()

	position := sort.Search(
		len(s.tickIndexes),
		func(position int) bool {
			return s.tickIndexes[position] >= index
		},
	)

	exists := position < len(s.tickIndexes) &&
		s.tickIndexes[position] == index

	if !tick.Initialized() {
		delete(s.ticks, index)

		if exists {
			s.tickIndexes = append(
				s.tickIndexes[:position],
				s.tickIndexes[position+1:]...,
			)
		}

		return
	}

	s.ticks[index] = tick

	if exists {
		return
	}

	s.tickIndexes = append(
		s.tickIndexes,
		0,
	)

	copy(
		s.tickIndexes[position+1:],
		s.tickIndexes[position:],
	)

	s.tickIndexes[position] = index
}
