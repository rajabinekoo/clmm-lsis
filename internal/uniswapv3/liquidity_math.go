package uniswapv3

import (
	"fmt"
	"math/big"
)

// AddLiquidityDelta applies a signed liquidity change to a uint128 liquidity
// value.
//
// The current liquidity and final result must remain inside the uint128 domain.
// A negative result indicates inconsistent tick accounting and is rejected.
func AddLiquidityDelta(
	current *big.Int,
	delta *big.Int,
) (*big.Int, error) {
	if err := validateUnsigned(
		"current liquidity",
		current,
		128,
	); err != nil {
		return nil, err
	}

	if delta == nil {
		return nil, fmt.Errorf("liquidity delta is required")
	}

	result := new(big.Int).Add(
		current,
		delta,
	)

	if result.Sign() < 0 {
		return nil, fmt.Errorf(
			"liquidity delta %s makes active liquidity negative from current value %s",
			delta,
			current,
		)
	}

	if result.Cmp(maxUint128) > 0 {
		return nil, fmt.Errorf(
			"resulting active liquidity %s exceeds uint128",
			result,
		)
	}

	return result, nil
}

// ApplyLiquidityNet applies one initialized tick's liquidityNet value while
// crossing that tick.
//
// Moving from token0 to token1 decreases the sqrt price and crosses the tick
// from right to left. Therefore, liquidityNet must be negated.
//
// Moving from token1 to token0 increases the sqrt price and crosses the tick
// from left to right. In that direction, liquidityNet is applied directly.
func ApplyLiquidityNet(
	currentLiquidity *big.Int,
	liquidityNet *big.Int,
	zeroForOne bool,
) (*big.Int, error) {
	if liquidityNet == nil {
		return nil, fmt.Errorf("tick liquidity net is required")
	}

	delta := cloneInt(liquidityNet)

	if zeroForOne {
		delta.Neg(delta)
	}

	return AddLiquidityDelta(
		currentLiquidity,
		delta,
	)
}
