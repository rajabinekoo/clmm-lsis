package reconstruction_test

import (
	"errors"
	"math/big"
	"testing"

	"github.com/rajabinekoo/clmm-lsis/internal/domain"
	"github.com/rajabinekoo/clmm-lsis/internal/reconstruction"
)

func TestMutablePoolStateInitialize(
	t *testing.T,
) {
	t.Parallel()

	pool := newReconstructionTestPool(t)
	state := newMutableState(t, pool)

	cursor := reconstructionCursor(
		100,
		0,
		0,
	)

	initializeStateAtTick(
		t,
		state,
		pool,
		cursor,
		0,
	)

	if !state.Initialized() {
		t.Fatal(
			"Initialized() = false, want true",
		)
	}

	currentTick, exists :=
		state.CurrentTick()

	if !exists {
		t.Fatal(
			"CurrentTick() expected initialized value",
		)
	}

	if currentTick != 0 {
		t.Fatalf(
			"CurrentTick() = %d, want 0",
			currentTick,
		)
	}

	activeLiquidity, exists :=
		state.ActiveLiquidity()

	if !exists {
		t.Fatal(
			"ActiveLiquidity() expected initialized value",
		)
	}

	assertReconstructionBigIntEqual(
		t,
		activeLiquidity,
		new(big.Int),
	)

	snapshot, err :=
		state.SnapshotAfterLastEvent()
	if err != nil {
		t.Fatalf(
			"SnapshotAfterLastEvent() error = %v",
			err,
		)
	}

	if len(snapshot.Positions()) != 0 {
		t.Fatalf(
			"position count = %d, want 0",
			len(snapshot.Positions()),
		)
	}

	if len(snapshot.Ticks()) != 0 {
		t.Fatalf(
			"tick count = %d, want 0",
			len(snapshot.Ticks()),
		)
	}
}

func TestMutablePoolStateRejectsMintBeforeInitialize(
	t *testing.T,
) {
	t.Parallel()

	pool := newReconstructionTestPool(t)
	state := newMutableState(t, pool)

	event := newMintPoolEvent(
		t,
		pool,
		reconstructionCursor(100, 0, 0),
		reconstructionOwner(t, 1),
		-100,
		100,
		big.NewInt(1_000),
	)

	err := state.Apply(event)

	if !errors.Is(
		err,
		reconstruction.ErrNotInitialized,
	) {
		t.Fatalf(
			"Apply() error = %v, want ErrNotInitialized",
			err,
		)
	}

	if state.Initialized() {
		t.Fatal(
			"failed mint initialized the state",
		)
	}

	if _, exists :=
		state.LastAppliedCursor(); exists {
		t.Fatal(
			"failed mint advanced event cursor",
		)
	}
}

func TestMutablePoolStateMintUpdatesPositionTicksAndLiquidity(
	t *testing.T,
) {
	t.Parallel()

	pool := newReconstructionTestPool(t)
	state := newMutableState(t, pool)

	initializeStateAtTick(
		t,
		state,
		pool,
		reconstructionCursor(100, 0, 0),
		0,
	)

	owner := reconstructionOwner(t, 1)

	mint := newMintPoolEvent(
		t,
		pool,
		reconstructionCursor(101, 0, 0),
		owner,
		-100,
		100,
		big.NewInt(1_000),
	)

	mustApplyEvent(
		t,
		state,
		mint,
	)

	key := domain.CorePositionKey{
		PoolAddress: pool.Address,
		Owner:       owner,
		TickLower:   -100,
		TickUpper:   100,
	}

	position, exists :=
		state.Position(key)

	if !exists {
		t.Fatal(
			"minted position not found",
		)
	}

	assertReconstructionBigIntEqual(
		t,
		position.Liquidity(),
		big.NewInt(1_000),
	)

	activeLiquidity, exists :=
		state.ActiveLiquidity()

	if !exists {
		t.Fatal(
			"ActiveLiquidity() expected value",
		)
	}

	assertReconstructionBigIntEqual(
		t,
		activeLiquidity,
		big.NewInt(1_000),
	)

	lowerTick, exists :=
		state.Tick(-100)

	if !exists {
		t.Fatal(
			"lower tick not found",
		)
	}

	assertReconstructionBigIntEqual(
		t,
		lowerTick.LiquidityGross(),
		big.NewInt(1_000),
	)

	assertReconstructionBigIntEqual(
		t,
		lowerTick.LiquidityNet(),
		big.NewInt(1_000),
	)

	upperTick, exists :=
		state.Tick(100)

	if !exists {
		t.Fatal(
			"upper tick not found",
		)
	}

	assertReconstructionBigIntEqual(
		t,
		upperTick.LiquidityGross(),
		big.NewInt(1_000),
	)

	assertReconstructionBigIntEqual(
		t,
		upperTick.LiquidityNet(),
		big.NewInt(-1_000),
	)
}

