package storage

import (
	"fmt"

	"github.com/rajabinekoo/clmm-lsis/internal/domain"
)

// SwapBatchCommit describes one atomic indexing progress update.
//
// The batch covers every block from ExpectedNextBlock through
// ProcessedThroughBlock, inclusively. Some or all of those blocks may contain
// no swaps.
//
// ProcessedThroughBlockHash is always required, even when the final processed
// block has no Swap event. It is used to detect chain reorganizations before
// the next indexing batch starts.
type SwapBatchCommit struct {
	RangeKey SwapIndexRangeKey

	ExpectedNextBlock     uint64
	ProcessedThroughBlock uint64

	ProcessedThroughBlockHash domain.Hash

	Swaps []SwapRecord
}

func (c SwapBatchCommit) Validate() error {
	if err := c.RangeKey.Validate(); err != nil {
		return fmt.Errorf(
			"%w: range key: %v",
			ErrInvalidSwapBatch,
			err,
		)
	}

	if c.ExpectedNextBlock <
		c.RangeKey.FromBlock ||
		c.ExpectedNextBlock >
			c.RangeKey.ToBlock {
		return fmt.Errorf(
			"%w: expected next block %d must be inside [%d,%d]",
			ErrInvalidSwapBatch,
			c.ExpectedNextBlock,
			c.RangeKey.FromBlock,
			c.RangeKey.ToBlock,
		)
	}

	if c.ProcessedThroughBlock <
		c.ExpectedNextBlock {
		return fmt.Errorf(
			"%w: processed-through block %d is before expected next block %d",
			ErrInvalidSwapBatch,
			c.ProcessedThroughBlock,
			c.ExpectedNextBlock,
		)
	}

	if c.ProcessedThroughBlock >
		c.RangeKey.ToBlock {
		return fmt.Errorf(
			"%w: processed-through block %d exceeds range end %d",
			ErrInvalidSwapBatch,
			c.ProcessedThroughBlock,
			c.RangeKey.ToBlock,
		)
	}

	if c.ProcessedThroughBlockHash.IsZero() {
		return fmt.Errorf(
			"%w: processed-through block hash is required",
			ErrInvalidSwapBatch,
		)
	}

	var previousCursor *domain.EventCursor

	for index, record := range c.Swaps {
		if record.Timestamp.IsZero() {
			return fmt.Errorf(
				"%w: swap %d timestamp is required",
				ErrInvalidSwapBatch,
				index,
			)
		}

		event, err := record.DomainEvent()
		if err != nil {
			return fmt.Errorf(
				"%w: swap %d: %v",
				ErrInvalidSwapBatch,
				index,
				err,
			)
		}

		if event.PoolAddress() !=
			c.RangeKey.PoolAddress {
			return fmt.Errorf(
				"%w: swap %d pool %s does not match range pool %s",
				ErrInvalidSwapBatch,
				index,
				event.PoolAddress(),
				c.RangeKey.PoolAddress,
			)
		}

		cursor := event.Cursor()

		if cursor.BlockNumber <
			c.ExpectedNextBlock ||
			cursor.BlockNumber >
				c.ProcessedThroughBlock {
			return fmt.Errorf(
				"%w: swap %d cursor %s is outside processed interval [%d,%d]",
				ErrInvalidSwapBatch,
				index,
				cursor,
				c.ExpectedNextBlock,
				c.ProcessedThroughBlock,
			)
		}

		if previousCursor != nil &&
			cursor.Compare(*previousCursor) <= 0 {
			return fmt.Errorf(
				"%w: swaps are not strictly ordered at cursor %s",
				ErrInvalidSwapBatch,
				cursor,
			)
		}

		if cursor.BlockNumber ==
			c.ProcessedThroughBlock &&
			event.BlockHash() !=
				c.ProcessedThroughBlockHash {
			return fmt.Errorf(
				"%w: final block swap hash %s does not match processed-through hash %s",
				ErrInvalidSwapBatch,
				event.BlockHash(),
				c.ProcessedThroughBlockHash,
			)
		}

		copiedCursor := cursor
		previousCursor = &copiedCursor
	}

	return nil
}

// CopySwaps returns an independent slice.
//
// SwapRecord currently contains immutable value fields and strings, so a
// shallow element copy is sufficient.
func (c SwapBatchCommit) CopySwaps() []SwapRecord {
	return append(
		[]SwapRecord(nil),
		c.Swaps...,
	)
}
