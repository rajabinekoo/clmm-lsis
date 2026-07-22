package postgres

import (
	"errors"
	"fmt"
	"testing"

	"github.com/rajabinekoo/clmm-lsis/internal/domain"
	"github.com/rajabinekoo/clmm-lsis/internal/storage"
)

func TestAdvanceSwapIndexRange(
	t *testing.T,
) {
	t.Parallel()

	poolAddress :=
		mustPostgresSwapAddress(t)

	current, err :=
		storage.NewSwapIndexRange(
			poolAddress,
			100,
			200,
		)
	if err != nil {
		t.Fatalf(
			"NewSwapIndexRange() error = %v",
			err,
		)
	}

	commit := storage.SwapBatchCommit{
		RangeKey: current.Key(),

		ExpectedNextBlock:     100,
		ProcessedThroughBlock: 149,

		ProcessedThroughBlockHash: mustPostgresSwapHash(
			t,
			149,
		),
	}

	next, err := advanceSwapIndexRange(
		current,
		commit,
	)
	if err != nil {
		t.Fatalf(
			"advanceSwapIndexRange() error = %v",
			err,
		)
	}

	if next.NextBlock != 150 {
		t.Fatalf(
			"NextBlock = %d, want 150",
			next.NextBlock,
		)
	}

	if next.Status !=
		storage.SwapIndexRunning {
		t.Fatalf(
			"Status = %s, want running",
			next.Status,
		)
	}

	if next.LastError != "" {
		t.Fatalf(
			"LastError = %q, want empty",
			next.LastError,
		)
	}
}

func TestAdvanceSwapIndexRangeCompletesRange(
	t *testing.T,
) {
	t.Parallel()

	poolAddress :=
		mustPostgresSwapAddress(t)

	current := storage.SwapIndexRange{
		PoolAddress: poolAddress,

		FromBlock: 100,
		ToBlock:   200,
		NextBlock: 150,

		Status: storage.SwapIndexRunning,

		LastProcessedBlockHash: mustPostgresSwapHash(
			t,
			149,
		),
	}

	commit := storage.SwapBatchCommit{
		RangeKey: current.Key(),

		ExpectedNextBlock:     150,
		ProcessedThroughBlock: 200,

		ProcessedThroughBlockHash: mustPostgresSwapHash(
			t,
			200,
		),
	}

	next, err := advanceSwapIndexRange(
		current,
		commit,
	)
	if err != nil {
		t.Fatalf(
			"advanceSwapIndexRange() error = %v",
			err,
		)
	}

	if !next.Complete() {
		t.Fatalf(
			"Complete() = false; status=%s next=%d",
			next.Status,
			next.NextBlock,
		)
	}

	if next.NextBlock != 201 {
		t.Fatalf(
			"NextBlock = %d, want 201",
			next.NextBlock,
		)
	}
}

func TestAdvanceSwapIndexRangeRejectsStaleWriter(
	t *testing.T,
) {
	t.Parallel()

	poolAddress :=
		mustPostgresSwapAddress(t)

	current := storage.SwapIndexRange{
		PoolAddress: poolAddress,

		FromBlock: 100,
		ToBlock:   200,
		NextBlock: 150,

		Status: storage.SwapIndexRunning,

		LastProcessedBlockHash: mustPostgresSwapHash(
			t,
			149,
		),
	}

	commit := storage.SwapBatchCommit{
		RangeKey: current.Key(),

		ExpectedNextBlock:     100,
		ProcessedThroughBlock: 120,

		ProcessedThroughBlockHash: mustPostgresSwapHash(
			t,
			120,
		),
	}

	_, err := advanceSwapIndexRange(
		current,
		commit,
	)

	if !errors.Is(
		err,
		storage.ErrSwapIndexProgressConflict,
	) {
		t.Fatalf(
			"advanceSwapIndexRange() error = %v, want ErrSwapIndexProgressConflict",
			err,
		)
	}
}

func TestFailSwapIndexRangePreservesProgress(
	t *testing.T,
) {
	t.Parallel()

	poolAddress :=
		mustPostgresSwapAddress(t)

	current := storage.SwapIndexRange{
		PoolAddress: poolAddress,

		FromBlock: 100,
		ToBlock:   200,
		NextBlock: 150,

		Status: storage.SwapIndexRunning,

		LastProcessedBlockHash: mustPostgresSwapHash(
			t,
			149,
		),
	}

	failed, err := failSwapIndexRange(
		current,
		150,
		"temporary RPC failure",
	)
	if err != nil {
		t.Fatalf(
			"failSwapIndexRange() error = %v",
			err,
		)
	}

	if failed.Status !=
		storage.SwapIndexFailed {
		t.Fatalf(
			"Status = %s, want failed",
			failed.Status,
		)
	}

	if failed.NextBlock != 150 {
		t.Fatalf(
			"NextBlock = %d, want 150",
			failed.NextBlock,
		)
	}

	if failed.LastProcessedBlockHash !=
		current.LastProcessedBlockHash {
		t.Fatal(
			"failure transition changed last processed hash",
		)
	}
}

func mustPostgresSwapAddress(
	t *testing.T,
) domain.Address {
	t.Helper()

	address, err := domain.ParseAddress(
		"0xaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
	)
	if err != nil {
		t.Fatalf(
			"ParseAddress() error = %v",
			err,
		)
	}

	return address
}

func mustPostgresSwapHash(
	t *testing.T,
	suffix uint64,
) domain.Hash {
	t.Helper()

	hash, err := domain.ParseHash(
		fmt.Sprintf(
			"0x%064x",
			suffix,
		),
	)
	if err != nil {
		t.Fatalf(
			"ParseHash() error = %v",
			err,
		)
	}

	return hash
}
