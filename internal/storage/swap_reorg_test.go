package storage_test

import (
	"errors"
	"testing"

	"github.com/rajabinekoo/clmm-lsis/internal/storage"
)

func TestSwapIndexRangeVerifyLastProcessedBlockHash(
	t *testing.T,
) {
	t.Parallel()

	pool := newStorageTestPool(t)

	storedHash := mustSwapBatchHash(
		t,
		storageHashString(150),
	)

	indexRange := storage.SwapIndexRange{
		PoolAddress: pool.Address,

		FromBlock: 100,
		ToBlock:   200,
		NextBlock: 151,

		Status: storage.SwapIndexRunning,

		LastProcessedBlockHash: storedHash,
	}

	if err := indexRange.Validate(); err != nil {
		t.Fatalf(
			"Validate() error = %v",
			err,
		)
	}

	if err :=
		indexRange.VerifyLastProcessedBlockHash(
			storedHash,
		); err != nil {
		t.Fatalf(
			"VerifyLastProcessedBlockHash() error = %v",
			err,
		)
	}
}

func TestSwapIndexRangeDetectsChainReorganization(
	t *testing.T,
) {
	t.Parallel()

	pool := newStorageTestPool(t)

	indexRange := storage.SwapIndexRange{
		PoolAddress: pool.Address,

		FromBlock: 100,
		ToBlock:   200,
		NextBlock: 151,

		Status: storage.SwapIndexRunning,

		LastProcessedBlockHash: mustSwapBatchHash(
			t,
			storageHashString(150),
		),
	}

	err :=
		indexRange.VerifyLastProcessedBlockHash(
			mustSwapBatchHash(
				t,
				storageHashString(999),
			),
		)

	if !errors.Is(
		err,
		storage.ErrChainReorganization,
	) {
		t.Fatalf(
			"VerifyLastProcessedBlockHash() error = %v, want ErrChainReorganization",
			err,
		)
	}
}
