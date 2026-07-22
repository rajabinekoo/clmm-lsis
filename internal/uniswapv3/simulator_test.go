package uniswapv3_test

import (
	"math/big"
	"testing"

	"github.com/rajabinekoo/clmm-lsis/internal/uniswapv3"
)

func TestSimulatorExactInputZeroForOneWithoutTickCrossing(
	t *testing.T,
) {
	t.Parallel()

	pool, simulator := newSimulatorForTest(t)

	currentSqrtPrice := mustSqrtPriceAtTick(
		t,
		0,
	)

	lowerBoundary := mustSqrtPriceAtTick(
		t,
		-100,
	)

	snapshot := buildSimulatorSnapshot(
		t,
		pool,
		1_000,
		0,
		currentSqrtPrice,
		[]simulatorPositionSpec{
			{
				ownerSuffix: 1,
				tickLower:   -100,
				tickUpper:   100,
				liquidity:   big.NewInt(1_000_000_000_000_000_000),
			},
		},
	)

	request, err := uniswapv3.NewExactInputRequest(
		big.NewInt(1_000_000_000_000),
		true,
		nil,
	)
	if err != nil {
		t.Fatalf(
			"NewExactInputRequest() error = %v",
			err,
		)
	}

	result, err := simulator.SimulateExactInput(
		snapshot,
		request,
	)
	if err != nil {
		t.Fatalf(
			"SimulateExactInput() error = %v",
			err,
		)
	}

	if !result.FullyExecuted() {
		t.Fatal(
			"FullyExecuted() = false, want true",
		)
	}

	if result.HitPriceLimit() {
		t.Fatal(
			"HitPriceLimit() = true, want false",
		)
	}

	if len(result.CrossedTicks()) != 0 {
		t.Fatalf(
			"CrossedTicks() = %v, want none",
			result.CrossedTicks(),
		)
	}

	if len(result.Trace()) != 1 {
		t.Fatalf(
			"len(Trace()) = %d, want 1",
			len(result.Trace()),
		)
	}

	if result.AmountOut().Sign() <= 0 {
		t.Fatalf(
			"AmountOut() = %s, want positive",
			result.AmountOut(),
		)
	}

	if result.FeeAmount().Sign() <= 0 {
		t.Fatalf(
			"FeeAmount() = %s, want positive",
			result.FeeAmount(),
		)
	}

	assertBigIntEqual(
		t,
		result.AmountInConsumed(),
		request.AmountIn(),
	)

	expectedGrossConsumption := new(big.Int).Add(
		result.AmountInNet(),
		result.FeeAmount(),
	)

	assertBigIntEqual(
		t,
		result.AmountInConsumed(),
		expectedGrossConsumption,
	)

	if result.SqrtPriceEndX96().Cmp(
		currentSqrtPrice,
	) >= 0 {
		t.Fatalf(
			"zero-for-one ending sqrt price %s must be below current price %s",
			result.SqrtPriceEndX96(),
			currentSqrtPrice,
		)
	}

	if result.SqrtPriceEndX96().Cmp(
		lowerBoundary,
	) <= 0 {
		t.Fatalf(
			"ending sqrt price %s crossed lower boundary %s",
			result.SqrtPriceEndX96(),
			lowerBoundary,
		)
	}

	if result.TickEnd() >= 0 ||
		result.TickEnd() <= -100 {
		t.Fatalf(
			"TickEnd() = %d, want between -100 and 0",
			result.TickEnd(),
		)
	}

	assertBigIntEqual(
		t,
		result.LiquidityEnd(),
		snapshot.ActiveLiquidity(),
	)
}

