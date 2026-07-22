package uniswapv3

import (
	"fmt"
	"math/big"
)

// MulDiv computes:
//
//	floor(a * b / denominator)
//
// The multiplication uses an exact, unbounded intermediate value. The method
// then verifies that the final quotient fits in uint256.
//
// Go's big.Int removes the overflow-management machinery required in Solidity
// while preserving the same mathematical result for valid uint256 inputs.
func MulDiv(
	a *big.Int,
	b *big.Int,
	denominator *big.Int,
) (*big.Int, error) {
	if err := validateUnsigned(
		"mul-div multiplicand a",
		a,
		256,
	); err != nil {
		return nil, err
	}

	if err := validateUnsigned(
		"mul-div multiplicand b",
		b,
		256,
	); err != nil {
		return nil, err
	}

	if err := validateUnsigned(
		"mul-div denominator",
		denominator,
		256,
	); err != nil {
		return nil, err
	}

	if denominator.Sign() == 0 {
		return nil, fmt.Errorf(
			"mul-div denominator must be greater than zero",
		)
	}

	product := new(big.Int).Mul(a, b)
	quotient := new(big.Int).Quo(
		product,
		denominator,
	)

	if quotient.Cmp(maxUint256) > 0 {
		return nil, fmt.Errorf(
			"mul-div result exceeds uint256",
		)
	}

	return quotient, nil
}

// MulDivRoundingUp computes:
//
//	ceil(a * b / denominator)
//
// The method only increments the quotient when the exact division has a
// non-zero remainder.
func MulDivRoundingUp(
	a *big.Int,
	b *big.Int,
	denominator *big.Int,
) (*big.Int, error) {
	quotient, err := MulDiv(
		a,
		b,
		denominator,
	)
	if err != nil {
		return nil, err
	}

	product := new(big.Int).Mul(a, b)
	remainder := new(big.Int).Mod(
		product,
		denominator,
	)

	if remainder.Sign() == 0 {
		return quotient, nil
	}

	if quotient.Cmp(maxUint256) == 0 {
		return nil, fmt.Errorf(
			"rounded mul-div result exceeds uint256",
		)
	}

	return quotient.Add(
		quotient,
		big.NewInt(1),
	), nil
}

// DivRoundingUp computes:
//
//	ceil(numerator / denominator)
//
// It is used when protocol semantics require rounding in favor of the pool.
func DivRoundingUp(
	numerator *big.Int,
	denominator *big.Int,
) (*big.Int, error) {
	if err := validateUnsigned(
		"division numerator",
		numerator,
		256,
	); err != nil {
		return nil, err
	}

	if err := validateUnsigned(
		"division denominator",
		denominator,
		256,
	); err != nil {
		return nil, err
	}

	if denominator.Sign() == 0 {
		return nil, fmt.Errorf(
			"division denominator must be greater than zero",
		)
	}

	quotient, remainder := new(big.Int).QuoRem(
		numerator,
		denominator,
		new(big.Int),
	)

	if remainder.Sign() == 0 {
		return quotient, nil
	}

	if quotient.Cmp(maxUint256) == 0 {
		return nil, fmt.Errorf(
			"rounded division result exceeds uint256",
		)
	}

	return quotient.Add(
		quotient,
		big.NewInt(1),
	), nil
}
