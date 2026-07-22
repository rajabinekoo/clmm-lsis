package uniswapv3

import (
	"fmt"
	"math/big"
)

// GetNextSqrtPriceFromInput computes the next sqrt price after consuming an
// exact amount of token input.
//
// For zeroForOne swaps, token0 enters the pool and the sqrt price decreases.
// For oneForZero swaps, token1 enters the pool and the sqrt price increases.
//
// amountIn is the amount remaining after deducting the swap fee.
func GetNextSqrtPriceFromInput(
	sqrtPriceX96 *big.Int,
	liquidity *big.Int,
	amountIn *big.Int,
	zeroForOne bool,
) (*big.Int, error) {
	if err := validateSqrtPriceX96(
		"current sqrt price X96",
		sqrtPriceX96,
	); err != nil {
		return nil, err
	}

	if err := validateLiquidity(liquidity); err != nil {
		return nil, err
	}

	if err := validateUnsigned(
		"input amount",
		amountIn,
		256,
	); err != nil {
		return nil, err
	}

	if amountIn.Sign() == 0 {
		return cloneInt(sqrtPriceX96), nil
	}

	if zeroForOne {
		return GetNextSqrtPriceFromAmount0RoundingUp(
			sqrtPriceX96,
			liquidity,
			amountIn,
			true,
		)
	}

	return GetNextSqrtPriceFromAmount1RoundingDown(
		sqrtPriceX96,
		liquidity,
		amountIn,
		true,
	)
}

// GetNextSqrtPriceFromAmount0RoundingUp computes the next sqrt price after
// adding or removing token0.
//
// The result always rounds upward. This direction of rounding prevents an
// exact-input swap from sending more token output than the protocol permits.
//
// The implementation deliberately preserves the overflow fallback used by
// Uniswap v3, even though big.Int itself does not overflow. Preserving that
// branch keeps integer rounding behavior aligned with the on-chain algorithm.
func GetNextSqrtPriceFromAmount0RoundingUp(
	sqrtPriceX96 *big.Int,
	liquidity *big.Int,
	amount *big.Int,
	add bool,
) (*big.Int, error) {
	if err := validateSqrtPriceX96(
		"sqrt price X96",
		sqrtPriceX96,
	); err != nil {
		return nil, err
	}

	if err := validateLiquidity(liquidity); err != nil {
		return nil, err
	}

	if err := validateUnsigned(
		"token0 amount",
		amount,
		256,
	); err != nil {
		return nil, err
	}

	if amount.Sign() == 0 {
		return cloneInt(sqrtPriceX96), nil
	}

	numerator1 := new(big.Int).Lsh(
		cloneInt(liquidity),
		fixedPoint96Resolution,
	)

	product := new(big.Int).Mul(
		amount,
		sqrtPriceX96,
	)

	if add {
		// The direct protocol path is used only when amount * sqrtPrice and
		// numerator1 + product both fit inside uint256.
		if product.Cmp(maxUint256) <= 0 {
			denominator := new(big.Int).Add(
				numerator1,
				product,
			)

			if denominator.Cmp(maxUint256) <= 0 &&
				denominator.Cmp(numerator1) >= 0 {
				next, err := MulDivRoundingUp(
					numerator1,
					sqrtPriceX96,
					denominator,
				)
				if err != nil {
					return nil, fmt.Errorf(
						"compute token0-add sqrt price: %w",
						err,
					)
				}

				if err := validateSqrtPriceX96(
					"next sqrt price X96",
					next,
				); err != nil {
					return nil, err
				}

				return next, nil
			}
		}

		// Overflow-safe algebraic form used by the Solidity implementation:
		//
		// ceil(
		//     numerator1 /
		//     (numerator1 / sqrtPriceX96 + amount)
		// )
		baseDenominator := new(big.Int).Quo(
			numerator1,
			sqrtPriceX96,
		)

		denominator := new(big.Int).Add(
			baseDenominator,
			amount,
		)

		if denominator.Cmp(maxUint256) > 0 {
			return nil, fmt.Errorf(
				"token0-add fallback denominator exceeds uint256",
			)
		}

		next, err := DivRoundingUp(
			numerator1,
			denominator,
		)
		if err != nil {
			return nil, fmt.Errorf(
				"compute token0-add fallback sqrt price: %w",
				err,
			)
		}

		if err := validateSqrtPriceX96(
			"next sqrt price X96",
			next,
		); err != nil {
			return nil, err
		}

		return next, nil
	}

	if product.Cmp(maxUint256) > 0 {
		return nil, fmt.Errorf(
			"token0-remove product exceeds uint256",
		)
	}

	if numerator1.Cmp(product) <= 0 {
		return nil, fmt.Errorf(
			"token0 removal amount is too large for the current liquidity and price",
		)
	}

	denominator := new(big.Int).Sub(
		numerator1,
		product,
	)

	next, err := MulDivRoundingUp(
		numerator1,
		sqrtPriceX96,
		denominator,
	)
	if err != nil {
		return nil, fmt.Errorf(
			"compute token0-remove sqrt price: %w",
			err,
		)
	}

	if err := validateSqrtPriceX96(
		"next sqrt price X96",
		next,
	); err != nil {
		return nil, err
	}

	return next, nil
}