func TestSimulatorZeroForOneCrossesOneTick(
	t *testing.T,
) {
	t.Parallel()

	pool, simulator := newSimulatorForTest(t)

	currentSqrtPrice := mustSqrtPriceAtTick(
		t,
		0,
	)

	targetSqrtPrice := mustSqrtPriceAtTick(
		t,
		-100,
	)

	currentLiquidity :=
		big.NewInt(1_000_000_000_000_000_000)

	nextLiquidity :=
		big.NewInt(500_000_000_000_000_000)

	snapshot := buildSimulatorSnapshot(
		t,
		pool,
		1_001,
		0,
		currentSqrtPrice,
		[]simulatorPositionSpec{
			{
				ownerSuffix: 1,
				tickLower:   -100,
				tickUpper:   100,
				liquidity:   currentLiquidity,
			},
			{
				ownerSuffix: 2,
				tickLower:   -200,
				tickUpper:   -100,
				liquidity:   nextLiquidity,
			},
		},
	)

	grossInput := grossInputRequiredToReachTarget(
		t,
		currentSqrtPrice,
		targetSqrtPrice,
		currentLiquidity,
		pool.FeePips,
		true,
	)

	request, err := uniswapv3.NewExactInputRequest(
		grossInput,
		true,
		nil,
	)
	if err != nil {
		t.Fatalf(
			"NewExactInputRequest() error = %v",
			err,
		)
	}

	result, err := simulator.SimulateExactInput(
		snapshot,
		request,
	)
	if err != nil {
		t.Fatalf(
			"SimulateExactInput() error = %v",
			err,
		)
	}

	if !result.FullyExecuted() {
		t.Fatal(
			"FullyExecuted() = false, want true",
		)
	}

	assertInt32SliceEqual(
		t,
		result.CrossedTicks(),
		[]int32{-100},
	)

	assertBigIntEqual(
		t,
		result.SqrtPriceEndX96(),
		targetSqrtPrice,
	)

	assertBigIntEqual(
		t,
		result.LiquidityEnd(),
		nextLiquidity,
	)

	if result.TickEnd() != -101 {
		t.Fatalf(
			"TickEnd() = %d, want -101",
			result.TickEnd(),
		)
	}

	if len(result.Trace()) != 1 {
		t.Fatalf(
			"len(Trace()) = %d, want 1",
			len(result.Trace()),
		)
	}

	crossedTick, crossed :=
		result.Trace()[0].CrossedTick()

	if !crossed {
		t.Fatal(
			"Trace()[0].CrossedTick() expected crossing",
		)
	}

	if crossedTick != -100 {
		t.Fatalf(
			"crossed tick = %d, want -100",
			crossedTick,
		)
	}
}

func TestSimulatorOneForZeroCrossesOneTick(
	t *testing.T,
) {
	t.Parallel()

	pool, simulator := newSimulatorForTest(t)

	currentSqrtPrice := mustSqrtPriceAtTick(
		t,
		0,
	)

	targetSqrtPrice := mustSqrtPriceAtTick(
		t,
		100,
	)

	currentLiquidity :=
		big.NewInt(1_000_000_000_000_000_000)

	nextLiquidity :=
		big.NewInt(700_000_000_000_000_000)

	snapshot := buildSimulatorSnapshot(
		t,
		pool,
		1_002,
		0,
		currentSqrtPrice,
		[]simulatorPositionSpec{
			{
				ownerSuffix: 1,
				tickLower:   -100,
				tickUpper:   100,
				liquidity:   currentLiquidity,
			},
			{
				ownerSuffix: 2,
				tickLower:   100,
				tickUpper:   200,
				liquidity:   nextLiquidity,
			},
		},
	)

	grossInput := grossInputRequiredToReachTarget(
		t,
		currentSqrtPrice,
		targetSqrtPrice,
		currentLiquidity,
		pool.FeePips,
		false,
	)

	request, err := uniswapv3.NewExactInputRequest(
		grossInput,
		false,
		nil,
	)
	if err != nil {
		t.Fatalf(
			"NewExactInputRequest() error = %v",
			err,
		)
	}

	result, err := simulator.SimulateExactInput(
		snapshot,
		request,
	)
	if err != nil {
		t.Fatalf(
			"SimulateExactInput() error = %v",
			err,
		)
	}

	if !result.FullyExecuted() {
		t.Fatal(
			"FullyExecuted() = false, want true",
		)
	}

	assertInt32SliceEqual(
		t,
		result.CrossedTicks(),
		[]int32{100},
	)

	assertBigIntEqual(
		t,
		result.SqrtPriceEndX96(),
		targetSqrtPrice,
	)

	assertBigIntEqual(
		t,
		result.LiquidityEnd(),
		nextLiquidity,
	)

	if result.TickEnd() != 100 {
		t.Fatalf(
			"TickEnd() = %d, want 100",
			result.TickEnd(),
		)
	}
}

