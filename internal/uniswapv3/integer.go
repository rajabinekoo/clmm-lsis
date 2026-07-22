package uniswapv3

import (
	"fmt"
	"math/big"
	"strings"
)

// cloneInt prevents callers from sharing mutable big.Int backing storage.
func cloneInt(value *big.Int) *big.Int {
	if value == nil {
		return nil
	}

	return new(big.Int).Set(value)
}

func mustDecimal(value string) *big.Int {
	parsed, ok := new(big.Int).SetString(value, 10)
	if !ok {
		panic(fmt.Sprintf(
			"invalid decimal integer constant %q",
			value,
		))
	}

	return parsed
}

func mustHex(value string) *big.Int {
	normalized := strings.TrimPrefix(value, "0x")

	parsed, ok := new(big.Int).SetString(normalized, 16)
	if !ok {
		panic(fmt.Sprintf(
			"invalid hexadecimal integer constant %q",
			value,
		))
	}

	return parsed
}

func powerOfTwo(exponent uint) *big.Int {
	return new(big.Int).Lsh(
		big.NewInt(1),
		exponent,
	)
}

func maxUnsignedInteger(bits uint) *big.Int {
	return new(big.Int).Sub(
		powerOfTwo(bits),
		big.NewInt(1),
	)
}

// validateUnsigned checks that a big integer belongs to the requested
// unsigned integer domain.
func validateUnsigned(
	name string,
	value *big.Int,
	bits uint,
) error {
	if value == nil {
		return fmt.Errorf("%s is required", name)
	}

	if value.Sign() < 0 {
		return fmt.Errorf("%s must not be negative", name)
	}

	if value.BitLen() > int(bits) {
		return fmt.Errorf(
			"%s exceeds uint%d",
			name,
			bits,
		)
	}

	return nil
}