func TestInactiveMintDoesNotChangeActiveLiquidity(
	t *testing.T,
) {
	t.Parallel()

	pool := newReconstructionTestPool(t)
	state := newMutableState(t, pool)

	initializeStateAtTick(
		t,
		state,
		pool,
		reconstructionCursor(100, 0, 0),
		0,
	)

	mint := newMintPoolEvent(
		t,
		pool,
		reconstructionCursor(101, 0, 0),
		reconstructionOwner(t, 1),
		100,
		200,
		big.NewInt(750),
	)

	mustApplyEvent(
		t,
		state,
		mint,
	)

	activeLiquidity, exists :=
		state.ActiveLiquidity()

	if !exists {
		t.Fatal(
			"ActiveLiquidity() expected value",
		)
	}

	assertReconstructionBigIntEqual(
		t,
		activeLiquidity,
		new(big.Int),
	)

	snapshot, err :=
		state.SnapshotAfterLastEvent()
	if err != nil {
		t.Fatalf(
			"SnapshotAfterLastEvent() error = %v",
			err,
		)
	}

	if len(snapshot.ActivePositions()) != 0 {
		t.Fatalf(
			"active position count = %d, want 0",
			len(snapshot.ActivePositions()),
		)
	}
}

func TestPartialBurnUpdatesAllLiquidityAccounting(
	t *testing.T,
) {
	t.Parallel()

	pool := newReconstructionTestPool(t)
	state := newMutableState(t, pool)

	initializeStateAtTick(
		t,
		state,
		pool,
		reconstructionCursor(100, 0, 0),
		0,
	)

	owner := reconstructionOwner(t, 1)

	mustApplyEvent(
		t,
		state,
		newMintPoolEvent(
			t,
			pool,
			reconstructionCursor(101, 0, 0),
			owner,
			-100,
			100,
			big.NewInt(1_000),
		),
	)

	mustApplyEvent(
		t,
		state,
		newBurnPoolEvent(
			t,
			pool,
			reconstructionCursor(102, 0, 0),
			owner,
			-100,
			100,
			big.NewInt(250),
		),
	)

	key := domain.CorePositionKey{
		PoolAddress: pool.Address,
		Owner:       owner,
		TickLower:   -100,
		TickUpper:   100,
	}

	position, exists :=
		state.Position(key)

	if !exists {
		t.Fatal(
			"partially burned position was removed",
		)
	}

	assertReconstructionBigIntEqual(
		t,
		position.Liquidity(),
		big.NewInt(750),
	)

	activeLiquidity, exists :=
		state.ActiveLiquidity()

	if !exists {
		t.Fatal(
			"ActiveLiquidity() expected value",
		)
	}

	assertReconstructionBigIntEqual(
		t,
		activeLiquidity,
		big.NewInt(750),
	)

	lowerTick, exists :=
		state.Tick(-100)

	if !exists {
		t.Fatal(
			"lower tick removed after partial burn",
		)
	}

	assertReconstructionBigIntEqual(
		t,
		lowerTick.LiquidityGross(),
		big.NewInt(750),
	)

	assertReconstructionBigIntEqual(
		t,
		lowerTick.LiquidityNet(),
		big.NewInt(750),
	)

	upperTick, exists :=
		state.Tick(100)

	if !exists {
		t.Fatal(
			"upper tick removed after partial burn",
		)
	}

	assertReconstructionBigIntEqual(
		t,
		upperTick.LiquidityGross(),
		big.NewInt(750),
	)

	assertReconstructionBigIntEqual(
		t,
		upperTick.LiquidityNet(),
		big.NewInt(-750),
	)

	if _, err :=
		state.SnapshotAfterLastEvent(); err != nil {
		t.Fatalf(
			"SnapshotAfterLastEvent() error = %v",
			err,
		)
	}
}