func TestSimulatorCrossesMultipleTicks(
	t *testing.T,
) {
	t.Parallel()

	pool, simulator := newSimulatorForTest(t)

	sqrtAtZero := mustSqrtPriceAtTick(
		t,
		0,
	)

	sqrtAtMinus100 := mustSqrtPriceAtTick(
		t,
		-100,
	)

	sqrtAtMinus200 := mustSqrtPriceAtTick(
		t,
		-200,
	)

	liquidityA :=
		big.NewInt(1_000_000_000_000_000_000)

	liquidityB :=
		big.NewInt(500_000_000_000_000_000)

	liquidityC :=
		big.NewInt(300_000_000_000_000_000)

	snapshot := buildSimulatorSnapshot(
		t,
		pool,
		1_003,
		0,
		sqrtAtZero,
		[]simulatorPositionSpec{
			{
				ownerSuffix: 1,
				tickLower:   -100,
				tickUpper:   100,
				liquidity:   liquidityA,
			},
			{
				ownerSuffix: 2,
				tickLower:   -200,
				tickUpper:   -100,
				liquidity:   liquidityB,
			},
			{
				ownerSuffix: 3,
				tickLower:   -300,
				tickUpper:   -200,
				liquidity:   liquidityC,
			},
		},
	)

	firstStepGross :=
		grossInputRequiredToReachTarget(
			t,
			sqrtAtZero,
			sqrtAtMinus100,
			liquidityA,
			pool.FeePips,
			true,
		)

	secondStepGross :=
		grossInputRequiredToReachTarget(
			t,
			sqrtAtMinus100,
			sqrtAtMinus200,
			liquidityB,
			pool.FeePips,
			true,
		)

	// One additional raw unit is consumed completely as fee. It creates one
	// final trace step without moving the price or crossing another tick.
	amountIn := new(big.Int).Add(
		firstStepGross,
		secondStepGross,
	)

	amountIn.Add(
		amountIn,
		big.NewInt(1),
	)

	request, err := uniswapv3.NewExactInputRequest(
		amountIn,
		true,
		nil,
	)
	if err != nil {
		t.Fatalf(
			"NewExactInputRequest() error = %v",
			err,
		)
	}

	result, err := simulator.SimulateExactInput(
		snapshot,
		request,
	)
	if err != nil {
		t.Fatalf(
			"SimulateExactInput() error = %v",
			err,
		)
	}

	if !result.FullyExecuted() {
		t.Fatal(
			"FullyExecuted() = false, want true",
		)
	}

	assertInt32SliceEqual(
		t,
		result.CrossedTicks(),
		[]int32{-100, -200},
	)

	assertBigIntEqual(
		t,
		result.SqrtPriceEndX96(),
		sqrtAtMinus200,
	)

	assertBigIntEqual(
		t,
		result.LiquidityEnd(),
		liquidityC,
	)

	if result.TickEnd() != -201 {
		t.Fatalf(
			"TickEnd() = %d, want -201",
			result.TickEnd(),
		)
	}

	if len(result.Trace()) != 3 {
		t.Fatalf(
			"len(Trace()) = %d, want 3",
			len(result.Trace()),
		)
	}

	lastStep := result.Trace()[2]

	assertBigIntEqual(
		t,
		lastStep.AmountIn(),
		new(big.Int),
	)

	assertBigIntEqual(
		t,
		lastStep.AmountOut(),
		new(big.Int),
	)

	assertBigIntEqual(
		t,
		lastStep.FeeAmount(),
		big.NewInt(1),
	)
}

