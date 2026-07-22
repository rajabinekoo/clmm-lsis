package storage

import "time"

// LegacyPoolRecord maps the existing pools table.
//
// Token symbols are intentionally absent because the legacy database stores
// token addresses and decimals, while symbols belong to the study
// configuration.
type LegacyPoolRecord struct {
	Address string

	Token0Address string
	Token1Address string

	Token0Decimals uint8
	Token1Decimals uint8

	FeeTier     uint32
	TickSpacing int32

	CreatedBlock uint64
}

// LegacyLPActionType maps the existing lp_actions.action values.
type LegacyLPActionType int16

const (
	LegacyLPActionMint LegacyLPActionType = 1
	LegacyLPActionBurn LegacyLPActionType = 2
)

// LegacyLPActionRecord maps one existing lp_actions row.
//
// TransactionIndex and BlockHash are optional because the original table did
// not persist them. LogIndex is sufficient for ordering inside a block.
type LegacyLPActionRecord struct {
	ID string

	PoolAddress string
	Action      LegacyLPActionType

	TransactionHash string
	BlockNumber     uint64
	LogIndex        uint32
	Timestamp       time.Time

	BlockHash        *string
	TransactionIndex *uint32

	Owner  *string
	Sender *string
	Origin *string

	TickLower int32
	TickUpper int32

	// LiquidityDelta is stored as a base-10 signed integer:
	//
	//	Mint: positive
	//	Burn: negative
	LiquidityDelta string
}

// LegacyPoolSnapshotRecord maps the existing pool_snapshots table.
//
// The legacy snapshot stores only scalar pool state. Position and tick state
// must be reconstructed from lp_actions.
type LegacyPoolSnapshotRecord struct {
	PoolAddress string
	BlockNumber uint64

	SqrtPriceX96    string
	CurrentTick     *int32
	ActiveLiquidity string
}

// SwapRecord maps the append-only pool_swaps table that will be added later.
//
// All token amounts are raw signed integer values exactly as emitted by the
// Uniswap v3 Swap event.
type SwapRecord struct {
	PoolAddress string

	BlockNumber      uint64
	BlockHash        string
	TransactionHash  string
	TransactionIndex uint32
	LogIndex         uint32
	Timestamp        time.Time

	Sender    string
	Recipient string

	Amount0Raw string
	Amount1Raw string

	SqrtPriceX96    string
	ActiveLiquidity string
	Tick            int32
}
