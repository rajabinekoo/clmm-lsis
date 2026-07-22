package uniswapv3

import (
	"fmt"
	"math/big"
	"sort"

	"github.com/rajabinekoo/clmm-lsis/internal/domain"
)

// Simulator executes deterministic exact-input swaps against immutable pool
// snapshots.
//
// The simulator itself stores only immutable pool metadata. Each simulation
// starts from values copied from the provided snapshot and never mutates that
// snapshot.
type Simulator struct {
	pool domain.Pool
}

func NewSimulator(
	pool domain.Pool,
) (Simulator, error) {
	if err := pool.Validate(); err != nil {
		return Simulator{}, fmt.Errorf(
			"create simulator: %w",
			err,
		)
	}

	if pool.FeePips >= FeeDenominatorPips {
		return Simulator{}, fmt.Errorf(
			"create simulator: fee pips %d must be smaller than %d",
			pool.FeePips,
			FeeDenominatorPips,
		)
	}

	return Simulator{
		pool: pool,
	}, nil
}

// SimulateExactInput executes a complete exact-input swap.
//
// The simulation stops when either:
//
//   - all gross input has been consumed; or
//   - the requested sqrt-price limit is reached.
//
// Reaching a limit with remaining input is not returned as an error. The
// remaining amount is preserved explicitly in ExactInputResult.
func (s Simulator) SimulateExactInput(
	snapshot domain.PoolSnapshot,
	request ExactInputRequest,
) (ExactInputResult, error) {
	if err := request.Validate(); err != nil {
		return ExactInputResult{}, fmt.Errorf(
			"simulate exact input: %w",
			err,
		)
	}

	if err := s.validateSnapshot(snapshot); err != nil {
		return ExactInputResult{}, fmt.Errorf(
			"simulate exact input: %w",
			err,
		)
	}

	sqrtPriceLimitX96, err := resolveSqrtPriceLimit(
		snapshot.SqrtPriceX96(),
		request,
	)
	if err != nil {
		return ExactInputResult{}, fmt.Errorf(
			"simulate exact input: %w",
			err,
		)
	}

	ticks := snapshot.Ticks()

	state := exactInputSimulationState{
		amountRemaining: request.AmountIn(),

		amountInNet: new(big.Int),
		amountOut:   new(big.Int),
		feeAmount:   new(big.Int),

		sqrtPriceX96: snapshot.SqrtPriceX96(),
		currentTick:  snapshot.CurrentTick(),
		liquidity:    snapshot.ActiveLiquidity(),

		crossedTicks: make([]int32, 0),
		trace:        make([]SwapTraceStep, 0),
	}

	maximumSteps := len(ticks) + 2

	for state.amountRemaining.Sign() > 0 &&
		state.sqrtPriceX96.Cmp(sqrtPriceLimitX96) != 0 {
		if len(state.trace) >= maximumSteps {
			return ExactInputResult{}, fmt.Errorf(
				"simulate exact input: exceeded maximum expected step count %d",
				maximumSteps,
			)
		}

		if err := s.executeNextStep(
			&state,
			ticks,
			sqrtPriceLimitX96,
			request.ZeroForOne(),
		); err != nil {
			return ExactInputResult{}, fmt.Errorf(
				"simulate exact input step %d: %w",
				len(state.trace),
				err,
			)
		}
	}

	amountInConsumed := new(big.Int).Sub(
		request.AmountIn(),
		state.amountRemaining,
	)

	expectedConsumed := new(big.Int).Add(
		state.amountInNet,
		state.feeAmount,
	)

	if amountInConsumed.Cmp(expectedConsumed) != 0 {
		return ExactInputResult{}, fmt.Errorf(
			"simulate exact input accounting mismatch: gross consumed=%s net input plus fee=%s",
			amountInConsumed,
			expectedConsumed,
		)
	}

	hitPriceLimit := state.sqrtPriceX96.Cmp(
		sqrtPriceLimitX96,
	) == 0

	return newExactInputResult(
		request.AmountIn(),
		amountInConsumed,
		state.amountRemaining,
		state.amountInNet,
		state.amountOut,
		state.feeAmount,
		snapshot.SqrtPriceX96(),
		state.sqrtPriceX96,
		snapshot.CurrentTick(),
		state.currentTick,
		snapshot.ActiveLiquidity(),
		state.liquidity,
		state.crossedTicks,
		state.trace,
		hitPriceLimit,
	), nil
}

type exactInputSimulationState struct {
	amountRemaining *big.Int

	amountInNet *big.Int
	amountOut   *big.Int
	feeAmount   *big.Int

	sqrtPriceX96 *big.Int
	currentTick  int32
	liquidity    *big.Int

	crossedTicks []int32
	trace        []SwapTraceStep
}