func TestSimulatorTraversesEmptyLiquidityGap(
	t *testing.T,
) {
	t.Parallel()

	pool, simulator := newSimulatorForTest(t)

	sqrtAtZero := mustSqrtPriceAtTick(
		t,
		0,
	)

	sqrtAtMinus100 := mustSqrtPriceAtTick(
		t,
		-100,
	)

	sqrtAtMinus200 := mustSqrtPriceAtTick(
		t,
		-200,
	)

	sqrtAtMinus300 := mustSqrtPriceAtTick(
		t,
		-300,
	)

	liquidityA :=
		big.NewInt(1_000_000_000_000_000_000)

	liquidityB :=
		big.NewInt(500_000_000_000_000_000)

	snapshot := buildSimulatorSnapshot(
		t,
		pool,
		1_004,
		0,
		sqrtAtZero,
		[]simulatorPositionSpec{
			{
				ownerSuffix: 1,
				tickLower:   -100,
				tickUpper:   100,
				liquidity:   liquidityA,
			},
			{
				ownerSuffix: 2,
				tickLower:   -300,
				tickUpper:   -200,
				liquidity:   liquidityB,
			},
		},
	)

	firstStepGross :=
		grossInputRequiredToReachTarget(
			t,
			sqrtAtZero,
			sqrtAtMinus100,
			liquidityA,
			pool.FeePips,
			true,
		)

	amountIn := new(big.Int).Add(
		firstStepGross,
		big.NewInt(1_000_000_000_000),
	)

	request, err := uniswapv3.NewExactInputRequest(
		amountIn,
		true,
		nil,
	)
	if err != nil {
		t.Fatalf(
			"NewExactInputRequest() error = %v",
			err,
		)
	}

	result, err := simulator.SimulateExactInput(
		snapshot,
		request,
	)
	if err != nil {
		t.Fatalf(
			"SimulateExactInput() error = %v",
			err,
		)
	}

	if !result.FullyExecuted() {
		t.Fatal(
			"FullyExecuted() = false, want true",
		)
	}

	assertInt32SliceEqual(
		t,
		result.CrossedTicks(),
		[]int32{-100, -200},
	)

	if len(result.Trace()) != 3 {
		t.Fatalf(
			"len(Trace()) = %d, want 3",
			len(result.Trace()),
		)
	}

	emptyGapStep := result.Trace()[1]

	assertBigIntEqual(
		t,
		emptyGapStep.SqrtPriceStartX96(),
		sqrtAtMinus100,
	)

	assertBigIntEqual(
		t,
		emptyGapStep.SqrtPriceEndX96(),
		sqrtAtMinus200,
	)

	assertBigIntEqual(
		t,
		emptyGapStep.LiquidityStart(),
		new(big.Int),
	)

	assertBigIntEqual(
		t,
		emptyGapStep.LiquidityEnd(),
		liquidityB,
	)

	assertBigIntEqual(
		t,
		emptyGapStep.TotalInputConsumed(),
		new(big.Int),
	)

	if result.SqrtPriceEndX96().Cmp(
		sqrtAtMinus200,
	) >= 0 {
		t.Fatalf(
			"ending sqrt price %s must be below gap boundary %s",
			result.SqrtPriceEndX96(),
			sqrtAtMinus200,
		)
	}

	if result.SqrtPriceEndX96().Cmp(
		sqrtAtMinus300,
	) <= 0 {
		t.Fatalf(
			"ending sqrt price %s crossed lower position boundary %s",
			result.SqrtPriceEndX96(),
			sqrtAtMinus300,
		)
	}
}

func TestSimulatorStopsAtExplicitPriceLimit(
	t *testing.T,
) {
	t.Parallel()

	pool, simulator := newSimulatorForTest(t)

	currentSqrtPrice := mustSqrtPriceAtTick(
		t,
		0,
	)

	limitSqrtPrice := mustSqrtPriceAtTick(
		t,
		-50,
	)

	liquidity :=
		big.NewInt(1_000_000_000_000_000_000)

	snapshot := buildSimulatorSnapshot(
		t,
		pool,
		1_005,
		0,
		currentSqrtPrice,
		[]simulatorPositionSpec{
			{
				ownerSuffix: 1,
				tickLower:   -100,
				tickUpper:   100,
				liquidity:   liquidity,
			},
		},
	)

	requiredGrossInput :=
		grossInputRequiredToReachTarget(
			t,
			currentSqrtPrice,
			limitSqrtPrice,
			liquidity,
			pool.FeePips,
			true,
		)

	excessInput := big.NewInt(
		1_000_000_000_000,
	)

	amountIn := new(big.Int).Add(
		requiredGrossInput,
		excessInput,
	)

	request, err := uniswapv3.NewExactInputRequest(
		amountIn,
		true,
		limitSqrtPrice,
	)
	if err != nil {
		t.Fatalf(
			"NewExactInputRequest() error = %v",
			err,
		)
	}

	result, err := simulator.SimulateExactInput(
		snapshot,
		request,
	)
	if err != nil {
		t.Fatalf(
			"SimulateExactInput() error = %v",
			err,
		)
	}

	if result.FullyExecuted() {
		t.Fatal(
			"FullyExecuted() = true, want false",
		)
	}

	if !result.HitPriceLimit() {
		t.Fatal(
			"HitPriceLimit() = false, want true",
		)
	}

	assertBigIntEqual(
		t,
		result.SqrtPriceEndX96(),
		limitSqrtPrice,
	)

	assertBigIntEqual(
		t,
		result.AmountInConsumed(),
		requiredGrossInput,
	)

	assertBigIntEqual(
		t,
		result.AmountInRemaining(),
		excessInput,
	)

	if len(result.CrossedTicks()) != 0 {
		t.Fatalf(
			"CrossedTicks() = %v, want none",
			result.CrossedTicks(),
		)
	}

	if result.TickEnd() != -50 {
		t.Fatalf(
			"TickEnd() = %d, want -50",
			result.TickEnd(),
		)
	}
}

