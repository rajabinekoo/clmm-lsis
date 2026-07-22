package domain

import (
	"fmt"
	"math/big"
)

func cloneBigInt(value *big.Int) *big.Int {
	if value == nil {
		return nil
	}

	return new(big.Int).Set(value)
}

func requireBigInt(
	fieldName string,
	value *big.Int,
) error {
	if value == nil {
		return fmt.Errorf("%s is required", fieldName)
	}

	return nil
}

func requirePositiveBigInt(
	fieldName string,
	value *big.Int,
) error {
	if err := requireBigInt(fieldName, value); err != nil {
		return err
	}

	if value.Sign() <= 0 {
		return fmt.Errorf("%s must be greater than zero", fieldName)
	}

	return nil
}

func requireNonNegativeBigInt(
	fieldName string,
	value *big.Int,
) error {
	if err := requireBigInt(fieldName, value); err != nil {
		return err
	}

	if value.Sign() < 0 {
		return fmt.Errorf("%s must not be negative", fieldName)
	}

	return nil
}

func absoluteBigInt(value *big.Int) *big.Int {
	if value == nil {
		return nil
	}

	return new(big.Int).Abs(value)
}