// GetNextSqrtPriceFromAmount1RoundingDown computes the next sqrt price after
// adding or removing token1.
//
// Adding token1 rounds down. Removing token1 rounds up before subtracting.
// Both choices preserve the pool-favoring rounding semantics of Uniswap v3.
func GetNextSqrtPriceFromAmount1RoundingDown(
	sqrtPriceX96 *big.Int,
	liquidity *big.Int,
	amount *big.Int,
	add bool,
) (*big.Int, error) {
	if err := validateSqrtPriceX96(
		"sqrt price X96",
		sqrtPriceX96,
	); err != nil {
		return nil, err
	}

	if err := validateLiquidity(liquidity); err != nil {
		return nil, err
	}

	if err := validateUnsigned(
		"token1 amount",
		amount,
		256,
	); err != nil {
		return nil, err
	}

	if amount.Sign() == 0 {
		return cloneInt(sqrtPriceX96), nil
	}

	var (
		quotient *big.Int
		err      error
	)

	if add {
		if amount.Cmp(maxUint160) <= 0 {
			shiftedAmount := new(big.Int).Lsh(
				cloneInt(amount),
				fixedPoint96Resolution,
			)

			quotient = new(big.Int).Quo(
				shiftedAmount,
				liquidity,
			)
		} else {
			quotient, err = MulDiv(
				amount,
				q96,
				liquidity,
			)
			if err != nil {
				return nil, fmt.Errorf(
					"compute token1-add quotient: %w",
					err,
				)
			}
		}

		next := new(big.Int).Add(
			sqrtPriceX96,
			quotient,
		)

		if err := validateSqrtPriceX96(
			"next sqrt price X96",
			next,
		); err != nil {
			return nil, err
		}

		return next, nil
	}

	if amount.Cmp(maxUint160) <= 0 {
		shiftedAmount := new(big.Int).Lsh(
			cloneInt(amount),
			fixedPoint96Resolution,
		)

		quotient, err = DivRoundingUp(
			shiftedAmount,
			liquidity,
		)
		if err != nil {
			return nil, fmt.Errorf(
				"compute token1-remove quotient: %w",
				err,
			)
		}
	} else {
		quotient, err = MulDivRoundingUp(
			amount,
			q96,
			liquidity,
		)
		if err != nil {
			return nil, fmt.Errorf(
				"compute token1-remove quotient: %w",
				err,
			)
		}
	}

	if sqrtPriceX96.Cmp(quotient) <= 0 {
		return nil, fmt.Errorf(
			"token1 removal amount is too large for the current sqrt price",
		)
	}

	next := new(big.Int).Sub(
		sqrtPriceX96,
		quotient,
	)

	if err := validateSqrtPriceX96(
		"next sqrt price X96",
		next,
	); err != nil {
		return nil, err
	}

	return next, nil
}

