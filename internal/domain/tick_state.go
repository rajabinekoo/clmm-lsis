package domain

import (
	"fmt"
	"math/big"
)

// TickState stores the liquidity accounting values of one initialized tick.
type TickState struct {
	index          int32
	liquidityGross *big.Int
	liquidityNet   *big.Int
}

func NewTickState(
	index int32,
	liquidityGross *big.Int,
	liquidityNet *big.Int,
) (TickState, error) {
	tick := TickState{
		index:          index,
		liquidityGross: cloneBigInt(liquidityGross),
		liquidityNet:   cloneBigInt(liquidityNet),
	}

	if err := tick.Validate(); err != nil {
		return TickState{}, err
	}

	return tick, nil
}

func (t TickState) Validate() error {
	if err := requireNonNegativeBigInt(
		"tick liquidity gross",
		t.liquidityGross,
	); err != nil {
		return err
	}

	if err := requireBigInt(
		"tick liquidity net",
		t.liquidityNet,
	); err != nil {
		return err
	}

	absoluteNet := absoluteBigInt(t.liquidityNet)

	if absoluteNet.Cmp(t.liquidityGross) > 0 {
		return fmt.Errorf(
			"absolute tick liquidity net %s exceeds liquidity gross %s at tick %d",
			absoluteNet,
			t.liquidityGross,
			t.index,
		)
	}

	if t.liquidityGross.Sign() == 0 &&
		t.liquidityNet.Sign() != 0 {
		return fmt.Errorf(
			"zero-gross tick %d must also have zero liquidity net",
			t.index,
		)
	}

	return nil
}

func (t TickState) Index() int32 {
	return t.index
}

func (t TickState) LiquidityGross() *big.Int {
	return cloneBigInt(t.liquidityGross)
}

func (t TickState) LiquidityNet() *big.Int {
	return cloneBigInt(t.liquidityNet)
}

func (t TickState) Initialized() bool {
	return t.liquidityGross.Sign() > 0
}

// ApplyPositionDelta applies one mint or burn to this boundary.
//
// At a lower boundary, liquidity net changes in the same direction as the
// position liquidity. At an upper boundary, the sign is reversed.
func (t TickState) ApplyPositionDelta(
	delta LiquidityDelta,
	lowerBoundary bool,
) (TickState, error) {
	if err := t.Validate(); err != nil {
		return TickState{}, err
	}

	if err := delta.Validate(); err != nil {
		return TickState{}, err
	}

	nextGross := new(big.Int).Add(
		t.liquidityGross,
		delta.Value(),
	)

	if nextGross.Sign() < 0 {
		return TickState{}, fmt.Errorf(
			"liquidity delta %s makes gross liquidity negative at tick %d",
			delta.Value(),
			t.index,
		)
	}

	netDelta := delta.Value()

	if !lowerBoundary {
		netDelta.Neg(netDelta)
	}

	nextNet := new(big.Int).Add(
		t.liquidityNet,
		netDelta,
	)

	return NewTickState(
		t.index,
		nextGross,
		nextNet,
	)
}

// EmptyTickState returns the temporary zero state used when a boundary has not
// previously been initialized.
func EmptyTickState(
	index int32,
) TickState {
	return TickState{
		index:          index,
		liquidityGross: new(big.Int),
		liquidityNet:   new(big.Int),
	}
}
