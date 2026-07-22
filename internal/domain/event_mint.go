package domain

import (
	"fmt"
	"math/big"
)

// MintEvent adds liquidity to one owner-range position.
//
// Sender and owner are distinct in the Uniswap v3 event. Sender initiates the
// action, while owner identifies the core position whose liquidity changes.
type MintEvent struct {
	sender    Address
	owner     Address
	tickLower int32
	tickUpper int32
	amount    *big.Int
}

func NewMintEvent(
	sender Address,
	owner Address,
	tickLower int32,
	tickUpper int32,
	amount *big.Int,
) (MintEvent, error) {
	event := MintEvent{
		sender:    sender,
		owner:     owner,
		tickLower: tickLower,
		tickUpper: tickUpper,
		amount:    cloneBigInt(amount),
	}

	if err := event.Validate(); err != nil {
		return MintEvent{}, err
	}

	return event, nil
}

func (MintEvent) Type() PoolEventType {
	return PoolEventMint
}

func (MintEvent) isPoolEventPayload() {}

func (e MintEvent) Validate() error {
	if e.sender.IsZero() {
		return fmt.Errorf("mint sender is required")
	}

	if e.owner.IsZero() {
		return fmt.Errorf("mint owner is required")
	}

	if e.tickLower >= e.tickUpper {
		return fmt.Errorf(
			"mint tick lower %d must be smaller than tick upper %d",
			e.tickLower,
			e.tickUpper,
		)
	}

	if err := requirePositiveBigInt(
		"mint liquidity amount",
		e.amount,
	); err != nil {
		return err
	}

	return nil
}

func (e MintEvent) Sender() Address {
	return e.sender
}

func (e MintEvent) Owner() Address {
	return e.owner
}

func (e MintEvent) TickLower() int32 {
	return e.tickLower
}

func (e MintEvent) TickUpper() int32 {
	return e.tickUpper
}

func (e MintEvent) Amount() *big.Int {
	return cloneBigInt(e.amount)
}

func (e MintEvent) PositionKey(
	poolAddress Address,
) CorePositionKey {
	return CorePositionKey{
		PoolAddress: poolAddress,
		Owner:       e.owner,
		TickLower:   e.tickLower,
		TickUpper:   e.tickUpper,
	}
}
