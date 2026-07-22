package reconstruction

import (
	"fmt"

	"github.com/rajabinekoo/clmm-lsis/internal/domain"
)

// SnapshotAfterLastEvent returns the immutable state immediately after the
// latest successfully applied event.
func (s *MutablePoolState) SnapshotAfterLastEvent() (
	domain.PoolSnapshot,
	error,
) {
	if !s.initialized {
		return domain.PoolSnapshot{}, ErrNotInitialized
	}

	if s.lastAppliedCursor == nil {
		return domain.PoolSnapshot{}, fmt.Errorf(
			"%w: no event has been applied",
			ErrInvalidSnapshotReference,
		)
	}

	reference, err :=
		domain.NewEventSnapshotReference(
			*s.lastAppliedCursor,
			domain.SnapshotAfterEvent,
		)
	if err != nil {
		return domain.PoolSnapshot{}, err
	}

	return s.buildSnapshot(reference)
}

// SnapshotBeforeEvent captures the current state as the exact pre-state of a
// future event.
//
// The supplied cursor must be a valid next event according to reconstruction
// order.
func (s *MutablePoolState) SnapshotBeforeEvent(
	cursor domain.EventCursor,
) (
	domain.PoolSnapshot,
	error,
) {
	if !s.initialized {
		return domain.PoolSnapshot{}, ErrNotInitialized
	}

	if err := s.validateEventOrder(cursor); err != nil {
		return domain.PoolSnapshot{}, fmt.Errorf(
			"%w: %v",
			ErrInvalidSnapshotReference,
			err,
		)
	}

	reference, err :=
		domain.NewEventSnapshotReference(
			cursor,
			domain.SnapshotBeforeEvent,
		)
	if err != nil {
		return domain.PoolSnapshot{}, err
	}

	return s.buildSnapshot(reference)
}

// SnapshotAtBlockEnd creates a block-end snapshot.
//
// The caller must ensure that all relevant pool events from that block have
// already been applied. The reconstruction state can validate chronology but
// cannot independently know whether an external event source was complete.
func (s *MutablePoolState) SnapshotAtBlockEnd(
	blockNumber uint64,
) (
	domain.PoolSnapshot,
	error,
) {
	if !s.initialized {
		return domain.PoolSnapshot{}, ErrNotInitialized
	}

	if err := s.validateBlockEndReference(
		blockNumber,
	); err != nil {
		return domain.PoolSnapshot{}, err
	}

	reference, err :=
		domain.NewBlockEndSnapshotReference(
			blockNumber,
		)
	if err != nil {
		return domain.PoolSnapshot{}, err
	}

	return s.buildSnapshot(reference)
}

func (s *MutablePoolState) buildSnapshot(
	reference domain.SnapshotReference,
) (
	domain.PoolSnapshot,
	error,
) {
	ticks := make(
		[]domain.TickState,
		0,
		len(s.tickIndexes),
	)

	for _, index := range s.tickIndexes {
		ticks = append(
			ticks,
			s.ticks[index],
		)
	}

	positions := make(
		[]domain.CorePosition,
		0,
		len(s.positions),
	)

	for _, position := range s.positions {
		positions = append(
			positions,
			position,
		)
	}

	snapshot, err := domain.NewPoolSnapshot(
		s.pool.Address,
		reference,
		s.sqrtPriceX96,
		s.currentTick,
		s.activeLiquidity,
		ticks,
		positions,
	)
	if err != nil {
		return domain.PoolSnapshot{}, fmt.Errorf(
			"build reconstructed snapshot: %w",
			err,
		)
	}

	return snapshot, nil
}

func (s *MutablePoolState) validateBlockEndReference(
	blockNumber uint64,
) error {
	if blockNumber == 0 {
		return fmt.Errorf(
			"%w: block number must be greater than zero",
			ErrInvalidSnapshotReference,
		)
	}

	if s.lastAppliedCursor != nil {
		if blockNumber <
			s.lastAppliedCursor.BlockNumber {
			return fmt.Errorf(
				"%w: block %d is before last applied event block %d",
				ErrInvalidSnapshotReference,
				blockNumber,
				s.lastAppliedCursor.BlockNumber,
			)
		}

		return nil
	}

	if !s.hasBaseReference {
		return nil
	}

	switch s.baseReference.Boundary() {
	case domain.SnapshotBlockEnd:
		if blockNumber <
			s.baseReference.BlockNumber() {
			return fmt.Errorf(
				"%w: block %d is before base block-end snapshot %d",
				ErrInvalidSnapshotReference,
				blockNumber,
				s.baseReference.BlockNumber(),
			)
		}

	case domain.SnapshotAfterEvent:
		if blockNumber <
			s.baseReference.BlockNumber() {
			return fmt.Errorf(
				"%w: block %d is before base event block %d",
				ErrInvalidSnapshotReference,
				blockNumber,
				s.baseReference.BlockNumber(),
			)
		}

	case domain.SnapshotBeforeEvent:
		// A state captured before an event cannot be considered block-complete
		// for that same block unless the event has subsequently been applied.
		if blockNumber <=
			s.baseReference.BlockNumber() {
			return fmt.Errorf(
				"%w: block-end reference must be after before-event base block %d",
				ErrInvalidSnapshotReference,
				s.baseReference.BlockNumber(),
			)
		}

	default:
		return fmt.Errorf(
			"%w: unsupported base boundary %s",
			ErrInvalidSnapshotReference,
			s.baseReference.Boundary(),
		)
	}

	return nil
}
