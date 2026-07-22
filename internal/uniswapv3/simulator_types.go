package uniswapv3

import (
	"fmt"
	"math/big"
)

// ExactInputRequest defines one exact-input swap simulation.
//
// AmountIn is the gross amount supplied by the trader and includes swap fees.
//
// SqrtPriceLimitX96 is optional. A nil value selects the widest valid protocol
// limit for the requested direction.
type ExactInputRequest struct {
	amountIn          *big.Int
	zeroForOne        bool
	sqrtPriceLimitX96 *big.Int
}

func NewExactInputRequest(
	amountIn *big.Int,
	zeroForOne bool,
	sqrtPriceLimitX96 *big.Int,
) (ExactInputRequest, error) {
	request := ExactInputRequest{
		amountIn:          cloneInt(amountIn),
		zeroForOne:        zeroForOne,
		sqrtPriceLimitX96: cloneInt(sqrtPriceLimitX96),
	}

	if err := request.Validate(); err != nil {
		return ExactInputRequest{}, err
	}

	return request, nil
}

func (r ExactInputRequest) Validate() error {
	if err := validateUnsigned(
		"exact-input amount",
		r.amountIn,
		256,
	); err != nil {
		return err
	}

	if r.amountIn.Sign() == 0 {
		return fmt.Errorf(
			"exact-input amount must be greater than zero",
		)
	}

	if r.sqrtPriceLimitX96 != nil {
		if err := validateSqrtPriceX96(
			"sqrt price limit X96",
			r.sqrtPriceLimitX96,
		); err != nil {
			return err
		}
	}

	return nil
}

func (r ExactInputRequest) AmountIn() *big.Int {
	return cloneInt(r.amountIn)
}

func (r ExactInputRequest) ZeroForOne() bool {
	return r.zeroForOne
}

func (r ExactInputRequest) SqrtPriceLimitX96() (
	*big.Int,
	bool,
) {
	if r.sqrtPriceLimitX96 == nil {
		return nil, false
	}

	return cloneInt(r.sqrtPriceLimitX96), true
}

// DefaultSqrtPriceLimit returns the widest valid protocol limit for one swap
// direction.
//
// Uniswap v3 requires the limit to remain strictly inside the absolute sqrt
// price bounds.
func DefaultSqrtPriceLimit(
	zeroForOne bool,
) *big.Int {
	if zeroForOne {
		return new(big.Int).Add(
			minSqrtRatio,
			big.NewInt(1),
		)
	}

	return new(big.Int).Sub(
		maxSqrtRatio,
		big.NewInt(1),
	)
}

// SwapTraceStep describes one price movement between the current sqrt price
// and one target.
//
// A step may:
//
//   - reach and cross an initialized tick;
//   - move partially toward the target;
//   - move through an empty-liquidity region without consuming input;
//   - consume a tiny input entirely as fee without moving the price.
type SwapTraceStep struct {
	index int

	sqrtPriceStartX96  *big.Int
	sqrtPriceTargetX96 *big.Int
	sqrtPriceEndX96    *big.Int

	liquidityStart *big.Int
	liquidityEnd   *big.Int

	amountIn  *big.Int
	amountOut *big.Int
	feeAmount *big.Int

	crossedTick *int32
}

func newSwapTraceStep(
	index int,
	sqrtPriceStartX96 *big.Int,
	sqrtPriceTargetX96 *big.Int,
	sqrtPriceEndX96 *big.Int,
	liquidityStart *big.Int,
	liquidityEnd *big.Int,
	amountIn *big.Int,
	amountOut *big.Int,
	feeAmount *big.Int,
	crossedTick *int32,
) SwapTraceStep {
	var copiedCrossedTick *int32

	if crossedTick != nil {
		value := *crossedTick
		copiedCrossedTick = &value
	}

	return SwapTraceStep{
		index: index,

		sqrtPriceStartX96:  cloneInt(sqrtPriceStartX96),
		sqrtPriceTargetX96: cloneInt(sqrtPriceTargetX96),
		sqrtPriceEndX96:    cloneInt(sqrtPriceEndX96),

		liquidityStart: cloneInt(liquidityStart),
		liquidityEnd:   cloneInt(liquidityEnd),

		amountIn:  cloneInt(amountIn),
		amountOut: cloneInt(amountOut),
		feeAmount: cloneInt(feeAmount),

		crossedTick: copiedCrossedTick,
	}
}

func (s SwapTraceStep) Index() int {
	return s.index
}

func (s SwapTraceStep) SqrtPriceStartX96() *big.Int {
	return cloneInt(s.sqrtPriceStartX96)
}

func (s SwapTraceStep) SqrtPriceTargetX96() *big.Int {
	return cloneInt(s.sqrtPriceTargetX96)
}

func (s SwapTraceStep) SqrtPriceEndX96() *big.Int {
	return cloneInt(s.sqrtPriceEndX96)
}

func (s SwapTraceStep) LiquidityStart() *big.Int {
	return cloneInt(s.liquidityStart)
}

func (s SwapTraceStep) LiquidityEnd() *big.Int {
	return cloneInt(s.liquidityEnd)
}