func TestFullBurnRemovesPositionAndTicks(
	t *testing.T,
) {
	t.Parallel()

	pool := newReconstructionTestPool(t)
	state := newMutableState(t, pool)

	initializeStateAtTick(
		t,
		state,
		pool,
		reconstructionCursor(100, 0, 0),
		0,
	)

	owner := reconstructionOwner(t, 1)

	mustApplyEvent(
		t,
		state,
		newMintPoolEvent(
			t,
			pool,
			reconstructionCursor(101, 0, 0),
			owner,
			-100,
			100,
			big.NewInt(1_000),
		),
	)

	mustApplyEvent(
		t,
		state,
		newBurnPoolEvent(
			t,
			pool,
			reconstructionCursor(102, 0, 0),
			owner,
			-100,
			100,
			big.NewInt(1_000),
		),
	)

	key := domain.CorePositionKey{
		PoolAddress: pool.Address,
		Owner:       owner,
		TickLower:   -100,
		TickUpper:   100,
	}

	if _, exists :=
		state.Position(key); exists {
		t.Fatal(
			"fully burned position still exists",
		)
	}

	if _, exists :=
		state.Tick(-100); exists {
		t.Fatal(
			"zero-gross lower tick still exists",
		)
	}

	if _, exists :=
		state.Tick(100); exists {
		t.Fatal(
			"zero-gross upper tick still exists",
		)
	}

	activeLiquidity, exists :=
		state.ActiveLiquidity()

	if !exists {
		t.Fatal(
			"ActiveLiquidity() expected value",
		)
	}

	assertReconstructionBigIntEqual(
		t,
		activeLiquidity,
		new(big.Int),
	)

	snapshot, err :=
		state.SnapshotAfterLastEvent()
	if err != nil {
		t.Fatalf(
			"SnapshotAfterLastEvent() error = %v",
			err,
		)
	}

	if len(snapshot.Positions()) != 0 {
		t.Fatalf(
			"position count = %d, want 0",
			len(snapshot.Positions()),
		)
	}

	if len(snapshot.Ticks()) != 0 {
		t.Fatalf(
			"tick count = %d, want 0",
			len(snapshot.Ticks()),
		)
	}
}

func TestSwapWithoutTickCrossingPreservesActiveLiquidity(
	t *testing.T,
) {
	t.Parallel()

	pool := newReconstructionTestPool(t)
	state := newMutableState(t, pool)

	initializeStateAtTick(
		t,
		state,
		pool,
		reconstructionCursor(100, 0, 0),
		0,
	)

	mustApplyEvent(
		t,
		state,
		newMintPoolEvent(
			t,
			pool,
			reconstructionCursor(101, 0, 0),
			reconstructionOwner(t, 1),
			-100,
			100,
			big.NewInt(1_000),
		),
	)

	swap := newSwapPoolEvent(
		t,
		pool,
		reconstructionCursor(102, 0, 0),
		true,
		mustReconstructionSqrtPrice(t, -50),
		-50,
		big.NewInt(1_000),
	)

	mustApplyEvent(
		t,
		state,
		swap,
	)

	currentTick, exists :=
		state.CurrentTick()

	if !exists {
		t.Fatal(
			"CurrentTick() expected value",
		)
	}

	if currentTick != -50 {
		t.Fatalf(
			"CurrentTick() = %d, want -50",
			currentTick,
		)
	}

	activeLiquidity, exists :=
		state.ActiveLiquidity()

	if !exists {
		t.Fatal(
			"ActiveLiquidity() expected value",
		)
	}

	assertReconstructionBigIntEqual(
		t,
		activeLiquidity,
		big.NewInt(1_000),
	)

	if _, err :=
		state.SnapshotAfterLastEvent(); err != nil {
		t.Fatalf(
			"SnapshotAfterLastEvent() error = %v",
			err,
		)
	}
}

