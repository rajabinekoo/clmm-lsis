package domain

import (
	"encoding/hex"
	"fmt"
	"strings"
)

const ethereumAddressHexLength = 40

// Address is a canonical, lower-case Ethereum address.
//
// Domain addresses are parsed at system boundaries. Internal services should
// not repeatedly validate or normalize raw address strings.
type Address string

// ParseAddress validates and canonicalizes an Ethereum address.
func ParseAddress(value string) (Address, error) {
	normalized := strings.ToLower(strings.TrimSpace(value))

	if !strings.HasPrefix(normalized, "0x") {
		return "", fmt.Errorf("ethereum address must start with 0x")
	}

	rawHex := normalized[2:]

	if len(rawHex) != ethereumAddressHexLength {
		return "", fmt.Errorf(
			"ethereum address must contain %d hexadecimal characters",
			ethereumAddressHexLength,
		)
	}

	if _, err := hex.DecodeString(rawHex); err != nil {
		return "", fmt.Errorf("ethereum address contains invalid hexadecimal data: %w", err)
	}

	return Address(normalized), nil
}

func (a Address) String() string {
	return string(a)
}

func (a Address) IsZero() bool {
	return a == ""
}
