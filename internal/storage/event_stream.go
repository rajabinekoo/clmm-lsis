package storage

import (
	"fmt"
	"sort"

	"github.com/rajabinekoo/clmm-lsis/internal/domain"
)

// BuildOrderedEventStream converts and merges liquidity actions and swaps into
// one deterministic pool-event stream.
//
// The result is ordered by:
//
//	block_number
//	log_index
//
// Duplicate cursors are rejected because one Ethereum log cannot represent
// two distinct pool events.
func BuildOrderedEventStream(
	liquidityActions []LegacyLPActionRecord,
	swaps []SwapRecord,
) ([]domain.PoolEvent, error) {
	events := make(
		[]domain.PoolEvent,
		0,
		len(liquidityActions)+len(swaps),
	)

	for _, record := range liquidityActions {
		event, err := record.DomainEvent()
		if err != nil {
			return nil, fmt.Errorf(
				"build event stream from lp action %s: %w",
				record.ID,
				err,
			)
		}

		events = append(
			events,
			event,
		)
	}

	for _, record := range swaps {
		event, err := record.DomainEvent()
		if err != nil {
			return nil, fmt.Errorf(
				"build event stream from swap at block %d log %d: %w",
				record.BlockNumber,
				record.LogIndex,
				err,
			)
		}

		events = append(
			events,
			event,
		)
	}

	sort.SliceStable(
		events,
		func(i, j int) bool {
			return events[i].Cursor().Compare(
				events[j].Cursor(),
			) < 0
		},
	)

	for index := 1; index < len(events); index++ {
		previous := events[index-1]
		current := events[index]

		if previous.Cursor().SameLog(
			current.Cursor(),
		) {
			return nil, fmt.Errorf(
				"%w: pool=%s cursor=%s previous_type=%s current_type=%s",
				ErrDuplicateEventCursor,
				current.PoolAddress(),
				current.Cursor(),
				previous.Type(),
				current.Type(),
			)
		}

		if previous.PoolAddress() !=
			current.PoolAddress() {
			return nil, fmt.Errorf(
				"%w: event stream contains multiple pools: %s and %s",
				ErrInvalidLegacyRecord,
				previous.PoolAddress(),
				current.PoolAddress(),
			)
		}
	}

	return events, nil
}