// GetAmount0Delta computes the token0 quantity represented by moving between
// two sqrt prices at constant liquidity.
//
// The sqrt-price arguments may be provided in either order.
func GetAmount0Delta(
	sqrtPriceAX96 *big.Int,
	sqrtPriceBX96 *big.Int,
	liquidity *big.Int,
	roundUp bool,
) (*big.Int, error) {
	if err := validateSqrtPriceX96(
		"sqrt price A X96",
		sqrtPriceAX96,
	); err != nil {
		return nil, err
	}

	if err := validateSqrtPriceX96(
		"sqrt price B X96",
		sqrtPriceBX96,
	); err != nil {
		return nil, err
	}

	if err := validateLiquidityAllowZero(liquidity); err != nil {
		return nil, err
	}

	if liquidity.Sign() == 0 ||
		sqrtPriceAX96.Cmp(sqrtPriceBX96) == 0 {
		return new(big.Int), nil
	}

	lower, upper := orderedSqrtPrices(
		sqrtPriceAX96,
		sqrtPriceBX96,
	)

	numerator1 := new(big.Int).Lsh(
		cloneInt(liquidity),
		fixedPoint96Resolution,
	)

	numerator2 := new(big.Int).Sub(
		upper,
		lower,
	)

	if roundUp {
		intermediate, err := MulDivRoundingUp(
			numerator1,
			numerator2,
			upper,
		)
		if err != nil {
			return nil, fmt.Errorf(
				"compute rounded token0 delta intermediate: %w",
				err,
			)
		}

		result, err := DivRoundingUp(
			intermediate,
			lower,
		)
		if err != nil {
			return nil, fmt.Errorf(
				"compute rounded token0 delta: %w",
				err,
			)
		}

		return result, nil
	}

	intermediate, err := MulDiv(
		numerator1,
		numerator2,
		upper,
	)
	if err != nil {
		return nil, fmt.Errorf(
			"compute token0 delta intermediate: %w",
			err,
		)
	}

	return new(big.Int).Quo(
		intermediate,
		lower,
	), nil
}

// GetAmount1Delta computes the token1 quantity represented by moving between
// two sqrt prices at constant liquidity.
//
// The sqrt-price arguments may be provided in either order.
func GetAmount1Delta(
	sqrtPriceAX96 *big.Int,
	sqrtPriceBX96 *big.Int,
	liquidity *big.Int,
	roundUp bool,
) (*big.Int, error) {
	if err := validateSqrtPriceX96(
		"sqrt price A X96",
		sqrtPriceAX96,
	); err != nil {
		return nil, err
	}

	if err := validateSqrtPriceX96(
		"sqrt price B X96",
		sqrtPriceBX96,
	); err != nil {
		return nil, err
	}

	if err := validateLiquidityAllowZero(liquidity); err != nil {
		return nil, err
	}

	if liquidity.Sign() == 0 ||
		sqrtPriceAX96.Cmp(sqrtPriceBX96) == 0 {
		return new(big.Int), nil
	}

	lower, upper := orderedSqrtPrices(
		sqrtPriceAX96,
		sqrtPriceBX96,
	)

	difference := new(big.Int).Sub(
		upper,
		lower,
	)

	if roundUp {
		result, err := MulDivRoundingUp(
			liquidity,
			difference,
			q96,
		)
		if err != nil {
			return nil, fmt.Errorf(
				"compute rounded token1 delta: %w",
				err,
			)
		}

		return result, nil
	}

	result, err := MulDiv(
		liquidity,
		difference,
		q96,
	)
	if err != nil {
		return nil, fmt.Errorf(
			"compute token1 delta: %w",
			err,
		)
	}

	return result, nil
}

func orderedSqrtPrices(
	left *big.Int,
	right *big.Int,
) (*big.Int, *big.Int) {
	if left.Cmp(right) <= 0 {
		return cloneInt(left), cloneInt(right)
	}

	return cloneInt(right), cloneInt(left)
}

func validateSqrtPriceX96(
	name string,
	value *big.Int,
) error {
	if err := validateUnsigned(
		name,
		value,
		160,
	); err != nil {
		return err
	}

	if value.Sign() == 0 {
		return fmt.Errorf("%s must be greater than zero", name)
	}

	return nil
}

func validateLiquidity(
	liquidity *big.Int,
) error {
	if err := validateUnsigned(
		"liquidity",
		liquidity,
		128,
	); err != nil {
		return err
	}

	if liquidity.Sign() == 0 {
		return fmt.Errorf("liquidity must be greater than zero")
	}

	return nil
}

func validateLiquidityAllowZero(
	liquidity *big.Int,
) error {
	return validateUnsigned(
		"liquidity",
		liquidity,
		128,
	)
}