// AmountIn returns the net input consumed by the price movement, excluding the
// fee charged in this step.
func (s SwapTraceStep) AmountIn() *big.Int {
	return cloneInt(s.amountIn)
}

func (s SwapTraceStep) AmountOut() *big.Int {
	return cloneInt(s.amountOut)
}

func (s SwapTraceStep) FeeAmount() *big.Int {
	return cloneInt(s.feeAmount)
}

func (s SwapTraceStep) TotalInputConsumed() *big.Int {
	return new(big.Int).Add(
		s.amountIn,
		s.feeAmount,
	)
}

func (s SwapTraceStep) CrossedTick() (
	int32,
	bool,
) {
	if s.crossedTick == nil {
		return 0, false
	}

	return *s.crossedTick, true
}

// ExactInputResult contains the complete result of one simulation.
//
// AmountInConsumed is gross trader input and therefore equals:
//
//	AmountInNet + FeeAmount
//
// When FullyExecuted is false, the result reached its price limit or ran out
// of usable liquidity before consuming the complete requested amount.
type ExactInputResult struct {
	amountInSpecified *big.Int
	amountInConsumed  *big.Int
	amountInRemaining *big.Int
	amountInNet       *big.Int

	amountOut *big.Int
	feeAmount *big.Int

	sqrtPriceStartX96 *big.Int
	sqrtPriceEndX96   *big.Int

	tickStart int32
	tickEnd   int32

	liquidityStart *big.Int
	liquidityEnd   *big.Int

	crossedTicks []int32
	trace        []SwapTraceStep

	hitPriceLimit bool
}

func newExactInputResult(
	amountInSpecified *big.Int,
	amountInConsumed *big.Int,
	amountInRemaining *big.Int,
	amountInNet *big.Int,
	amountOut *big.Int,
	feeAmount *big.Int,
	sqrtPriceStartX96 *big.Int,
	sqrtPriceEndX96 *big.Int,
	tickStart int32,
	tickEnd int32,
	liquidityStart *big.Int,
	liquidityEnd *big.Int,
	crossedTicks []int32,
	trace []SwapTraceStep,
	hitPriceLimit bool,
) ExactInputResult {
	return ExactInputResult{
		amountInSpecified: cloneInt(amountInSpecified),
		amountInConsumed:  cloneInt(amountInConsumed),
		amountInRemaining: cloneInt(amountInRemaining),
		amountInNet:       cloneInt(amountInNet),

		amountOut: cloneInt(amountOut),
		feeAmount: cloneInt(feeAmount),

		sqrtPriceStartX96: cloneInt(sqrtPriceStartX96),
		sqrtPriceEndX96:   cloneInt(sqrtPriceEndX96),

		tickStart: tickStart,
		tickEnd:   tickEnd,

		liquidityStart: cloneInt(liquidityStart),
		liquidityEnd:   cloneInt(liquidityEnd),

		crossedTicks: append(
			[]int32(nil),
			crossedTicks...,
		),

		trace: append(
			[]SwapTraceStep(nil),
			trace...,
		),

		hitPriceLimit: hitPriceLimit,
	}
}

func (r ExactInputResult) AmountInSpecified() *big.Int {
	return cloneInt(r.amountInSpecified)
}

func (r ExactInputResult) AmountInConsumed() *big.Int {
	return cloneInt(r.amountInConsumed)
}

func (r ExactInputResult) AmountInRemaining() *big.Int {
	return cloneInt(r.amountInRemaining)
}

func (r ExactInputResult) AmountInNet() *big.Int {
	return cloneInt(r.amountInNet)
}

func (r ExactInputResult) AmountOut() *big.Int {
	return cloneInt(r.amountOut)
}

func (r ExactInputResult) FeeAmount() *big.Int {
	return cloneInt(r.feeAmount)
}

func (r ExactInputResult) SqrtPriceStartX96() *big.Int {
	return cloneInt(r.sqrtPriceStartX96)
}

func (r ExactInputResult) SqrtPriceEndX96() *big.Int {
	return cloneInt(r.sqrtPriceEndX96)
}

func (r ExactInputResult) TickStart() int32 {
	return r.tickStart
}

func (r ExactInputResult) TickEnd() int32 {
	return r.tickEnd
}

func (r ExactInputResult) LiquidityStart() *big.Int {
	return cloneInt(r.liquidityStart)
}

func (r ExactInputResult) LiquidityEnd() *big.Int {
	return cloneInt(r.liquidityEnd)
}

func (r ExactInputResult) CrossedTicks() []int32 {
	return append(
		[]int32(nil),
		r.crossedTicks...,
	)
}

func (r ExactInputResult) Trace() []SwapTraceStep {
	return append(
		[]SwapTraceStep(nil),
		r.trace...,
	)
}

func (r ExactInputResult) HitPriceLimit() bool {
	return r.hitPriceLimit
}

// FullyExecuted reports whether the simulator consumed all requested gross
// input.
func (r ExactInputResult) FullyExecuted() bool {
	return r.amountInRemaining.Sign() == 0
}
