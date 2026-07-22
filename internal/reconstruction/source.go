package reconstruction

import (
	"context"

	"github.com/rajabinekoo/clmm-lsis/internal/domain"
)

// HistoricalSource provides complete reconstruction inputs without exposing
// the underlying persistence schema.
//
// Implementations may combine:
//
//   - legacy pool_snapshots;
//   - legacy lp_actions;
//   - newly indexed pool_swaps;
//   - RPC metadata enrichment.
//
// The reconstruction layer depends only on this interface and remains
// independent of PostgreSQL and Ethereum clients.
type HistoricalSource interface {
	// LoadLatestCheckpoint returns the latest complete reconstructed snapshot
	// whose block is not greater than atOrBeforeBlock.
	//
	// The implementation may derive position and tick geometry from legacy
	// liquidity actions while using pool_snapshots for scalar pool state.
	LoadLatestCheckpoint(
		ctx context.Context,
		pool domain.Pool,
		atOrBeforeBlock uint64,
	) (domain.PoolSnapshot, error)

	// LoadOrderedEvents returns all supported pool events in the inclusive
	// block interval.
	//
	// Events must be strictly ordered by block number and global log index.
	LoadOrderedEvents(
		ctx context.Context,
		poolAddress domain.Address,
		fromBlockInclusive uint64,
		toBlockInclusive uint64,
	) ([]domain.PoolEvent, error)
}
