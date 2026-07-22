package uniswapv3

import (
	"fmt"
	"math/big"
)

// SwapStepResult contains the result of one exact-input swap step.
//
// One step either:
//
//   - reaches the next initialized tick price; or
//   - consumes all remaining input before reaching that target.
//
// Values are kept private so mutable big.Int pointers cannot escape.
type SwapStepResult struct {
	sqrtPriceNextX96 *big.Int
	amountIn         *big.Int
	amountOut        *big.Int
	feeAmount        *big.Int
	reachedTarget    bool
}

func newSwapStepResult(
	sqrtPriceNextX96 *big.Int,
	amountIn *big.Int,
	amountOut *big.Int,
	feeAmount *big.Int,
	reachedTarget bool,
) SwapStepResult {
	return SwapStepResult{
		sqrtPriceNextX96: cloneInt(sqrtPriceNextX96),
		amountIn:         cloneInt(amountIn),
		amountOut:        cloneInt(amountOut),
		feeAmount:        cloneInt(feeAmount),
		reachedTarget:    reachedTarget,
	}
}

func (r SwapStepResult) SqrtPriceNextX96() *big.Int {
	return cloneInt(r.sqrtPriceNextX96)
}

func (r SwapStepResult) AmountIn() *big.Int {
	return cloneInt(r.amountIn)
}

func (r SwapStepResult) AmountOut() *big.Int {
	return cloneInt(r.amountOut)
}

func (r SwapStepResult) FeeAmount() *big.Int {
	return cloneInt(r.feeAmount)
}

func (r SwapStepResult) ReachedTarget() bool {
	return r.reachedTarget
}

// TotalInputConsumed returns amountIn + feeAmount.
func (r SwapStepResult) TotalInputConsumed() *big.Int {
	return new(big.Int).Add(
		r.amountIn,
		r.feeAmount,
	)
}

// ComputeSwapStepExactInput executes one Uniswap v3 exact-input swap step.
//
// amountRemaining is the gross input still available, including the fee.
// feePips is expressed over FeeDenominatorPips.
//
// The target must be:
//
//   - less than or equal to the current price for zeroForOne; or
//   - greater than or equal to the current price for oneForZero.
func ComputeSwapStepExactInput(
	sqrtPriceCurrentX96 *big.Int,
	sqrtPriceTargetX96 *big.Int,
	liquidity *big.Int,
	amountRemaining *big.Int,
	feePips uint32,
	zeroForOne bool,
) (SwapStepResult, error) {
	if err := validateSqrtPriceX96(
		"current sqrt price X96",
		sqrtPriceCurrentX96,
	); err != nil {
		return SwapStepResult{}, err
	}

	if err := validateSqrtPriceX96(
		"target sqrt price X96",
		sqrtPriceTargetX96,
	); err != nil {
		return SwapStepResult{}, err
	}

	if err := validateLiquidity(liquidity); err != nil {
		return SwapStepResult{}, err
	}

	if err := validateUnsigned(
		"remaining input amount",
		amountRemaining,
		256,
	); err != nil {
		return SwapStepResult{}, err
	}

	if feePips >= FeeDenominatorPips {
		return SwapStepResult{}, fmt.Errorf(
			"fee pips %d must be smaller than %d",
			feePips,
			FeeDenominatorPips,
		)
	}

	if zeroForOne &&
		sqrtPriceTargetX96.Cmp(sqrtPriceCurrentX96) > 0 {
		return SwapStepResult{}, fmt.Errorf(
			"zero-for-one target sqrt price must not exceed current sqrt price",
		)
	}

	if !zeroForOne &&
		sqrtPriceTargetX96.Cmp(sqrtPriceCurrentX96) < 0 {
		return SwapStepResult{}, fmt.Errorf(
			"one-for-zero target sqrt price must not be smaller than current sqrt price",
		)
	}

	if sqrtPriceCurrentX96.Cmp(sqrtPriceTargetX96) == 0 {
		return newSwapStepResult(
			sqrtPriceCurrentX96,
			new(big.Int),
			new(big.Int),
			new(big.Int),
			true,
		), nil
	}

	if amountRemaining.Sign() == 0 {
		return newSwapStepResult(
			sqrtPriceCurrentX96,
			new(big.Int),
			new(big.Int),
			new(big.Int),
			false,
		), nil
	}

	feeComplement := new(big.Int).SetUint64(
		uint64(FeeDenominatorPips - feePips),
	)

	amountRemainingLessFee, err := MulDiv(
		amountRemaining,
		feeComplement,
		new(big.Int).SetUint64(
			uint64(FeeDenominatorPips),
		),
	)
	if err != nil {
		return SwapStepResult{}, fmt.Errorf(
			"deduct swap-step fee: %w",
			err,
		)
	}

	amountInToTarget, err := amountInForPriceMovement(
		sqrtPriceCurrentX96,
		sqrtPriceTargetX96,
		liquidity,
		zeroForOne,
	)
	if err != nil {
		return SwapStepResult{}, err
	}

	var sqrtPriceNextX96 *big.Int

	if amountRemainingLessFee.Cmp(amountInToTarget) >= 0 {
		sqrtPriceNextX96 = cloneInt(sqrtPriceTargetX96)
	} else {
		sqrtPriceNextX96, err = GetNextSqrtPriceFromInput(
			sqrtPriceCurrentX96,
			liquidity,
			amountRemainingLessFee,
			zeroForOne,
		)
		if err != nil {
			return SwapStepResult{}, fmt.Errorf(
				"compute partial swap-step price: %w",
				err,
			)
		}
	}

	reachedTarget := sqrtPriceNextX96.Cmp(
		sqrtPriceTargetX96,
	) == 0

	if err := validateNextPriceDirection(
		sqrtPriceCurrentX96,
		sqrtPriceTargetX96,
		sqrtPriceNextX96,
		zeroForOne,
	); err != nil {
		return SwapStepResult{}, err
	}

	amountIn, amountOut, err := swapStepAmounts(
		sqrtPriceCurrentX96,
		sqrtPriceNextX96,
		liquidity,
		zeroForOne,
	)
	if err != nil {
		return SwapStepResult{}, err
	}

	var feeAmount *big.Int

	if reachedTarget {
		feeAmount, err = MulDivRoundingUp(
			amountIn,
			new(big.Int).SetUint64(uint64(feePips)),
			feeComplement,
		)
		if err != nil {
			return SwapStepResult{}, fmt.Errorf(
				"compute target-reaching swap-step fee: %w",
				err,
			)
		}
	} else {
		// When the target is not reached, all gross remaining input is
		// consumed by this step. The difference is the exact fee amount.
		if amountRemaining.Cmp(amountIn) < 0 {
			return SwapStepResult{}, fmt.Errorf(
				"swap-step input %s exceeds gross remaining amount %s",
				amountIn,
				amountRemaining,
			)
		}

		feeAmount = new(big.Int).Sub(
			amountRemaining,
			amountIn,
		)
	}

	totalConsumed := new(big.Int).Add(
		amountIn,
		feeAmount,
	)

	if totalConsumed.Cmp(amountRemaining) > 0 {
		return SwapStepResult{}, fmt.Errorf(
			"swap step consumed %s but only %s input remained",
			totalConsumed,
			amountRemaining,
		)
	}

	return newSwapStepResult(
		sqrtPriceNextX96,
		amountIn,
		amountOut,
		feeAmount,
		reachedTarget,
	), nil
}

