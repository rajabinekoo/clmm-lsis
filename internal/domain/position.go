package domain

import (
	"fmt"
	"math/big"
)

// CorePositionKey identifies one Uniswap v3 core owner-range position.
//
// This key does not necessarily correspond one-to-one with an NFT token ID.
// Multiple higher-level positions may be aggregated by the core pool under
// the same owner and tick range.
type CorePositionKey struct {
	PoolAddress Address
	Owner       Address
	TickLower   int32
	TickUpper   int32
}

func (k CorePositionKey) Validate() error {
	if k.PoolAddress.IsZero() {
		return fmt.Errorf("position pool address is required")
	}

	if k.Owner.IsZero() {
		return fmt.Errorf("position owner is required")
	}

	if k.TickLower >= k.TickUpper {
		return fmt.Errorf(
			"position tick lower %d must be smaller than tick upper %d",
			k.TickLower,
			k.TickUpper,
		)
	}

	return nil
}

func (k CorePositionKey) String() string {
	return fmt.Sprintf(
		"%s:%s:%d:%d",
		k.PoolAddress,
		k.Owner,
		k.TickLower,
		k.TickUpper,
	)
}

// CorePosition stores the current liquidity of one owner-range position.
type CorePosition struct {
	key       CorePositionKey
	liquidity *big.Int
}

func NewCorePosition(
	key CorePositionKey,
	liquidity *big.Int,
) (CorePosition, error) {
	position := CorePosition{
		key:       key,
		liquidity: cloneBigInt(liquidity),
	}

	if err := position.Validate(); err != nil {
		return CorePosition{}, err
	}

	return position, nil
}

func (p CorePosition) Validate() error {
	if err := p.key.Validate(); err != nil {
		return err
	}

	if err := requirePositiveBigInt(
		"position liquidity",
		p.liquidity,
	); err != nil {
		return err
	}

	return nil
}

func (p CorePosition) Key() CorePositionKey {
	return p.key
}

func (p CorePosition) Liquidity() *big.Int {
	return cloneBigInt(p.liquidity)
}

func (p CorePosition) IsActiveAt(
	currentTick int32,
) bool {
	return p.key.TickLower <= currentTick &&
		currentTick < p.key.TickUpper
}

func (p CorePosition) ApplyDelta(
	delta LiquidityDelta,
) (*CorePosition, error) {
	if err := p.Validate(); err != nil {
		return nil, err
	}

	nextLiquidity, err := delta.Apply(p.liquidity)
	if err != nil {
		return nil, fmt.Errorf(
			"apply liquidity delta to position %s: %w",
			p.key,
			err,
		)
	}

	if nextLiquidity.Sign() == 0 {
		return nil, nil
	}

	nextPosition, err := NewCorePosition(
		p.key,
		nextLiquidity,
	)
	if err != nil {
		return nil, err
	}

	return &nextPosition, nil
}
