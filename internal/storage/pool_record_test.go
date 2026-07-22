package storage_test

import (
	"errors"
	"testing"

	"github.com/rajabinekoo/clmm-lsis/internal/storage"
)

func TestLegacyPoolRecordValidateAgainst(
	t *testing.T,
) {
	t.Parallel()

	pool := newStorageTestPool(t)

	record := storage.LegacyPoolRecord{
		Address: pool.Address.String(),

		Token0Address: "0x1111111111111111111111111111111111111111",

		Token1Address: "0x2222222222222222222222222222222222222222",

		Token0Decimals: pool.Token0.Decimals,
		Token1Decimals: pool.Token1.Decimals,

		FeeTier:     pool.FeePips,
		TickSpacing: pool.TickSpacing,

		CreatedBlock: 100,
	}

	if err := record.ValidateAgainst(pool); err != nil {
		t.Fatalf(
			"ValidateAgainst() error = %v",
			err,
		)
	}
}

func TestLegacyPoolRecordRejectsMetadataMismatch(
	t *testing.T,
) {
	t.Parallel()

	pool := newStorageTestPool(t)

	record := storage.LegacyPoolRecord{
		Address: pool.Address.String(),

		Token0Address: "0x1111111111111111111111111111111111111111",

		Token1Address: "0x2222222222222222222222222222222222222222",

		Token0Decimals: pool.Token0.Decimals,
		Token1Decimals: pool.Token1.Decimals,

		FeeTier:     3_000,
		TickSpacing: pool.TickSpacing,

		CreatedBlock: 100,
	}

	err := record.ValidateAgainst(pool)

	if !errors.Is(
		err,
		storage.ErrCheckpointMismatch,
	) {
		t.Fatalf(
			"ValidateAgainst() error = %v, want ErrCheckpointMismatch",
			err,
		)
	}
}
