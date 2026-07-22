package storage

import (
	"fmt"
	"math"

	"github.com/rajabinekoo/clmm-lsis/internal/domain"
)

// SwapIndexRangeKey uniquely identifies one inclusive indexing interval.
//
// Separate ranges allow the empirical study to index sparse windows around
// selected withdrawal events without downloading every intermediate block.
type SwapIndexRangeKey struct {
	PoolAddress domain.Address

	FromBlock uint64
	ToBlock   uint64
}

func NewSwapIndexRangeKey(
	poolAddress domain.Address,
	fromBlock uint64,
	toBlock uint64,
) (SwapIndexRangeKey, error) {
	key := SwapIndexRangeKey{
		PoolAddress: poolAddress,
		FromBlock:   fromBlock,
		ToBlock:     toBlock,
	}

	if err := key.Validate(); err != nil {
		return SwapIndexRangeKey{}, err
	}

	return key, nil
}

func (k SwapIndexRangeKey) Validate() error {
	if k.PoolAddress.IsZero() {
		return fmt.Errorf(
			"swap index range key pool address is required",
		)
	}

	if k.FromBlock == 0 {
		return fmt.Errorf(
			"swap index range key from block must be greater than zero",
		)
	}

	if k.FromBlock > k.ToBlock {
		return fmt.Errorf(
			"swap index range key from block %d exceeds to block %d",
			k.FromBlock,
			k.ToBlock,
		)
	}

	// pool_swap_index_ranges uses PostgreSQL BIGINT.
	if k.ToBlock >= uint64(math.MaxInt64) {
		return fmt.Errorf(
			"swap index range key to block %d exceeds PostgreSQL BIGINT range",
			k.ToBlock,
		)
	}

	return nil
}

func (k SwapIndexRangeKey) String() string {
	return fmt.Sprintf(
		"%s:%d-%d",
		k.PoolAddress,
		k.FromBlock,
		k.ToBlock,
	)
}

// Key returns the persistent identity of this indexing range.
func (r SwapIndexRange) Key() SwapIndexRangeKey {
	return SwapIndexRangeKey{
		PoolAddress: r.PoolAddress,
		FromBlock:   r.FromBlock,
		ToBlock:     r.ToBlock,
	}
}

// VerifyLastProcessedBlockHash checks that the previously committed block is
// still part of the canonical chain.
//
// The indexer calls this before resuming a started range. A mismatch indicates
// that the chain reorganized after the previous batch was committed.
func (r SwapIndexRange) VerifyLastProcessedBlockHash(
	canonicalHash domain.Hash,
) error {
	if err := r.Validate(); err != nil {
		return err
	}

	if !r.Started() {
		return nil
	}

	if canonicalHash.IsZero() {
		return fmt.Errorf(
			"canonical last-processed block hash is required",
		)
	}

	if canonicalHash != r.LastProcessedBlockHash {
		lastBlock, _ := r.LastProcessedBlock()

		return fmt.Errorf(
			"%w: pool=%s block=%d stored_hash=%s canonical_hash=%s",
			ErrChainReorganization,
			r.PoolAddress,
			lastBlock,
			r.LastProcessedBlockHash,
			canonicalHash,
		)
	}

	return nil
}
