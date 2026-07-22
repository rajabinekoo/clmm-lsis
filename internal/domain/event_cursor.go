package domain

import "fmt"

// EventCursor identifies an exact Ethereum log position.
//
// Block number alone is insufficient for event-level reconstruction because
// multiple swaps and liquidity actions may occur in the same block.
type EventCursor struct {
	BlockNumber      uint64
	TransactionIndex uint32
	LogIndex         uint32
}

func (c EventCursor) Validate() error {
	if c.BlockNumber == 0 {
		return fmt.Errorf("event cursor block number must be greater than zero")
	}

	return nil
}

// Compare returns:
//
//   - -1 when c occurs before other;
//   - 0 when both cursors refer to the same log;
//   - 1 when c occurs after other.
func (c EventCursor) Compare(other EventCursor) int {
	switch {
	case c.BlockNumber < other.BlockNumber:
		return -1
	case c.BlockNumber > other.BlockNumber:
		return 1
	case c.TransactionIndex < other.TransactionIndex:
		return -1
	case c.TransactionIndex > other.TransactionIndex:
		return 1
	case c.LogIndex < other.LogIndex:
		return -1
	case c.LogIndex > other.LogIndex:
		return 1
	default:
		return 0
	}
}

func (c EventCursor) Before(other EventCursor) bool {
	return c.Compare(other) < 0
}

func (c EventCursor) After(other EventCursor) bool {
	return c.Compare(other) > 0
}

func (c EventCursor) String() string {
	return fmt.Sprintf(
		"%d:%d:%d",
		c.BlockNumber,
		c.TransactionIndex,
		c.LogIndex,
	)
}