func (s Simulator) executeNextStep(
	state *exactInputSimulationState,
	ticks []domain.TickState,
	sqrtPriceLimitX96 *big.Int,
	zeroForOne bool,
) error {
	sqrtPriceStartX96 := cloneInt(
		state.sqrtPriceX96,
	)

	liquidityStart := cloneInt(
		state.liquidity,
	)

	nextTick, initialized := findNextInitializedTick(
		ticks,
		state.currentTick,
		zeroForOne,
	)

	nextTickIndex := boundaryTickForDirection(
		zeroForOne,
	)

	if initialized {
		nextTickIndex = nextTick.Index()
	}

	sqrtPriceNextTickX96, err := GetSqrtRatioAtTick(
		nextTickIndex,
	)
	if err != nil {
		return fmt.Errorf(
			"compute sqrt price for next tick %d: %w",
			nextTickIndex,
			err,
		)
	}

	sqrtPriceTargetX96 := selectStepTarget(
		sqrtPriceNextTickX96,
		sqrtPriceLimitX96,
		zeroForOne,
	)

	var stepResult SwapStepResult

	if state.liquidity.Sign() == 0 {
		// Moving through an empty range requires no token amount. The price
		// advances directly to the next initialized boundary or price limit.
		stepResult = newSwapStepResult(
			sqrtPriceTargetX96,
			new(big.Int),
			new(big.Int),
			new(big.Int),
			true,
		)
	} else {
		stepResult, err = ComputeSwapStepExactInput(
			state.sqrtPriceX96,
			sqrtPriceTargetX96,
			state.liquidity,
			state.amountRemaining,
			s.pool.FeePips,
			zeroForOne,
		)
		if err != nil {
			return err
		}
	}

	totalInputConsumed := stepResult.TotalInputConsumed()

	if totalInputConsumed.Cmp(state.amountRemaining) > 0 {
		return fmt.Errorf(
			"step consumed %s but only %s input remained",
			totalInputConsumed,
			state.amountRemaining,
		)
	}

	state.amountRemaining.Sub(
		state.amountRemaining,
		totalInputConsumed,
	)

	state.amountInNet.Add(
		state.amountInNet,
		stepResult.AmountIn(),
	)

	state.amountOut.Add(
		state.amountOut,
		stepResult.AmountOut(),
	)

	state.feeAmount.Add(
		state.feeAmount,
		stepResult.FeeAmount(),
	)

	state.sqrtPriceX96 = stepResult.SqrtPriceNextX96()

	var crossedTick *int32

	reachedNextTickBoundary :=
		state.sqrtPriceX96.Cmp(
			sqrtPriceNextTickX96,
		) == 0

	if reachedNextTickBoundary {
		if initialized {
			nextLiquidity, err := ApplyLiquidityNet(
				state.liquidity,
				nextTick.LiquidityNet(),
				zeroForOne,
			)
			if err != nil {
				return fmt.Errorf(
					"cross initialized tick %d: %w",
					nextTickIndex,
					err,
				)
			}

			state.liquidity = nextLiquidity

			crossedValue := nextTickIndex
			crossedTick = &crossedValue

			state.crossedTicks = append(
				state.crossedTicks,
				nextTickIndex,
			)
		}

		if zeroForOne {
			state.currentTick = nextTickIndex - 1
		} else {
			state.currentTick = nextTickIndex
		}
	} else if state.sqrtPriceX96.Cmp(
		sqrtPriceStartX96,
	) != 0 {
		state.currentTick, err = GetTickAtSqrtRatio(
			state.sqrtPriceX96,
		)
		if err != nil {
			return fmt.Errorf(
				"derive current tick from ending sqrt price: %w",
				err,
			)
		}
	}

	state.trace = append(
		state.trace,
		newSwapTraceStep(
			len(state.trace),
			sqrtPriceStartX96,
			sqrtPriceTargetX96,
			state.sqrtPriceX96,
			liquidityStart,
			state.liquidity,
			stepResult.AmountIn(),
			stepResult.AmountOut(),
			stepResult.FeeAmount(),
			crossedTick,
		),
	)

	if totalInputConsumed.Sign() == 0 &&
		state.sqrtPriceX96.Cmp(sqrtPriceStartX96) == 0 &&
		crossedTick == nil {
		return fmt.Errorf(
			"swap step made no progress",
		)
	}

	return nil
}

