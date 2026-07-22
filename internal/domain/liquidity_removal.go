package domain

import (
	"fmt"
	"math/big"
)

// LiquidityRemoval describes an observed full or partial withdrawal from one
// position.
type LiquidityRemoval struct {
	liquidityBefore  *big.Int
	liquidityRemoved *big.Int
}

func NewLiquidityRemoval(
	liquidityBefore *big.Int,
	liquidityRemoved *big.Int,
) (LiquidityRemoval, error) {
	removal := LiquidityRemoval{
		liquidityBefore:  cloneBigInt(liquidityBefore),
		liquidityRemoved: cloneBigInt(liquidityRemoved),
	}

	if err := removal.Validate(); err != nil {
		return LiquidityRemoval{}, err
	}

	return removal, nil
}

func (r LiquidityRemoval) Validate() error {
	if err := requirePositiveBigInt(
		"liquidity before removal",
		r.liquidityBefore,
	); err != nil {
		return err
	}

	if err := requirePositiveBigInt(
		"liquidity removed",
		r.liquidityRemoved,
	); err != nil {
		return err
	}

	if r.liquidityRemoved.Cmp(r.liquidityBefore) > 0 {
		return fmt.Errorf(
			"removed liquidity %s exceeds position liquidity %s",
			r.liquidityRemoved,
			r.liquidityBefore,
		)
	}

	return nil
}

func (r LiquidityRemoval) LiquidityBefore() *big.Int {
	return cloneBigInt(r.liquidityBefore)
}

func (r LiquidityRemoval) LiquidityRemoved() *big.Int {
	return cloneBigInt(r.liquidityRemoved)
}

func (r LiquidityRemoval) RemainingLiquidity() *big.Int {
	return new(big.Int).Sub(
		r.liquidityBefore,
		r.liquidityRemoved,
	)
}

func (r LiquidityRemoval) IsFullRemoval() bool {
	return r.liquidityRemoved.Cmp(r.liquidityBefore) == 0
}

// Fraction returns the exact rational removal fraction.
//
// The value is intentionally represented as big.Rat rather than float64 so
// that event treatment intensity is never rounded during data preparation.
func (r LiquidityRemoval) Fraction() *big.Rat {
	return new(big.Rat).SetFrac(
		r.liquidityRemoved,
		r.liquidityBefore,
	)
}
