package domain

import "fmt"

// PoolEventType identifies one state-changing Uniswap v3 pool event.
//
// Collect, Flash and protocol-fee events are intentionally excluded because
// they do not change the liquidity geometry, active liquidity, current tick or
// sqrt price required by the primary LSIS analysis.
type PoolEventType uint8

const (
	PoolEventInitialize PoolEventType = iota + 1
	PoolEventMint
	PoolEventBurn
	PoolEventSwap
)

func (t PoolEventType) String() string {
	switch t {
	case PoolEventInitialize:
		return "initialize"
	case PoolEventMint:
		return "mint"
	case PoolEventBurn:
		return "burn"
	case PoolEventSwap:
		return "swap"
	default:
		return fmt.Sprintf("unknown(%d)", t)
	}
}

func (t PoolEventType) Validate() error {
	switch t {
	case PoolEventInitialize,
		PoolEventMint,
		PoolEventBurn,
		PoolEventSwap:
		return nil
	default:
		return fmt.Errorf("unsupported pool event type %d", t)
	}
}
