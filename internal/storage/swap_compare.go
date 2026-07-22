package storage

import (
	"fmt"

	"github.com/rajabinekoo/clmm-lsis/internal/domain"
)

// Validate verifies both persistence metadata and the underlying domain event.
func (r SwapRecord) Validate() error {
	if r.Timestamp.IsZero() {
		return fmt.Errorf(
			"swap timestamp is required",
		)
	}

	if _, err := r.DomainEvent(); err != nil {
		return err
	}

	return nil
}

// Equivalent reports whether two records represent the exact same canonical
// Swap log.
//
// Textual differences such as address casing or decimal leading zeros do not
// create a false conflict because both records are first converted to domain
// values.
func (r SwapRecord) Equivalent(
	other SwapRecord,
) (bool, error) {
	leftEvent, err := r.DomainEvent()
	if err != nil {
		return false, fmt.Errorf(
			"validate left swap: %w",
			err,
		)
	}

	rightEvent, err := other.DomainEvent()
	if err != nil {
		return false, fmt.Errorf(
			"validate right swap: %w",
			err,
		)
	}

	if !r.Timestamp.Equal(other.Timestamp) {
		return false, nil
	}

	if leftEvent.PoolAddress() !=
		rightEvent.PoolAddress() {
		return false, nil
	}

	leftCursor := leftEvent.Cursor()
	rightCursor := rightEvent.Cursor()

	if leftCursor.BlockNumber !=
		rightCursor.BlockNumber ||
		leftCursor.TransactionIndex !=
			rightCursor.TransactionIndex ||
		leftCursor.LogIndex !=
			rightCursor.LogIndex {
		return false, nil
	}

	if leftEvent.BlockHash() !=
		rightEvent.BlockHash() {
		return false, nil
	}

	if leftEvent.TransactionHash() !=
		rightEvent.TransactionHash() {
		return false, nil
	}

	leftSwap, ok :=
		leftEvent.Payload().(domain.SwapEvent)

	if !ok {
		return false, fmt.Errorf(
			"left event payload is %T, want domain.SwapEvent",
			leftEvent.Payload(),
		)
	}

	rightSwap, ok :=
		rightEvent.Payload().(domain.SwapEvent)

	if !ok {
		return false, fmt.Errorf(
			"right event payload is %T, want domain.SwapEvent",
			rightEvent.Payload(),
		)
	}

	if leftSwap.Sender() !=
		rightSwap.Sender() {
		return false, nil
	}

	if leftSwap.Recipient() !=
		rightSwap.Recipient() {
		return false, nil
	}

	if leftSwap.Amount0().Cmp(
		rightSwap.Amount0(),
	) != 0 {
		return false, nil
	}

	if leftSwap.Amount1().Cmp(
		rightSwap.Amount1(),
	) != 0 {
		return false, nil
	}

	if leftSwap.SqrtPriceX96().Cmp(
		rightSwap.SqrtPriceX96(),
	) != 0 {
		return false, nil
	}

	if leftSwap.ActiveLiquidity().Cmp(
		rightSwap.ActiveLiquidity(),
	) != 0 {
		return false, nil
	}

	if leftSwap.Tick() !=
		rightSwap.Tick() {
		return false, nil
	}

	return true, nil
}