func amountInForPriceMovement(
	sqrtPriceCurrentX96 *big.Int,
	sqrtPriceTargetX96 *big.Int,
	liquidity *big.Int,
	zeroForOne bool,
) (*big.Int, error) {
	if zeroForOne {
		amount, err := GetAmount0Delta(
			sqrtPriceTargetX96,
			sqrtPriceCurrentX96,
			liquidity,
			true,
		)
		if err != nil {
			return nil, fmt.Errorf(
				"compute token0 input required to reach target: %w",
				err,
			)
		}

		return amount, nil
	}

	amount, err := GetAmount1Delta(
		sqrtPriceCurrentX96,
		sqrtPriceTargetX96,
		liquidity,
		true,
	)
	if err != nil {
		return nil, fmt.Errorf(
			"compute token1 input required to reach target: %w",
			err,
		)
	}

	return amount, nil
}

func swapStepAmounts(
	sqrtPriceCurrentX96 *big.Int,
	sqrtPriceNextX96 *big.Int,
	liquidity *big.Int,
	zeroForOne bool,
) (*big.Int, *big.Int, error) {
	if zeroForOne {
		amountIn, err := GetAmount0Delta(
			sqrtPriceNextX96,
			sqrtPriceCurrentX96,
			liquidity,
			true,
		)
		if err != nil {
			return nil, nil, fmt.Errorf(
				"compute zero-for-one step input: %w",
				err,
			)
		}

		amountOut, err := GetAmount1Delta(
			sqrtPriceNextX96,
			sqrtPriceCurrentX96,
			liquidity,
			false,
		)
		if err != nil {
			return nil, nil, fmt.Errorf(
				"compute zero-for-one step output: %w",
				err,
			)
		}

		return amountIn, amountOut, nil
	}

	amountIn, err := GetAmount1Delta(
		sqrtPriceCurrentX96,
		sqrtPriceNextX96,
		liquidity,
		true,
	)
	if err != nil {
		return nil, nil, fmt.Errorf(
			"compute one-for-zero step input: %w",
			err,
		)
	}

	amountOut, err := GetAmount0Delta(
		sqrtPriceCurrentX96,
		sqrtPriceNextX96,
		liquidity,
		false,
	)
	if err != nil {
		return nil, nil, fmt.Errorf(
			"compute one-for-zero step output: %w",
			err,
		)
	}

	return amountIn, amountOut, nil
}

func validateNextPriceDirection(
	current *big.Int,
	target *big.Int,
	next *big.Int,
	zeroForOne bool,
) error {
	if zeroForOne {
		if next.Cmp(current) > 0 {
			return fmt.Errorf(
				"zero-for-one next price exceeds current price",
			)
		}

		if next.Cmp(target) < 0 {
			return fmt.Errorf(
				"zero-for-one next price moved beyond target price",
			)
		}

		return nil
	}

	if next.Cmp(current) < 0 {
		return fmt.Errorf(
			"one-for-zero next price is smaller than current price",
		)
	}

	if next.Cmp(target) > 0 {
		return fmt.Errorf(
			"one-for-zero next price moved beyond target price",
		)
	}

	return nil
}
