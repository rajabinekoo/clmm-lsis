package domain

import "fmt"

// SnapshotBoundary defines the exact temporal interpretation of a snapshot.
type SnapshotBoundary uint8

const (
	SnapshotBeforeEvent SnapshotBoundary = iota + 1
	SnapshotAfterEvent
	SnapshotBlockEnd
)

func (b SnapshotBoundary) String() string {
	switch b {
	case SnapshotBeforeEvent:
		return "before_event"
	case SnapshotAfterEvent:
		return "after_event"
	case SnapshotBlockEnd:
		return "block_end"
	default:
		return fmt.Sprintf("unknown(%d)", b)
	}
}

// SnapshotReference identifies the exact historical point represented by a
// reconstructed pool snapshot.
type SnapshotReference struct {
	blockNumber uint64
	boundary    SnapshotBoundary
	cursor      *EventCursor
}

func NewEventSnapshotReference(
	cursor EventCursor,
	boundary SnapshotBoundary,
) (SnapshotReference, error) {
	if boundary != SnapshotBeforeEvent &&
		boundary != SnapshotAfterEvent {
		return SnapshotReference{}, fmt.Errorf(
			"event snapshot boundary must be before_event or after_event",
		)
	}

	reference := SnapshotReference{
		blockNumber: cursor.BlockNumber,
		boundary:    boundary,
		cursor:      &cursor,
	}

	if err := reference.Validate(); err != nil {
		return SnapshotReference{}, err
	}

	return reference, nil
}

func NewBlockEndSnapshotReference(
	blockNumber uint64,
) (SnapshotReference, error) {
	reference := SnapshotReference{
		blockNumber: blockNumber,
		boundary:    SnapshotBlockEnd,
		cursor:      nil,
	}

	if err := reference.Validate(); err != nil {
		return SnapshotReference{}, err
	}

	return reference, nil
}

func (r SnapshotReference) Validate() error {
	if r.blockNumber == 0 {
		return fmt.Errorf(
			"snapshot reference block number must be greater than zero",
		)
	}

	switch r.boundary {
	case SnapshotBeforeEvent, SnapshotAfterEvent:
		if r.cursor == nil {
			return fmt.Errorf(
				"snapshot boundary %s requires an event cursor",
				r.boundary,
			)
		}

		if err := r.cursor.Validate(); err != nil {
			return fmt.Errorf("snapshot event cursor: %w", err)
		}

		if r.cursor.BlockNumber != r.blockNumber {
			return fmt.Errorf(
				"snapshot cursor block %d does not match reference block %d",
				r.cursor.BlockNumber,
				r.blockNumber,
			)
		}

	case SnapshotBlockEnd:
		if r.cursor != nil {
			return fmt.Errorf(
				"block-end snapshot must not contain an event cursor",
			)
		}

	default:
		return fmt.Errorf(
			"unsupported snapshot boundary %d",
			r.boundary,
		)
	}

	return nil
}

func (r SnapshotReference) BlockNumber() uint64 {
	return r.blockNumber
}

func (r SnapshotReference) Boundary() SnapshotBoundary {
	return r.boundary
}

func (r SnapshotReference) Cursor() (
	EventCursor,
	bool,
) {
	if r.cursor == nil {
		return EventCursor{}, false
	}

	return *r.cursor, true
}