func TestZeroForOneSwapCrossesTickAndChangesLiquidity(
	t *testing.T,
) {
	t.Parallel()

	pool := newReconstructionTestPool(t)
	state := newMutableState(t, pool)

	initializeStateAtTick(
		t,
		state,
		pool,
		reconstructionCursor(100, 0, 0),
		0,
	)

	ownerA := reconstructionOwner(t, 1)
	ownerB := reconstructionOwner(t, 2)

	mustApplyEvent(
		t,
		state,
		newMintPoolEvent(
			t,
			pool,
			reconstructionCursor(101, 0, 0),
			ownerA,
			-100,
			100,
			big.NewInt(1_000),
		),
	)

	mustApplyEvent(
		t,
		state,
		newMintPoolEvent(
			t,
			pool,
			reconstructionCursor(101, 0, 1),
			ownerB,
			-200,
			-100,
			big.NewInt(500),
		),
	)

	swap := newSwapPoolEvent(
		t,
		pool,
		reconstructionCursor(102, 0, 0),
		true,
		mustReconstructionSqrtPrice(t, -150),
		-150,
		big.NewInt(500),
	)

	mustApplyEvent(
		t,
		state,
		swap,
	)

	activeLiquidity, exists :=
		state.ActiveLiquidity()

	if !exists {
		t.Fatal(
			"ActiveLiquidity() expected value",
		)
	}

	assertReconstructionBigIntEqual(
		t,
		activeLiquidity,
		big.NewInt(500),
	)

	snapshot, err :=
		state.SnapshotAfterLastEvent()
	if err != nil {
		t.Fatalf(
			"SnapshotAfterLastEvent() error = %v",
			err,
		)
	}

	activePositions :=
		snapshot.ActivePositions()

	if len(activePositions) != 1 {
		t.Fatalf(
			"active position count = %d, want 1",
			len(activePositions),
		)
	}

	if activePositions[0].Key().Owner != ownerB {
		t.Fatalf(
			"active owner = %s, want %s",
			activePositions[0].Key().Owner,
			ownerB,
		)
	}
}

func TestOneForZeroSwapCrossesTickAndChangesLiquidity(
	t *testing.T,
) {
	t.Parallel()

	pool := newReconstructionTestPool(t)
	state := newMutableState(t, pool)

	initializeStateAtTick(
		t,
		state,
		pool,
		reconstructionCursor(100, 0, 0),
		0,
	)

	ownerA := reconstructionOwner(t, 1)
	ownerB := reconstructionOwner(t, 2)

	mustApplyEvent(
		t,
		state,
		newMintPoolEvent(
			t,
			pool,
			reconstructionCursor(101, 0, 0),
			ownerA,
			-100,
			100,
			big.NewInt(1_000),
		),
	)

	mustApplyEvent(
		t,
		state,
		newMintPoolEvent(
			t,
			pool,
			reconstructionCursor(101, 0, 1),
			ownerB,
			100,
			200,
			big.NewInt(700),
		),
	)

	swap := newSwapPoolEvent(
		t,
		pool,
		reconstructionCursor(102, 0, 0),
		false,
		mustReconstructionSqrtPrice(t, 150),
		150,
		big.NewInt(700),
	)

	mustApplyEvent(
		t,
		state,
		swap,
	)

	activeLiquidity, exists :=
		state.ActiveLiquidity()

	if !exists {
		t.Fatal(
			"ActiveLiquidity() expected value",
		)
	}

	assertReconstructionBigIntEqual(
		t,
		activeLiquidity,
		big.NewInt(700),
	)

	snapshot, err :=
		state.SnapshotAfterLastEvent()
	if err != nil {
		t.Fatalf(
			"SnapshotAfterLastEvent() error = %v",
			err,
		)
	}

	activePositions :=
		snapshot.ActivePositions()

	if len(activePositions) != 1 {
		t.Fatalf(
			"active position count = %d, want 1",
			len(activePositions),
		)
	}

	if activePositions[0].Key().Owner != ownerB {
		t.Fatalf(
			"active owner = %s, want %s",
			activePositions[0].Key().Owner,
			ownerB,
		)
	}
}

