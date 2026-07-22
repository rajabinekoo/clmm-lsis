package storage_test

import (
	"errors"
	"testing"
	"time"

	"github.com/rajabinekoo/clmm-lsis/internal/domain"
	"github.com/rajabinekoo/clmm-lsis/internal/storage"
)

func TestSwapBatchCommitValidate(
	t *testing.T,
) {
	t.Parallel()

	pool := newStorageTestPool(t)

	blockHash100 := storageHashString(100)
	blockHash101 := storageHashString(101)

	commit := storage.SwapBatchCommit{
		RangeKey: storage.SwapIndexRangeKey{
			PoolAddress: pool.Address,
			FromBlock:   100,
			ToBlock:     200,
		},

		ExpectedNextBlock:     100,
		ProcessedThroughBlock: 101,

		ProcessedThroughBlockHash: mustSwapBatchHash(
			t,
			blockHash101,
		),

		Swaps: []storage.SwapRecord{
			newSwapBatchRecord(
				t,
				pool.Address.String(),
				100,
				blockHash100,
				1,
				10,
				-10,
			),
			newSwapBatchRecord(
				t,
				pool.Address.String(),
				101,
				blockHash101,
				0,
				2,
				-20,
			),
		},
	}

	if err := commit.Validate(); err != nil {
		t.Fatalf(
			"Validate() error = %v",
			err,
		)
	}
}

func TestSwapBatchCommitAllowsBlocksWithoutSwaps(
	t *testing.T,
) {
	t.Parallel()

	pool := newStorageTestPool(t)

	commit := storage.SwapBatchCommit{
		RangeKey: storage.SwapIndexRangeKey{
			PoolAddress: pool.Address,
			FromBlock:   100,
			ToBlock:     200,
		},

		ExpectedNextBlock:     100,
		ProcessedThroughBlock: 150,

		ProcessedThroughBlockHash: mustSwapBatchHash(
			t,
			storageHashString(150),
		),

		Swaps: nil,
	}

	if err := commit.Validate(); err != nil {
		t.Fatalf(
			"Validate() error = %v",
			err,
		)
	}
}

func TestSwapBatchCommitRejectsOutOfOrderSwaps(
	t *testing.T,
) {
	t.Parallel()

	pool := newStorageTestPool(t)

	blockHash := storageHashString(100)

	commit := storage.SwapBatchCommit{
		RangeKey: storage.SwapIndexRangeKey{
			PoolAddress: pool.Address,
			FromBlock:   100,
			ToBlock:     200,
		},

		ExpectedNextBlock:     100,
		ProcessedThroughBlock: 100,

		ProcessedThroughBlockHash: mustSwapBatchHash(
			t,
			blockHash,
		),

		Swaps: []storage.SwapRecord{
			newSwapBatchRecord(
				t,
				pool.Address.String(),
				100,
				blockHash,
				0,
				10,
				-10,
			),
			newSwapBatchRecord(
				t,
				pool.Address.String(),
				100,
				blockHash,
				0,
				9,
				-9,
			),
		},
	}

	err := commit.Validate()

	if !errors.Is(
		err,
		storage.ErrInvalidSwapBatch,
	) {
		t.Fatalf(
			"Validate() error = %v, want ErrInvalidSwapBatch",
			err,
		)
	}
}

func TestSwapBatchCommitRejectsFinalBlockHashMismatch(
	t *testing.T,
) {
	t.Parallel()

	pool := newStorageTestPool(t)

	commit := storage.SwapBatchCommit{
		RangeKey: storage.SwapIndexRangeKey{
			PoolAddress: pool.Address,
			FromBlock:   100,
			ToBlock:     200,
		},

		ExpectedNextBlock:     100,
		ProcessedThroughBlock: 100,

		ProcessedThroughBlockHash: mustSwapBatchHash(
			t,
			storageHashString(999),
		),

		Swaps: []storage.SwapRecord{
			newSwapBatchRecord(
				t,
				pool.Address.String(),
				100,
				storageHashString(100),
				0,
				1,
				-10,
			),
		},
	}

	err := commit.Validate()

	if !errors.Is(
		err,
		storage.ErrInvalidSwapBatch,
	) {
		t.Fatalf(
			"Validate() error = %v, want ErrInvalidSwapBatch",
			err,
		)
	}
}

func TestSwapRecordEquivalentIgnoresNumericFormatting(
	t *testing.T,
) {
	t.Parallel()

	pool := newStorageTestPool(t)

	left := newSwapBatchRecord(
		t,
		pool.Address.String(),
		100,
		storageHashString(100),
		0,
		1,
		-10,
	)

	right := left

	right.Amount0Raw = "0001000"
	right.Amount1Raw = "-000900"
	right.SqrtPriceX96 =
		"0" + right.SqrtPriceX96

	equivalent, err :=
		left.Equivalent(right)
	if err != nil {
		t.Fatalf(
			"Equivalent() error = %v",
			err,
		)
	}

	if !equivalent {
		t.Fatal(
			"Equivalent() = false, want true",
		)
	}
}

func TestSwapRecordEquivalentDetectsDifferentLiquidity(
	t *testing.T,
) {
	t.Parallel()

	pool := newStorageTestPool(t)

	left := newSwapBatchRecord(
		t,
		pool.Address.String(),
		100,
		storageHashString(100),
		0,
		1,
		-10,
	)

	right := left
	right.ActiveLiquidity = "999"

	equivalent, err :=
		left.Equivalent(right)
	if err != nil {
		t.Fatalf(
			"Equivalent() error = %v",
			err,
		)
	}

	if equivalent {
		t.Fatal(
			"Equivalent() = true, want false",
		)
	}
}

func newSwapBatchRecord(
	t *testing.T,
	poolAddress string,
	blockNumber uint64,
	blockHash string,
	transactionIndex uint32,
	logIndex uint32,
	tick int32,
) storage.SwapRecord {
	t.Helper()

	return storage.SwapRecord{
		PoolAddress: poolAddress,

		BlockNumber: blockNumber,
		BlockHash:   blockHash,

		TransactionHash: storageHashString(
			blockNumber*1_000 +
				uint64(logIndex) + 1,
		),

		TransactionIndex: transactionIndex,

		LogIndex: logIndex,

		Timestamp: time.Unix(
			int64(blockNumber),
			0,
		).UTC(),

		Sender: storageAddressString(10),

		Recipient: storageAddressString(11),

		Amount0Raw: "1000",
		Amount1Raw: "-900",

		SqrtPriceX96: mustStorageSqrtPrice(
			t,
			tick,
		).String(),

		ActiveLiquidity: "1000",

		Tick: tick,
	}
}

func mustSwapBatchHash(
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