func (s Simulator) validateSnapshot(
	snapshot domain.PoolSnapshot,
) error {
	if err := snapshot.Validate(); err != nil {
		return fmt.Errorf(
			"invalid pool snapshot: %w",
			err,
		)
	}

	if snapshot.PoolAddress() != s.pool.Address {
		return fmt.Errorf(
			"snapshot pool %s does not match simulator pool %s",
			snapshot.PoolAddress(),
			s.pool.Address,
		)
	}

	if snapshot.CurrentTick() < MinTick ||
		snapshot.CurrentTick() > MaxTick {
		return fmt.Errorf(
			"snapshot current tick %d is outside supported range [%d,%d]",
			snapshot.CurrentTick(),
			MinTick,
			MaxTick,
		)
	}

	currentSqrtPrice := snapshot.SqrtPriceX96()

	if currentSqrtPrice.Cmp(minSqrtRatio) < 0 ||
		currentSqrtPrice.Cmp(maxSqrtRatio) >= 0 {
		return fmt.Errorf(
			"snapshot sqrt price %s is outside protocol range [%s,%s)",
			currentSqrtPrice,
			minSqrtRatio,
			maxSqrtRatio,
		)
	}

	derivedTick, err := GetTickAtSqrtRatio(
		currentSqrtPrice,
	)
	if err != nil {
		return fmt.Errorf(
			"derive snapshot tick from sqrt price: %w",
			err,
		)
	}

	tickIsConsistent :=
		snapshot.CurrentTick() == derivedTick

	if !tickIsConsistent &&
		snapshot.CurrentTick() == derivedTick-1 {
		boundaryPrice, err :=
			GetSqrtRatioAtTick(derivedTick)
		if err != nil {
			return fmt.Errorf(
				"compute derived tick boundary price: %w",
				err,
			)
		}

		tickIsConsistent =
			boundaryPrice.Cmp(currentSqrtPrice) == 0
	}

	if !tickIsConsistent {
		return fmt.Errorf(
			"snapshot tick %d is inconsistent with sqrt-price-derived tick %d",
			snapshot.CurrentTick(),
			derivedTick,
		)
	}

	for _, tick := range snapshot.Ticks() {
		if tick.Index()%s.pool.TickSpacing != 0 {
			return fmt.Errorf(
				"initialized tick %d is not aligned with pool tick spacing %d",
				tick.Index(),
				s.pool.TickSpacing,
			)
		}
	}

	return nil
}

func resolveSqrtPriceLimit(
	currentSqrtPriceX96 *big.Int,
	request ExactInputRequest,
) (*big.Int, error) {
	limit, provided := request.SqrtPriceLimitX96()

	if !provided {
		limit = DefaultSqrtPriceLimit(
			request.ZeroForOne(),
		)
	}

	if limit.Cmp(minSqrtRatio) <= 0 ||
		limit.Cmp(maxSqrtRatio) >= 0 {
		return nil, fmt.Errorf(
			"sqrt price limit %s must be strictly inside protocol bounds (%s,%s)",
			limit,
			minSqrtRatio,
			maxSqrtRatio,
		)
	}

	if request.ZeroForOne() {
		if limit.Cmp(currentSqrtPriceX96) >= 0 {
			return nil, fmt.Errorf(
				"zero-for-one sqrt price limit %s must be smaller than current price %s",
				limit,
				currentSqrtPriceX96,
			)
		}

		return limit, nil
	}

	if limit.Cmp(currentSqrtPriceX96) <= 0 {
		return nil, fmt.Errorf(
			"one-for-zero sqrt price limit %s must be greater than current price %s",
			limit,
			currentSqrtPriceX96,
		)
	}

	return limit, nil
}

func findNextInitializedTick(
	ticks []domain.TickState,
	currentTick int32,
	zeroForOne bool,
) (
	domain.TickState,
	bool,
) {
	if zeroForOne {
		// Find the first tick strictly greater than currentTick, then select
		// its predecessor. The result is the greatest initialized tick less
		// than or equal to currentTick.
		index := sort.Search(
			len(ticks),
			func(index int) bool {
				return ticks[index].Index() > currentTick
			},
		)

		if index == 0 {
			return domain.TickState{}, false
		}

		return ticks[index-1], true
	}

	// Moving right searches for the smallest initialized tick strictly greater
	// than currentTick.
	index := sort.Search(
		len(ticks),
		func(index int) bool {
			return ticks[index].Index() > currentTick
		},
	)

	if index >= len(ticks) {
		return domain.TickState{}, false
	}

	return ticks[index], true
}

func boundaryTickForDirection(
	zeroForOne bool,
) int32 {
	if zeroForOne {
		return MinTick
	}

	return MaxTick
}

func selectStepTarget(
	sqrtPriceNextTickX96 *big.Int,
	sqrtPriceLimitX96 *big.Int,
	zeroForOne bool,
) *big.Int {
	if zeroForOne {
		// Price decreases. Stop at whichever boundary has the greater sqrt
		// price because it is encountered first.
		if sqrtPriceNextTickX96.Cmp(
			sqrtPriceLimitX96,
		) < 0 {
			return cloneInt(sqrtPriceLimitX96)
		}

		return cloneInt(sqrtPriceNextTickX96)
	}

	// Price increases. Stop at whichever boundary has the smaller sqrt price.
	if sqrtPriceNextTickX96.Cmp(
		sqrtPriceLimitX96,
	) > 0 {
		return cloneInt(sqrtPriceLimitX96)
	}

	return cloneInt(sqrtPriceNextTickX96)
}