func TestZeroForOneSwapAcceptsExactBoundaryTickMinusOne(
	t *testing.T,
) {
	t.Parallel()

	pool := newReconstructionTestPool(t)
	state := newMutableState(t, pool)

	initializeStateAtTick(
		t,
		state,
		pool,
		reconstructionCursor(100, 0, 0),
		0,
	)

	mustApplyEvent(
		t,
		state,
		newMintPoolEvent(
			t,
			pool,
			reconstructionCursor(101, 0, 0),
			reconstructionOwner(t, 1),
			-100,
			100,
			big.NewInt(1_000),
		),
	)

	mustApplyEvent(
		t,
		state,
		newMintPoolEvent(
			t,
			pool,
			reconstructionCursor(101, 0, 1),
			reconstructionOwner(t, 2),
			-200,
			-100,
			big.NewInt(500),
		),
	)

	// The sqrt price lies exactly on tick -100, but after a right-to-left
	// crossing Uniswap stores slot0.tick as -101.
	swap := newSwapPoolEvent(
		t,
		pool,
		reconstructionCursor(102, 0, 0),
		true,
		mustReconstructionSqrtPrice(t, -100),
		-101,
		big.NewInt(500),
	)

	mustApplyEvent(
		t,
		state,
		swap,
	)

	currentTick, exists :=
		state.CurrentTick()

	if !exists {
		t.Fatal(
			"CurrentTick() expected value",
		)
	}

	if currentTick != -101 {
		t.Fatalf(
			"CurrentTick() = %d, want -101",
			currentTick,
		)
	}

	if _, err :=
		state.SnapshotAfterLastEvent(); err != nil {
		t.Fatalf(
			"SnapshotAfterLastEvent() error = %v",
			err,
		)
	}
}

func TestSwapRejectsIncorrectEmittedLiquidityAtomically(
	t *testing.T,
) {
	t.Parallel()

	pool := newReconstructionTestPool(t)
	state := newMutableState(t, pool)

	initializeStateAtTick(
		t,
		state,
		pool,
		reconstructionCursor(100, 0, 0),
		0,
	)

	mustApplyEvent(
		t,
		state,
		newMintPoolEvent(
			t,
			pool,
			reconstructionCursor(101, 0, 0),
			reconstructionOwner(t, 1),
			-100,
			100,
			big.NewInt(1_000),
		),
	)

	beforeFailure, err :=
		state.SnapshotAfterLastEvent()
	if err != nil {
		t.Fatalf(
			"SnapshotAfterLastEvent() error = %v",
			err,
		)
	}

	failedCursor :=
		reconstructionCursor(102, 0, 0)

	invalidSwap := newSwapPoolEvent(
		t,
		pool,
		failedCursor,
		true,
		mustReconstructionSqrtPrice(t, -50),
		-50,
		big.NewInt(999),
	)

	err = state.Apply(invalidSwap)

	if !errors.Is(
		err,
		reconstruction.ErrInconsistentSwap,
	) {
		t.Fatalf(
			"Apply() error = %v, want ErrInconsistentSwap",
			err,
		)
	}

	afterFailure, err :=
		state.SnapshotAfterLastEvent()
	if err != nil {
		t.Fatalf(
			"SnapshotAfterLastEvent() error = %v",
			err,
		)
	}

	assertSnapshotStateEqual(
		t,
		afterFailure,
		beforeFailure,
	)

	lastCursor, exists :=
		state.LastAppliedCursor()

	if !exists {
		t.Fatal(
			"LastAppliedCursor() expected value",
		)
	}

	if lastCursor.Compare(
		reconstructionCursor(101, 0, 0),
	) != 0 {
		t.Fatalf(
			"last cursor = %s, want 101:0:0",
			lastCursor,
		)
	}

	// The same cursor remains usable because the failed event was never
	// committed.
	validSwap := newSwapPoolEvent(
		t,
		pool,
		failedCursor,
		true,
		mustReconstructionSqrtPrice(t, -50),
		-50,
		big.NewInt(1_000),
	)

	mustApplyEvent(
		t,
		state,
		validSwap,
	)
}

func TestEventsMustBeAppliedInStrictCursorOrder(
	t *testing.T,
) {
	t.Parallel()

	pool := newReconstructionTestPool(t)
	state := newMutableState(t, pool)

	initializeStateAtTick(
		t,
		state,
		pool,
		reconstructionCursor(100, 0, 0),
		0,
	)

	owner := reconstructionOwner(t, 1)

	mustApplyEvent(
		t,
		state,
		newMintPoolEvent(
			t,
			pool,
			reconstructionCursor(101, 0, 2),
			owner,
			-100,
			100,
			big.NewInt(1_000),
		),
	)

	outOfOrder := newBurnPoolEvent(
		t,
		pool,
		reconstructionCursor(101, 0, 1),
		owner,
		-100,
		100,
		big.NewInt(100),
	)

	err := state.Apply(outOfOrder)

	if !errors.Is(
		err,
		reconstruction.ErrOutOfOrder,
	) {
		t.Fatalf(
			"Apply() error = %v, want ErrOutOfOrder",
			err,
		)
	}

	duplicateCursor := newBurnPoolEvent(
		t,
		pool,
		reconstructionCursor(101, 0, 2),
		owner,
		-100,
		100,
		big.NewInt(100),
	)

	err = state.Apply(duplicateCursor)

	if !errors.Is(
		err,
		reconstruction.ErrOutOfOrder,
	) {
		t.Fatalf(
			"duplicate Apply() error = %v, want ErrOutOfOrder",
			err,
		)
	}
}

