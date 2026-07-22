package storage_test

import (
	"fmt"
	"testing"

	"github.com/rajabinekoo/clmm-lsis/internal/domain"
	"github.com/rajabinekoo/clmm-lsis/internal/storage"
)

func TestNewSwapIndexRange(
	t *testing.T,
) {
	t.Parallel()

	poolAddress := mustSwapSyncAddress(
		t,
		"0xaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
	)

	indexRange, err := storage.NewSwapIndexRange(
		poolAddress,
		1_000,
		2_000,
	)
	if err != nil {
		t.Fatalf(
			"NewSwapIndexRange() error = %v",
			err,
		)
	}

	if indexRange.Status !=
		storage.SwapIndexPending {
		t.Fatalf(
			"Status = %s, want pending",
			indexRange.Status,
		)
	}

	if indexRange.NextBlock != 1_000 {
		t.Fatalf(
			"NextBlock = %d, want 1000",
			indexRange.NextBlock,
		)
	}

	if indexRange.Started() {
		t.Fatal(
			"Started() = true, want false",
		)
	}

	if indexRange.Complete() {
		t.Fatal(
			"Complete() = true, want false",
		)
	}

	if indexRange.RemainingBlocks() != 1_001 {
		t.Fatalf(
			"RemainingBlocks() = %d, want 1001",
			indexRange.RemainingBlocks(),
		)
	}
}

func TestSwapIndexRangeCompletedState(
	t *testing.T,
) {
	t.Parallel()

	poolAddress := mustSwapSyncAddress(
		t,
		"0xaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
	)

	blockHash := mustSwapSyncHash(
		t,
		fmt.Sprintf(
			"0x%064x",
			2_000,
		),
	)

	indexRange := storage.SwapIndexRange{
		PoolAddress: poolAddress,

		FromBlock: 1_000,
		ToBlock:   2_000,
		NextBlock: 2_001,

		Status: storage.SwapIndexComplete,

		LastProcessedBlockHash: blockHash,
	}

	if err := indexRange.Validate(); err != nil {
		t.Fatalf(
			"Validate() error = %v",
			err,
		)
	}

	if !indexRange.Started() {
		t.Fatal(
			"Started() = false, want true",
		)
	}

	if !indexRange.Complete() {
		t.Fatal(
			"Complete() = false, want true",
		)
	}

	if indexRange.RemainingBlocks() != 0 {
		t.Fatalf(
			"RemainingBlocks() = %d, want 0",
			indexRange.RemainingBlocks(),
		)
	}

	lastBlock, exists :=
		indexRange.LastProcessedBlock()

	if !exists {
		t.Fatal(
			"LastProcessedBlock() expected value",
		)
	}

	if lastBlock != 2_000 {
		t.Fatalf(
			"LastProcessedBlock() = %d, want 2000",
			lastBlock,
		)
	}
}

func TestSwapIndexRangeRejectsCompletedStatusBeforeEnd(
	t *testing.T,
) {
	t.Parallel()

	poolAddress := mustSwapSyncAddress(
		t,
		"0xaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
	)

	blockHash := mustSwapSyncHash(
		t,
		fmt.Sprintf(
			"0x%064x",
			1_499,
		),
	)

	indexRange := storage.SwapIndexRange{
		PoolAddress: poolAddress,

		FromBlock: 1_000,
		ToBlock:   2_000,
		NextBlock: 1_500,

		Status: storage.SwapIndexComplete,

		LastProcessedBlockHash: blockHash,
	}

	if err := indexRange.Validate(); err == nil {
		t.Fatal(
			"Validate() expected incomplete-range error",
		)
	}
}

func TestSwapIndexRangeRejectsMissingProcessedHash(
	t *testing.T,
) {
	t.Parallel()

	poolAddress := mustSwapSyncAddress(
		t,
		"0xaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
	)

	indexRange := storage.SwapIndexRange{
		PoolAddress: poolAddress,

		FromBlock: 1_000,
		ToBlock:   2_000,
		NextBlock: 1_500,

		Status: storage.SwapIndexRunning,
	}

	if err := indexRange.Validate(); err == nil {
		t.Fatal(
			"Validate() expected missing block-hash error",
		)
	}
}

func mustSwapSyncAddress(
	t *testing.T,
	value string,
) domain.Address {
	t.Helper()

	address, err := domain.ParseAddress(value)
	if err != nil {
		t.Fatalf(
			"ParseAddress(%q) error = %v",
			value,
			err,
		)
	}

	return address
}

func mustSwapSyncHash(
	t *testing.T,
	value string,
) domain.Hash {
	t.Helper()

	hash, err := domain.ParseHash(value)
	if err != nil {
		t.Fatalf(
			"ParseHash(%q) error = %v",
			value,
			err,
		)
	}

	return hash
}
