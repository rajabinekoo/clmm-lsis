package postgres

import (
	"database/sql"
	"fmt"
)

const (
	maxUint32AsInt64 = int64(^uint32(0))

	minInt32AsInt64 = int64(-1 << 31)
	maxInt32AsInt64 = int64(1<<31 - 1)
)

func checkedUint64(
	fieldName string,
	value int64,
) (uint64, error) {
	if value < 0 {
		return 0, fmt.Errorf(
			"%s must not be negative: %d",
			fieldName,
			value,
		)
	}

	return uint64(value), nil
}

func checkedUint32(
	fieldName string,
	value int64,
) (uint32, error) {
	if value < 0 ||
		value > maxUint32AsInt64 {
		return 0, fmt.Errorf(
			"%s is outside uint32 range: %d",
			fieldName,
			value,
		)
	}

	return uint32(value), nil
}

func checkedInt32(
	fieldName string,
	value int64,
) (int32, error) {
	if value < minInt32AsInt64 ||
		value > maxInt32AsInt64 {
		return 0, fmt.Errorf(
			"%s is outside int32 range: %d",
			fieldName,
			value,
		)
	}

	return int32(value), nil
}

func nullableStringPointer(
	value sql.NullString,
) *string {
	if !value.Valid {
		return nil
	}

	copied := value.String

	return &copied
}

func nullableUint32Pointer(
	fieldName string,
	value sql.NullInt64,
) (*uint32, error) {
	if !value.Valid {
		return nil, nil
	}

	converted, err := checkedUint32(
		fieldName,
		value.Int64,
	)
	if err != nil {
		return nil, err
	}

	return &converted, nil
}
