package domain

import (
	"encoding/hex"
	"fmt"
	"strings"
)

const ethereumHashHexLength = 64

// Hash is a canonical, lower-case Ethereum 32-byte hash.
type Hash string

func ParseHash(value string) (Hash, error) {
	normalized := strings.ToLower(strings.TrimSpace(value))

	if !strings.HasPrefix(normalized, "0x") {
		return "", fmt.Errorf("ethereum hash must start with 0x")
	}

	rawHex := normalized[2:]

	if len(rawHex) != ethereumHashHexLength {
		return "", fmt.Errorf(
			"ethereum hash must contain %d hexadecimal characters",
			ethereumHashHexLength,
		)
	}

	if _, err := hex.DecodeString(rawHex); err != nil {
		return "", fmt.Errorf("ethereum hash contains invalid hexadecimal data: %w", err)
	}

	return Hash(normalized), nil
}

func (h Hash) String() string {
	return string(h)
}

func (h Hash) IsZero() bool {
	return h == ""
}
