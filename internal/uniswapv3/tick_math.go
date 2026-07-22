package uniswapv3

import (
	"fmt"
	"math/big"
)

type tickMultiplier struct {
	mask   uint32
	factor *big.Int
}

var tickMultipliers = []tickMultiplier{
	{
		mask:   0x1,
		factor: mustHex("fffcb933bd6fad37aa2d162d1a594001"),
	},
	{
		mask:   0x2,
		factor: mustHex("fff97272373d413259a46990580e213a"),
	},
	{
		mask:   0x4,
		factor: mustHex("fff2e50f5f656932ef12357cf3c7fdcc"),
	},
	{
		mask:   0x8,
		factor: mustHex("ffe5caca7e10e4e61c3624eaa0941cd0"),
	},
	{
		mask:   0x10,
		factor: mustHex("ffcb9843d60f6159c9db58835c926644"),
	},
	{
		mask:   0x20,
		factor: mustHex("ff973b41fa98c081472e6896dfb254c0"),
	},
	{
		mask:   0x40,
		factor: mustHex("ff2ea16466c96a3843ec78b326b52861"),
	},
	{
		mask:   0x80,
		factor: mustHex("fe5dee046a99a2a811c461f1969c3053"),
	},
	{
		mask:   0x100,
		factor: mustHex("fcbe86c7900a88aedcffc83b479aa3a4"),
	},
	{
		mask:   0x200,
		factor: mustHex("f987a7253ac413176f2b074cf7815e54"),
	},
	{
		mask:   0x400,
		factor: mustHex("f3392b0822b70005940c7a398e4b70f3"),
	},
	{
		mask:   0x800,
		factor: mustHex("e7159475a2c29b7443b29c7fa6e889d9"),
	},
	{
		mask:   0x1000,
		factor: mustHex("d097f3bdfd2022b8845ad8f792aa5825"),
	},
	{
		mask:   0x2000,
		factor: mustHex("a9f746462d870fdf8a65dc1f90e061e5"),
	},
	{
		mask:   0x4000,
		factor: mustHex("70d869a156d2a1b890bb3df62baf32f7"),
	},
	{
		mask:   0x8000,
		factor: mustHex("31be135f97d08fd981231505542fcfa6"),
	},
	{
		mask:   0x10000,
		factor: mustHex("9aa508b5b7a84e1c677de54f3e99bc9"),
	},
	{
		mask:   0x20000,
		factor: mustHex("5d6af8dedb81196699c329225ee604"),
	},
	{
		mask:   0x40000,
		factor: mustHex("2216e584f5fa1ea926041bedfe98"),
	},
	{
		mask:   0x80000,
		factor: mustHex("48a170391f7dc42444e8fa2"),
	},
}

// GetSqrtRatioAtTick returns:
//
//	ceil(sqrt(1.0001^tick) * 2^96)
//
// All operations use exact integer arithmetic.
func GetSqrtRatioAtTick(
	tick int32,
) (*big.Int, error) {
	if tick < MinTick || tick > MaxTick {
		return nil, fmt.Errorf(
			"tick %d is outside supported range [%d,%d]",
			tick,
			MinTick,
			MaxTick,
		)
	}

	absTick := uint32(tick)

	if tick < 0 {
		// Convert through int64 so negating MinTick remains safe and explicit.
		absTick = uint32(-int64(tick))
	}

	ratio := cloneInt(q128)

	for _, multiplier := range tickMultipliers {
		if absTick&multiplier.mask == 0 {
			continue
		}

		ratio.Mul(
			ratio,
			multiplier.factor,
		)

		ratio.Rsh(
			ratio,
			fixedPoint128Resolution,
		)
	}

	if tick > 0 {
		ratio.Quo(
			maxUint256,
			ratio,
		)
	}

	const q128ToQ96Shift = 32

	sqrtPriceX96 := new(big.Int).Rsh(
		cloneInt(ratio),
		q128ToQ96Shift,
	)

	// Moving from Q128.128 to Q128.96 discards 32 fractional bits.
	// Round upward whenever at least one discarded bit is non-zero.
	lowBitsMask := new(big.Int).Sub(
		powerOfTwo(q128ToQ96Shift),
		big.NewInt(1),
	)

	if new(big.Int).And(
		ratio,
		lowBitsMask,
	).Sign() != 0 {
		sqrtPriceX96.Add(
			sqrtPriceX96,
			big.NewInt(1),
		)
	}

	if sqrtPriceX96.Cmp(maxUint160) > 0 {
		return nil, fmt.Errorf(
			"sqrt price for tick %d exceeds uint160",
			tick,
		)
	}

	return sqrtPriceX96, nil
}

// GetTickAtSqrtRatio returns the greatest tick satisfying:
//
//	GetSqrtRatioAtTick(tick) <= sqrtPriceX96
//
// The inverse uses an exact binary search across the bounded tick domain.
// This is slower than the gas-optimized contract implementation, but simpler
// to audit and sufficiently fast for offline reconstruction and simulation.
func GetTickAtSqrtRatio(
	sqrtPriceX96 *big.Int,
) (int32, error) {
	if err := validateUnsigned(
		"sqrt price X96",
		sqrtPriceX96,
		160,
	); err != nil {
		return 0, err
	}

	if sqrtPriceX96.Cmp(minSqrtRatio) < 0 ||
		sqrtPriceX96.Cmp(maxSqrtRatio) >= 0 {
		return 0, fmt.Errorf(
			"sqrt price X96 %s is outside supported range [%s,%s)",
			sqrtPriceX96,
			minSqrtRatio,
			maxSqrtRatio,
		)
	}

	low := MinTick

	// MaxSqrtRatio itself is excluded from the valid input range. Therefore,
	// the greatest possible returned tick is MaxTick-1.
	high := MaxTick - 1

	result := MinTick

	for low <= high {
		mid := low + (high-low)/2

		midRatio, err := GetSqrtRatioAtTick(mid)
		if err != nil {
			return 0, fmt.Errorf(
				"compute sqrt ratio for candidate tick %d: %w",
				mid,
				err,
			)
		}

		if midRatio.Cmp(sqrtPriceX96) <= 0 {
			result = mid
			low = mid + 1

			continue
		}

		high = mid - 1
	}

	return result, nil
}
