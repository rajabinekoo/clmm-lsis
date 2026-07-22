package domain

import (
	"fmt"
	"math/big"
)

// LiquidityDelta is a signed change in position and tick liquidity.
//
// Positive values add liquidity. Negative values remove liquidity.
type LiquidityDelta struct {
	value *big.Int
}

func NewLiquidityDelta(
	value *big.Int,
) (LiquidityDelta, error) {
	if err := requireBigInt("liquidity delta", value); err != nil {
		return LiquidityDelta{}, err
	}

	if value.Sign() == 0 {
		return LiquidityDelta{}, fmt.Errorf(
			"liquidity delta must not be zero",
		)
	}

	return LiquidityDelta{
		value: cloneBigInt(value),
	}, nil
}

func NewMintLiquidityDelta(
	amount *big.Int,
) (LiquidityDelta, error) {
	if err := requirePositiveBigInt(
		"mint liquidity amount",
		amount,
	); err != nil {
		return LiquidityDelta{}, err
	}

	return LiquidityDelta{
		value: cloneBigInt(amount),
	}, nil
}

func NewBurnLiquidityDelta(
	amount *big.Int,
) (LiquidityDelta, error) {
	if err := requirePositiveBigInt(
		"burn liquidity amount",
		amount,
	); err != nil {
		return LiquidityDelta{}, err
	}

	return LiquidityDelta{
		value: new(big.Int).Neg(amount),
	}, nil
}

func (d LiquidityDelta) Validate() error {
	if err := requireBigInt("liquidity delta", d.value); err != nil {
		return err
	}

	if d.value.Sign() == 0 {
		return fmt.Errorf("liquidity delta must not be zero")
	}

	return nil
}

func (d LiquidityDelta) Value() *big.Int {
	return cloneBigInt(d.value)
}

func (d LiquidityDelta) AbsoluteValue() *big.Int {
	return absoluteBigInt(d.value)
}

func (d LiquidityDelta) IsMint() bool {
	return d.value != nil && d.value.Sign() > 0
}

func (d LiquidityDelta) IsBurn() bool {
	return d.value != nil && d.value.Sign() < 0
}

// Apply applies the signed delta while preventing negative liquidity.
func (d LiquidityDelta) Apply(
	current *big.Int,
) (*big.Int, error) {
	if err := requireNonNegativeBigInt(
		"current liquidity",
		current,
	); err != nil {
		return nil, err
	}

	if err := d.Validate(); err != nil {
		return nil, err
	}

	result := new(big.Int).Add(current, d.value)

	if result.Sign() < 0 {
		return nil, fmt.Errorf(
			"liquidity delta %s exceeds current liquidity %s",
			d.value,
			current,
		)
	}

	return result, nil
}
