package domain

import "fmt"

// PoolEventPayload is a sealed interface implemented only by supported
// Uniswap v3 event payloads in this package.
type PoolEventPayload interface {
	Type() PoolEventType
	Validate() error

	isPoolEventPayload()
}

// PoolEvent contains one ordered, canonical on-chain pool event.
//
// The event cursor determines the exact order of logs. Block number alone is
// insufficient because several swaps and liquidity actions may occur in the
// same transaction or block.
type PoolEvent struct {
	poolAddress     Address
	cursor          EventCursor
	blockHash       Hash
	transactionHash Hash
	payload         PoolEventPayload
}

func NewPoolEvent(
	poolAddress Address,
	cursor EventCursor,
	blockHash Hash,
	transactionHash Hash,
	payload PoolEventPayload,
) (PoolEvent, error) {
	event := PoolEvent{
		poolAddress:     poolAddress,
		cursor:          cursor,
		blockHash:       blockHash,
		transactionHash: transactionHash,
		payload:         payload,
	}

	if err := event.Validate(); err != nil {
		return PoolEvent{}, err
	}

	return event, nil
}

func (e PoolEvent) Validate() error {
	if e.poolAddress.IsZero() {
		return fmt.Errorf("pool event pool address is required")
	}

	if err := e.cursor.Validate(); err != nil {
		return fmt.Errorf("pool event cursor: %w", err)
	}

	if e.blockHash.IsZero() {
		return fmt.Errorf("pool event block hash is required")
	}

	if e.transactionHash.IsZero() {
		return fmt.Errorf("pool event transaction hash is required")
	}

	if e.payload == nil {
		return fmt.Errorf("pool event payload is required")
	}

	if err := e.payload.Type().Validate(); err != nil {
		return fmt.Errorf("pool event payload type: %w", err)
	}

	if err := e.payload.Validate(); err != nil {
		return fmt.Errorf(
			"pool event %s payload: %w",
			e.payload.Type(),
			err,
		)
	}

	return nil
}

func (e PoolEvent) PoolAddress() Address {
	return e.poolAddress
}

func (e PoolEvent) Cursor() EventCursor {
	return e.cursor
}

func (e PoolEvent) BlockHash() Hash {
	return e.blockHash
}

func (e PoolEvent) TransactionHash() Hash {
	return e.transactionHash
}

func (e PoolEvent) Type() PoolEventType {
	return e.payload.Type()
}

func (e PoolEvent) Payload() PoolEventPayload {
	return e.payload
}
