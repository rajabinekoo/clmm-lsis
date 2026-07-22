package domain

import "fmt"

// EventCursor identifies one exact Ethereum log position.
//
// Ethereum log indexes are globally ordered within a block. TransactionIndex
// is retained as provenance metadata, but event chronology is determined by:
//
//	block number
//	log index
//
// This design allows legacy liquidity events that do not store transaction
// indexes to be merged safely with newly indexed swap events.
type EventCursor struct {
	BlockNumber      uint64
	TransactionIndex uint32
	LogIndex         uint32
}

func (c EventCursor) Validate() error {
	if c.BlockNumber == 0 {
		return fmt.Errorf(
			"event cursor block number must be greater than zero",
		)
	}

	return nil
}

// Compare returns:
//
//   - -1 when c occurs before other;
//   - 0 when both cursors refer to the same Ethereum log;
//   - 1 when c occurs after other.
//
// TransactionIndex is intentionally excluded from ordering because LogIndex is
// already globally ordered across the entire block.
func (c EventCursor) Compare(
	other EventCursor,
) int {
	switch {
	case c.BlockNumber < other.BlockNumber:
		return -1

	case c.BlockNumber > other.BlockNumber:
		return 1

	case c.LogIndex < other.LogIndex:
		return -1

	case c.LogIndex > other.LogIndex:
		return 1

	default:
		return 0
	}
}

func (c EventCursor) Before(
	other EventCursor,
) bool {
	return c.Compare(other) < 0
}

func (c EventCursor) After(
	other EventCursor,
) bool {
	return c.Compare(other) > 0
}

// SameLog reports whether two cursors identify the same block-level log.
//
// TransactionIndex differences are ignored because a block-level log index
// uniquely identifies one log.
func (c EventCursor) SameLog(
	other EventCursor,
) bool {
	return c.Compare(other) == 0
}

func (c EventCursor) String() string {
	return fmt.Sprintf(
		"%d:%d:%d",
		c.BlockNumber,
		c.TransactionIndex,
		c.LogIndex,
	)
}
