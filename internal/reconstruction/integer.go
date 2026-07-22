package reconstruction

import "math/big"

// cloneBigInt prevents mutable big.Int values from being shared between the
// reconstruction state and its callers.
func cloneBigInt(
	value *big.Int,
) *big.Int {
	if value == nil {
		return nil
	}

	return new(big.Int).Set(value)
}