func TestFailedBurnDoesNotMutateStateOrAdvanceCursor(
	t *testing.T,
) {
	t.Parallel()

	pool := newReconstructionTestPool(t)
	state := newMutableState(t, pool)

	initializeStateAtTick(
		t,
		state,
		pool,
		reconstructionCursor(100, 0, 0),
		0,
	)

	owner := reconstructionOwner(t, 1)

	mustApplyEvent(
		t,
		state,
		newMintPoolEvent(
			t,
			pool,
			reconstructionCursor(101, 0, 0),
			owner,
			-100,
			100,
			big.NewInt(1_000),
		),
	)

	beforeFailure, err :=
		state.SnapshotAfterLastEvent()
	if err != nil {
		t.Fatalf(
			"SnapshotAfterLastEvent() error = %v",
			err,
		)
	}

	failedCursor :=
		reconstructionCursor(102, 0, 0)

	excessiveBurn := newBurnPoolEvent(
		t,
		pool,
		failedCursor,
		owner,
		-100,
		100,
		big.NewInt(1_001),
	)

	if err := state.Apply(
		excessiveBurn,
	); err == nil {
		t.Fatal(
			"Apply(excessive burn) expected error",
		)
	}

	afterFailure, err :=
		state.SnapshotAfterLastEvent()
	if err != nil {
		t.Fatalf(
			"SnapshotAfterLastEvent() error = %v",
			err,
		)
	}

	assertSnapshotStateEqual(
		t,
		afterFailure,
		beforeFailure,
	)

	lastCursor, exists :=
		state.LastAppliedCursor()

	if !exists {
		t.Fatal(
			"LastAppliedCursor() expected value",
		)
	}

	if lastCursor.Compare(
		reconstructionCursor(101, 0, 0),
	) != 0 {
		t.Fatalf(
			"last cursor = %s, want 101:0:0",
			lastCursor,
		)
	}

	validBurn := newBurnPoolEvent(
		t,
		pool,
		failedCursor,
		owner,
		-100,
		100,
		big.NewInt(250),
	)

	mustApplyEvent(
		t,
		state,
		validBurn,
	)

	key := domain.CorePositionKey{
		PoolAddress: pool.Address,
		Owner:       owner,
		TickLower:   -100,
		TickUpper:   100,
	}

	position, exists :=
		state.Position(key)

	if !exists {
		t.Fatal(
			"position missing after valid partial burn",
		)
	}

	assertReconstructionBigIntEqual(
		t,
		position.Liquidity(),
		big.NewInt(750),
	)
}

func TestCheckpointReplayMatchesDirectReplay(
	t *testing.T,
) {
	t.Parallel()

	pool := newReconstructionTestPool(t)

	directState :=
		newMutableState(t, pool)

	initializeStateAtTick(
		t,
		directState,
		pool,
		reconstructionCursor(100, 0, 0),
		0,
	)

	ownerA := reconstructionOwner(t, 1)
	ownerB := reconstructionOwner(t, 2)

	mintA := newMintPoolEvent(
		t,
		pool,
		reconstructionCursor(101, 0, 0),
		ownerA,
		-100,
		100,
		big.NewInt(1_000),
	)

	mintB := newMintPoolEvent(
		t,
		pool,
		reconstructionCursor(101, 0, 1),
		ownerB,
		-200,
		-100,
		big.NewInt(500),
	)

	swap := newSwapPoolEvent(
		t,
		pool,
		reconstructionCursor(102, 0, 0),
		true,
		mustReconstructionSqrtPrice(t, -150),
		-150,
		big.NewInt(500),
	)

	mustApplyEvent(
		t,
		directState,
		mintA,
	)

	mustApplyEvent(
		t,
		directState,
		mintB,
	)

	mustApplyEvent(
		t,
		directState,
		swap,
	)

	checkpoint, err :=
		directState.SnapshotAtBlockEnd(102)
	if err != nil {
		t.Fatalf(
			"SnapshotAtBlockEnd() error = %v",
			err,
		)
	}

	checkpointState, err :=
		reconstruction.NewMutablePoolStateFromSnapshot(
			pool,
			checkpoint,
		)
	if err != nil {
		t.Fatalf(
			"NewMutablePoolStateFromSnapshot() error = %v",
			err,
		)
	}

	burn := newBurnPoolEvent(
		t,
		pool,
		reconstructionCursor(103, 0, 0),
		ownerB,
		-200,
		-100,
		big.NewInt(200),
	)

	mustApplyEvent(
		t,
		directState,
		burn,
	)

	mustApplyEvent(
		t,
		checkpointState,
		burn,
	)

	directFinal, err :=
		directState.SnapshotAfterLastEvent()
	if err != nil {
		t.Fatalf(
			"direct SnapshotAfterLastEvent() error = %v",
			err,
		)
	}

	checkpointFinal, err :=
		checkpointState.SnapshotAfterLastEvent()
	if err != nil {
		t.Fatalf(
			"checkpoint SnapshotAfterLastEvent() error = %v",
			err,
		)
	}

	assertSnapshotStateEqual(
		t,
		checkpointFinal,
		directFinal,
	)
}

