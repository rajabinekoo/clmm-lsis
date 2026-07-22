package storage

import (
	"fmt"
	"math"
	"time"

	"github.com/rajabinekoo/clmm-lsis/internal/domain"
)

// SwapIndexStatus represents the lifecycle of one requested indexing range.
type SwapIndexStatus string

const (
	SwapIndexPending  SwapIndexStatus = "pending"
	SwapIndexRunning  SwapIndexStatus = "running"
	SwapIndexComplete SwapIndexStatus = "complete"
	SwapIndexFailed   SwapIndexStatus = "failed"
)

func (s SwapIndexStatus) Validate() error {
	switch s {
	case SwapIndexPending,
		SwapIndexRunning,
		SwapIndexComplete,
		SwapIndexFailed:
		return nil

	default:
		return fmt.Errorf(
			"unsupported swap index status %q",
			s,
		)
	}
}

// SwapIndexRange records progress for one inclusive block interval.
//
// NextBlock is the first block that has not yet been indexed. Therefore:
//
//	pending:  NextBlock == FromBlock
//	complete: NextBlock == ToBlock + 1
//
// The range model allows sparse empirical windows without pretending that
// every block between two distant withdrawal events has been indexed.
type SwapIndexRange struct {
	PoolAddress domain.Address

	FromBlock uint64
	ToBlock   uint64
	NextBlock uint64

	Status SwapIndexStatus

	LastProcessedBlockHash domain.Hash
	LastError              string

	CreatedAt time.Time
	UpdatedAt time.Time
}

func NewSwapIndexRange(
	poolAddress domain.Address,
	fromBlock uint64,
	toBlock uint64,
) (SwapIndexRange, error) {
	indexRange := SwapIndexRange{
		PoolAddress: poolAddress,

		FromBlock: fromBlock,
		ToBlock:   toBlock,
		NextBlock: fromBlock,

		Status: SwapIndexPending,
	}

	if err := indexRange.Validate(); err != nil {
		return SwapIndexRange{}, err
	}

	return indexRange, nil
}

func (r SwapIndexRange) Validate() error {
	if r.PoolAddress.IsZero() {
		return fmt.Errorf(
			"swap index range pool address is required",
		)
	}

	if r.FromBlock == 0 {
		return fmt.Errorf(
			"swap index range from block must be greater than zero",
		)
	}

	if r.FromBlock > r.ToBlock {
		return fmt.Errorf(
			"swap index range from block %d exceeds to block %d",
			r.FromBlock,
			r.ToBlock,
		)
	}

	// PostgreSQL BIGINT is signed. Restrict all persisted block numbers to the
	// same representable domain.
	if r.ToBlock >= uint64(math.MaxInt64) {
		return fmt.Errorf(
			"swap index range to block %d exceeds supported PostgreSQL BIGINT range",
			r.ToBlock,
		)
	}

	maximumNextBlock := r.ToBlock + 1

	if r.NextBlock < r.FromBlock ||
		r.NextBlock > maximumNextBlock {
		return fmt.Errorf(
			"swap index range next block %d must be inside [%d,%d]",
			r.NextBlock,
			r.FromBlock,
			maximumNextBlock,
		)
	}

	if err := r.Status.Validate(); err != nil {
		return err
	}

	if r.NextBlock == r.FromBlock &&
		!r.LastProcessedBlockHash.IsZero() {
		return fmt.Errorf(
			"unstarted swap index range must not have a processed block hash",
		)
	}

	if r.NextBlock > r.FromBlock &&
		r.LastProcessedBlockHash.IsZero() {
		return fmt.Errorf(
			"started swap index range requires the last processed block hash",
		)
	}

	if r.Status == SwapIndexPending &&
		r.NextBlock != r.FromBlock {
		return fmt.Errorf(
			"pending swap index range must start at block %d",
			r.FromBlock,
		)
	}

	if r.Status == SwapIndexComplete &&
		r.NextBlock != maximumNextBlock {
		return fmt.Errorf(
			"complete swap index range must have next block %d",
			maximumNextBlock,
		)
	}

	if r.NextBlock == maximumNextBlock &&
		r.Status != SwapIndexComplete {
		return fmt.Errorf(
			"fully processed swap index range must have complete status",
		)
	}

	return nil
}

func (r SwapIndexRange) Started() bool {
	return r.NextBlock > r.FromBlock
}

func (r SwapIndexRange) Complete() bool {
	return r.Status == SwapIndexComplete &&
		r.NextBlock == r.ToBlock+1
}

func (r SwapIndexRange) RemainingBlocks() uint64 {
	if r.Complete() {
		return 0
	}

	return r.ToBlock - r.NextBlock + 1
}

// LastProcessedBlock returns the latest successfully committed block.
func (r SwapIndexRange) LastProcessedBlock() (
	uint64,
	bool,
) {
	if !r.Started() {
		return 0, false
	}

	return r.NextBlock - 1, true
}
