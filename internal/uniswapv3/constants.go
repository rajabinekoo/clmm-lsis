package uniswapv3

import "math/big"

const (
	// MinTick and MaxTick bound the tick domain supported by Uniswap v3.
	MinTick int32 = -887272
	MaxTick int32 = 887272

	fixedPoint96Resolution  uint = 96
	fixedPoint128Resolution uint = 128
)

var (
	q96  = powerOfTwo(fixedPoint96Resolution)
	q128 = powerOfTwo(fixedPoint128Resolution)

	maxUint160 = maxUnsignedInteger(160)
	maxUint256 = maxUnsignedInteger(256)

	minSqrtRatio = mustDecimal("4295128739")
	maxSqrtRatio = mustDecimal(
		"1461446703485210103287273052203988822378723970342",
	)
)

// Q96 returns 2^96, the scaling factor used by Q64.96 sqrt prices.
//
// The returned integer is an independent copy and may safely be mutated by
// the caller.
func Q96() *big.Int {
	return cloneInt(q96)
}

// Q128 returns 2^128, the scaling factor used by Q128.128 intermediates.
//
// The returned integer is an independent copy and may safely be mutated by
// the caller.
func Q128() *big.Int {
	return cloneInt(q128)
}

// MinSqrtRatio returns the minimum sqrt price produced by MinTick.
func MinSqrtRatio() *big.Int {
	return cloneInt(minSqrtRatio)
}

// MaxSqrtRatio returns the upper sqrt-price boundary used by Uniswap v3.
//
// Valid pool sqrt prices are strictly smaller than this value.
func MaxSqrtRatio() *big.Int {
	return cloneInt(maxSqrtRatio)
}