func TestSimulatorRejectsInvalidPriceLimitDirection(
	t *testing.T,
) {
	t.Parallel()

	pool, simulator := newSimulatorForTest(t)

	currentSqrtPrice := mustSqrtPriceAtTick(
		t,
		0,
	)

	higherPriceLimit := mustSqrtPriceAtTick(
		t,
		50,
	)

	snapshot := buildSimulatorSnapshot(
		t,
		pool,
		1_006,
		0,
		currentSqrtPrice,
		[]simulatorPositionSpec{
			{
				ownerSuffix: 1,
				tickLower:   -100,
				tickUpper:   100,
				liquidity:   big.NewInt(1_000_000_000_000_000_000),
			},
		},
	)

	request, err := uniswapv3.NewExactInputRequest(
		big.NewInt(1_000_000),
		true,
		higherPriceLimit,
	)
	if err != nil {
		t.Fatalf(
			"NewExactInputRequest() error = %v",
			err,
		)
	}

	_, err = simulator.SimulateExactInput(
		snapshot,
		request,
	)
	if err == nil {
		t.Fatal(
			"SimulateExactInput() expected invalid price-limit error",
		)
	}
}

func TestSimulatorDoesNotMutateSnapshotOrRequest(
	t *testing.T,
) {
	t.Parallel()

	pool, simulator := newSimulatorForTest(t)

	currentSqrtPrice := mustSqrtPriceAtTick(
		t,
		0,
	)

	snapshot := buildSimulatorSnapshot(
		t,
		pool,
		1_007,
		0,
		currentSqrtPrice,
		[]simulatorPositionSpec{
			{
				ownerSuffix: 1,
				tickLower:   -100,
				tickUpper:   100,
				liquidity:   big.NewInt(1_000_000_000_000_000_000),
			},
		},
	)

	originalSnapshotSqrt :=
		snapshot.SqrtPriceX96()

	originalSnapshotLiquidity :=
		snapshot.ActiveLiquidity()

	originalRequestAmount :=
		big.NewInt(1_000_000_000_000)

	request, err := uniswapv3.NewExactInputRequest(
		originalRequestAmount,
		true,
		nil,
	)
	if err != nil {
		t.Fatalf(
			"NewExactInputRequest() error = %v",
			err,
		)
	}

	result, err := simulator.SimulateExactInput(
		snapshot,
		request,
	)
	if err != nil {
		t.Fatalf(
			"SimulateExactInput() error = %v",
			err,
		)
	}

	assertBigIntEqual(
		t,
		snapshot.SqrtPriceX96(),
		originalSnapshotSqrt,
	)

	assertBigIntEqual(
		t,
		snapshot.ActiveLiquidity(),
		originalSnapshotLiquidity,
	)

	assertBigIntEqual(
		t,
		request.AmountIn(),
		originalRequestAmount,
	)

	exposedResultAmount := result.AmountOut()
	exposedResultAmount.SetInt64(0)

	if result.AmountOut().Sign() <= 0 {
		t.Fatal(
			"mutating returned AmountOut changed result state",
		)
	}

	exposedCrossedTicks := result.CrossedTicks()

	if len(exposedCrossedTicks) > 0 {
		exposedCrossedTicks[0] = 123456
	}

	if len(result.CrossedTicks()) > 0 &&
		result.CrossedTicks()[0] == 123456 {
		t.Fatal(
			"mutating returned crossed-tick slice changed result state",
		)
	}
}

func TestSimulatorRejectsSnapshotFromDifferentPool(
	t *testing.T,
) {
	t.Parallel()

	pool, simulator := newSimulatorForTest(t)

	otherPool := pool
	otherPool.Address = mustSimulatorAddress(
		t,
		"0xbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
	)

	snapshot := buildSimulatorSnapshot(
		t,
		otherPool,
		1_008,
		0,
		mustSqrtPriceAtTick(t, 0),
		[]simulatorPositionSpec{
			{
				ownerSuffix: 1,
				tickLower:   -100,
				tickUpper:   100,
				liquidity:   big.NewInt(1_000_000_000_000_000_000),
			},
		},
	)

	request, err := uniswapv3.NewExactInputRequest(
		big.NewInt(1_000_000),
		true,
		nil,
	)
	if err != nil {
		t.Fatalf(
			"NewExactInputRequest() error = %v",
			err,
		)
	}

	_, err = simulator.SimulateExactInput(
		snapshot,
		request,
	)
	if err == nil {
		t.Fatal(
			"SimulateExactInput() expected pool mismatch error",
		)
	}
}
