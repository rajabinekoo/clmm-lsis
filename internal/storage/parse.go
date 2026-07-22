package storage

import (
	"fmt"
	"math/big"
	"strings"

	"github.com/rajabinekoo/clmm-lsis/internal/domain"
)

func parseStorageBigInt(
	fieldName string,
	value string,
) (*big.Int, error) {
	normalized := strings.TrimSpace(value)

	if normalized == "" {
		return nil, fmt.Errorf(
			"%w: %s is empty",
			ErrInvalidLegacyRecord,
			fieldName,
		)
	}

	parsed, ok := new(big.Int).SetString(
		normalized,
		10,
	)
	if !ok {
		return nil, fmt.Errorf(
			"%w: parse %s as integer: %q",
			ErrInvalidLegacyRecord,
			fieldName,
			value,
		)
	}

	return parsed, nil
}

func parseRequiredStorageAddress(
	fieldName string,
	value string,
) (domain.Address, error) {
	address, err := domain.ParseAddress(
		strings.TrimSpace(value),
	)
	if err != nil {
		return "", fmt.Errorf(
			"%w: parse %s: %v",
			ErrInvalidLegacyRecord,
			fieldName,
			err,
		)
	}

	return address, nil
}

func parseOptionalStorageAddress(
	fieldName string,
	value *string,
) (domain.Address, bool, error) {
	if value == nil {
		return "", false, nil
	}

	normalized := strings.TrimSpace(*value)

	if normalized == "" {
		return "", false, nil
	}

	address, err := domain.ParseAddress(normalized)
	if err != nil {
		return "", false, fmt.Errorf(
			"%w: parse %s: %v",
			ErrInvalidLegacyRecord,
			fieldName,
			err,
		)
	}

	return address, true, nil
}

func parseRequiredStorageHash(
	fieldName string,
	value string,
) (domain.Hash, error) {
	hash, err := domain.ParseHash(
		strings.TrimSpace(value),
	)
	if err != nil {
		return "", fmt.Errorf(
			"%w: parse %s: %v",
			ErrInvalidLegacyRecord,
			fieldName,
			err,
		)
	}

	return hash, nil
}

func parseOptionalStorageHash(
	fieldName string,
	value *string,
) (domain.Hash, error) {
	if value == nil {
		return "", nil
	}

	normalized := strings.TrimSpace(*value)

	if normalized == "" {
		return "", nil
	}

	hash, err := domain.ParseHash(normalized)
	if err != nil {
		return "", fmt.Errorf(
			"%w: parse %s: %v",
			ErrInvalidLegacyRecord,
			fieldName,
			err,
		)
	}

	return hash, nil
}
