package domain

import (
	"fmt"
	"math/big"
)

// InitializeEvent establishes the first valid pool price and tick.
//
// A Uniswap v3 pool cannot process liquidity or swaps before initialization.
type InitializeEvent struct {
	sqrtPriceX96 *big.Int
	tick         int32
}

func NewInitializeEvent(
	sqrtPriceX96 *big.Int,
	tick int32,
) (InitializeEvent, error) {
	event := InitializeEvent{
		sqrtPriceX96: cloneBigInt(sqrtPriceX96),
		tick:         tick,
	}

	if err := event.Validate(); err != nil {
		return InitializeEvent{}, err
	}

	return event, nil
}

func (InitializeEvent) Type() PoolEventType {
	return PoolEventInitialize
}

func (InitializeEvent) isPoolEventPayload() {}

func (e InitializeEvent) Validate() error {
	if err := requirePositiveBigInt(
		"initialize sqrt price X96",
		e.sqrtPriceX96,
	); err != nil {
		return err
	}

	return nil
}

func (e InitializeEvent) SqrtPriceX96() *big.Int {
	return cloneBigInt(e.sqrtPriceX96)
}

func (e InitializeEvent) Tick() int32 {
	return e.tick
}

func (e InitializeEvent) String() string {
	return fmt.Sprintf(
		"initialize(sqrt_price_x96=%s,tick=%d)",
		e.sqrtPriceX96,
		e.tick,
	)
}
