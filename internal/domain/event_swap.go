package domain

import (
	"fmt"
	"math/big"
)

// SwapEvent stores the complete post-swap pool state emitted by Uniswap v3.
//
// Amount0 and Amount1 use pool-relative signed semantics:
//   - positive means the pool received the token;
//   - negative means the pool sent the token.
type SwapEvent struct {
	sender          Address
	recipient       Address
	amount0         *big.Int
	amount1         *big.Int
	sqrtPriceX96    *big.Int
	activeLiquidity *big.Int
	tick            int32
}

func NewSwapEvent(
	sender Address,
	recipient Address,
	amount0 *big.Int,
	amount1 *big.Int,
	sqrtPriceX96 *big.Int,
	activeLiquidity *big.Int,
	tick int32,
) (SwapEvent, error) {
	event := SwapEvent{
		sender:          sender,
		recipient:       recipient,
		amount0:         cloneBigInt(amount0),
		amount1:         cloneBigInt(amount1),
		sqrtPriceX96:    cloneBigInt(sqrtPriceX96),
		activeLiquidity: cloneBigInt(activeLiquidity),
		tick:            tick,
	}

	if err := event.Validate(); err != nil {
		return SwapEvent{}, err
	}

	return event, nil
}

func (SwapEvent) Type() PoolEventType {
	return PoolEventSwap
}

func (SwapEvent) isPoolEventPayload() {}

func (e SwapEvent) Validate() error {
	if e.sender.IsZero() {
		return fmt.Errorf("swap sender is required")
	}

	if e.recipient.IsZero() {
		return fmt.Errorf("swap recipient is required")
	}

	if err := requireBigInt("swap amount0", e.amount0); err != nil {
		return err
	}

	if err := requireBigInt("swap amount1", e.amount1); err != nil {
		return err
	}

	if e.amount0.Sign() == 0 {
		return fmt.Errorf("swap amount0 must not be zero")
	}

	if e.amount1.Sign() == 0 {
		return fmt.Errorf("swap amount1 must not be zero")
	}

	if e.amount0.Sign() == e.amount1.Sign() {
		return fmt.Errorf(
			"swap amount0 and amount1 must have opposite signs",
		)
	}

	if err := requirePositiveBigInt(
		"swap sqrt price X96",
		e.sqrtPriceX96,
	); err != nil {
		return err
	}

	if err := requireNonNegativeBigInt(
		"swap active liquidity",
		e.activeLiquidity,
	); err != nil {
		return err
	}

	return nil
}

func (e SwapEvent) Sender() Address {
	return e.sender
}

func (e SwapEvent) Recipient() Address {
	return e.recipient
}

func (e SwapEvent) Amount0() *big.Int {
	return cloneBigInt(e.amount0)
}

func (e SwapEvent) Amount1() *big.Int {
	return cloneBigInt(e.amount1)
}

func (e SwapEvent) SqrtPriceX96() *big.Int {
	return cloneBigInt(e.sqrtPriceX96)
}

func (e SwapEvent) ActiveLiquidity() *big.Int {
	return cloneBigInt(e.activeLiquidity)
}

func (e SwapEvent) Tick() int32 {
	return e.tick
}

// ZeroForOne reports whether token0 was supplied to the pool in exchange for
// token1.
func (e SwapEvent) ZeroForOne() bool {
	return e.amount0.Sign() > 0 && e.amount1.Sign() < 0
}