func TestBeforeEventSnapshotCanReplayExactReferencedEvent(
	t *testing.T,
) {
	t.Parallel()

	pool := newReconstructionTestPool(t)
	state := newMutableState(t, pool)

	initializeStateAtTick(
		t,
		state,
		pool,
		reconstructionCursor(100, 0, 0),
		0,
	)

	owner := reconstructionOwner(t, 1)

	mustApplyEvent(
		t,
		state,
		newMintPoolEvent(
			t,
			pool,
			reconstructionCursor(101, 0, 0),
			owner,
			-100,
			100,
			big.NewInt(1_000),
		),
	)

	burnCursor :=
		reconstructionCursor(102, 0, 0)

	beforeBurn, err :=
		state.SnapshotBeforeEvent(
			burnCursor,
		)
	if err != nil {
		t.Fatalf(
			"SnapshotBeforeEvent() error = %v",
			err,
		)
	}

	if beforeBurn.Reference().Boundary() !=
		domain.SnapshotBeforeEvent {
		t.Fatalf(
			"boundary = %s, want before_event",
			beforeBurn.Reference().Boundary(),
		)
	}

	referenceCursor, exists :=
		beforeBurn.Reference().Cursor()

	if !exists {
		t.Fatal(
			"before-event snapshot cursor missing",
		)
	}

	if referenceCursor.Compare(
		burnCursor,
	) != 0 {
		t.Fatalf(
			"reference cursor = %s, want %s",
			referenceCursor,
			burnCursor,
		)
	}

	replayState, err :=
		reconstruction.NewMutablePoolStateFromSnapshot(
			pool,
			beforeBurn,
		)
	if err != nil {
		t.Fatalf(
			"NewMutablePoolStateFromSnapshot() error = %v",
			err,
		)
	}

	burn := newBurnPoolEvent(
		t,
		pool,
		burnCursor,
		owner,
		-100,
		100,
		big.NewInt(250),
	)

	mustApplyEvent(
		t,
		replayState,
		burn,
	)

	finalSnapshot, err :=
		replayState.SnapshotAfterLastEvent()
	if err != nil {
		t.Fatalf(
			"SnapshotAfterLastEvent() error = %v",
			err,
		)
	}

	assertReconstructionBigIntEqual(
		t,
		finalSnapshot.ActiveLiquidity(),
		big.NewInt(750),
	)
}

func TestMutablePoolStateRejectsEventFromDifferentPool(
	t *testing.T,
) {
	t.Parallel()

	pool := newReconstructionTestPool(t)
	state := newMutableState(t, pool)

	initializeStateAtTick(
		t,
		state,
		pool,
		reconstructionCursor(100, 0, 0),
		0,
	)

	otherPool := pool
	otherPool.Address = mustReconstructionAddress(
		t,
		"0xbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
	)

	event := newMintPoolEvent(
		t,
		otherPool,
		reconstructionCursor(101, 0, 0),
		reconstructionOwner(t, 1),
		-100,
		100,
		big.NewInt(1_000),
	)

	err := state.Apply(event)

	if !errors.Is(
		err,
		reconstruction.ErrPoolMismatch,
	) {
		t.Fatalf(
			"Apply() error = %v, want ErrPoolMismatch",
			err,
		)
	}
}
