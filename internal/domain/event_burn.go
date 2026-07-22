package domain

import (
	"fmt"
	"math/big"
)

// BurnEvent removes liquidity from one owner-range core position.
//
// Burn does not necessarily mean that the position is fully removed. The event
// amount may represent any fraction of the position's current liquidity.
type BurnEvent struct {
	owner     Address
	tickLower int32
	tickUpper int32
	amount    *big.Int
}

func NewBurnEvent(
	owner Address,
	tickLower int32,
	tickUpper int32,
	amount *big.Int,
) (BurnEvent, error) {
	event := BurnEvent{
		owner:     owner,
		tickLower: tickLower,
		tickUpper: tickUpper,
		amount:    cloneBigInt(amount),
	}

	if err := event.Validate(); err != nil {
		return BurnEvent{}, err
	}

	return event, nil
}

func (BurnEvent) Type() PoolEventType {
	return PoolEventBurn
}

func (BurnEvent) isPoolEventPayload() {}

func (e BurnEvent) Validate() error {
	if e.owner.IsZero() {
		return fmt.Errorf("burn owner is required")
	}

	if e.tickLower >= e.tickUpper {
		return fmt.Errorf(
			"burn tick lower %d must be smaller than tick upper %d",
			e.tickLower,
			e.tickUpper,
		)
	}

	if err := requirePositiveBigInt(
		"burn liquidity amount",
		e.amount,
	); err != nil {
		return err
	}

	return nil
}

func (e BurnEvent) Owner() Address {
	return e.owner
}

func (e BurnEvent) TickLower() int32 {
	return e.tickLower
}

func (e BurnEvent) TickUpper() int32 {
	return e.tickUpper
}

func (e BurnEvent) Amount() *big.Int {
	return cloneBigInt(e.amount)
}

func (e BurnEvent) PositionKey(
	poolAddress Address,
) CorePositionKey {
	return CorePositionKey{
		PoolAddress: poolAddress,
		Owner:       e.owner,
		TickLower:   e.tickLower,
		TickUpper:   e.tickUpper,
	}
}
