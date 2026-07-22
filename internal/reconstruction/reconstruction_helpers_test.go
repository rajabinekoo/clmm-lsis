package reconstruction_test

import (
	"fmt"
	"math/big"
	"testing"

	"github.com/rajabinekoo/clmm-lsis/internal/domain"
	"github.com/rajabinekoo/clmm-lsis/internal/reconstruction"
	"github.com/rajabinekoo/clmm-lsis/internal/uniswapv3"
)

func newReconstructionTestPool(
	t *testing.T,
) domain.Pool {
	t.Helper()

	pool := domain.Pool{
		Name: "reconstruction_test_pool",

		Address: mustReconstructionAddress(
			t,
			"0xaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		),

		FeePips:     500,
		TickSpacing: 10,

		Token0: domain.Token{
			Symbol:   "TOKEN0",
			Decimals: 18,
		},

		Token1: domain.Token{
			Symbol:   "TOKEN1",
			Decimals: 18,
		},
	}

	if err := pool.Validate(); err != nil {
		t.Fatalf(
			"test pool validation error = %v",
			err,
		)
	}

	return pool
}

func newMutableState(
	t *testing.T,
	pool domain.Pool,
) *reconstruction.MutablePoolState {
	t.Helper()

	state, err :=
		reconstruction.NewMutablePoolState(pool)
	if err != nil {
		t.Fatalf(
			"NewMutablePoolState() error = %v",
			err,
		)
	}

	return state
}

func initializeStateAtTick(
	t *testing.T,
	state *reconstruction.MutablePoolState,
	pool domain.Pool,
	cursor domain.EventCursor,
	tick int32,
) domain.PoolEvent {
	t.Helper()

	sqrtPriceX96 := mustReconstructionSqrtPrice(
		t,
		tick,
	)

	payload, err := domain.NewInitializeEvent(
		sqrtPriceX96,
		tick,
	)
	if err != nil {
		t.Fatalf(
			"NewInitializeEvent() error = %v",
			err,
		)
	}

	event := mustPoolEvent(
		t,
		pool.Address,
		cursor,
		payload,
	)

	if err := state.Apply(event); err != nil {
		t.Fatalf(
			"Apply(initialize) error = %v",
			err,
		)
	}

	return event
}

func newMintPoolEvent(
	t *testing.T,
	pool domain.Pool,
	cursor domain.EventCursor,
	owner domain.Address,
	tickLower int32,
	tickUpper int32,
	amount *big.Int,
) domain.PoolEvent {
	t.Helper()

	sender := mustReconstructionAddress(
		t,
		"0x1111111111111111111111111111111111111111",
	)

	payload, err := domain.NewMintEvent(
		sender,
		owner,
		tickLower,
		tickUpper,
		amount,
	)
	if err != nil {
		t.Fatalf(
			"NewMintEvent() error = %v",
			err,
		)
	}

	return mustPoolEvent(
		t,
		pool.Address,
		cursor,
		payload,
	)
}

func newBurnPoolEvent(
	t *testing.T,
	pool domain.Pool,
	cursor domain.EventCursor,
	owner domain.Address,
	tickLower int32,
	tickUpper int32,
	amount *big.Int,
) domain.PoolEvent {
	t.Helper()

	payload, err := domain.NewBurnEvent(
		owner,
		tickLower,
		tickUpper,
		amount,
	)
	if err != nil {
		t.Fatalf(
			"NewBurnEvent() error = %v",
			err,
		)
	}

	return mustPoolEvent(
		t,
		pool.Address,
		cursor,
		payload,
	)
}

func newSwapPoolEvent(
	t *testing.T,
	pool domain.Pool,
	cursor domain.EventCursor,
	zeroForOne bool,
	sqrtPriceX96 *big.Int,
	tick int32,
	activeLiquidity *big.Int,
) domain.PoolEvent {
	t.Helper()

	sender := mustReconstructionAddress(
		t,
		"0x2222222222222222222222222222222222222222",
	)

	recipient := mustReconstructionAddress(
		t,
		"0x3333333333333333333333333333333333333333",
	)

	amount0 := big.NewInt(1_000_000)
	amount1 := big.NewInt(-900_000)

	if !zeroForOne {
		amount0 = big.NewInt(-900_000)
		amount1 = big.NewInt(1_000_000)
	}

	payload, err := domain.NewSwapEvent(
		sender,
		recipient,
		amount0,
		amount1,
		sqrtPriceX96,
		activeLiquidity,
		tick,
	)
	if err != nil {
		t.Fatalf(
			"NewSwapEvent() error = %v",
			err,
		)
	}

	return mustPoolEvent(
		t,
		pool.Address,
		cursor,
		payload,
	)
}

func mustPoolEvent(
	t *testing.T,
	poolAddress domain.Address,
	cursor domain.EventCursor,
	payload domain.PoolEventPayload,
) domain.PoolEvent {
	t.Helper()

	blockHash := mustReconstructionHash(
		t,
		fmt.Sprintf(
			"0x%064x",
			cursor.BlockNumber,
		),
	)

	transactionHash := mustReconstructionHash(
		t,
		fmt.Sprintf(
			"0x%064x",
			uint64(cursor.BlockNumber)*1_000_000+
				uint64(cursor.TransactionIndex)*1_000+
				uint64(cursor.LogIndex)+1,
		),
	)

	event, err := domain.NewPoolEvent(
		poolAddress,
		cursor,
		blockHash,
		transactionHash,
		payload,
	)
	if err != nil {
		t.Fatalf(
			"NewPoolEvent() error = %v",
			err,
		)
	}

	return event
}

func mustApplyEvent(
	t *testing.T,
	state *reconstruction.MutablePoolState,
	event domain.PoolEvent,
) {
	t.Helper()

	if err := state.Apply(event); err != nil {
		t.Fatalf(
			"Apply(%s at %s) error = %v",
			event.Type(),
			event.Cursor(),
			err,
		)
	}
}

func mustReconstructionAddress(
	t *testing.T,
	value string,
) domain.Address {
	t.Helper()

	address, err := domain.ParseAddress(value)
	if err != nil {
		t.Fatalf(
			"ParseAddress(%q) error = %v",
			value,
			err,
		)
	}

	return address
}

func mustReconstructionHash(
	t *testing.T,
	value string,
) domain.Hash {
	t.Helper()

	hash, err := domain.ParseHash(value)
	if err != nil {
		t.Fatalf(
			"ParseHash(%q) error = %v",
			value,
			err,
		)
	}

	return hash
}

func mustReconstructionSqrtPrice(
	t *testing.T,
	tick int32,
) *big.Int {
	t.Helper()

	sqrtPriceX96, err :=
		uniswapv3.GetSqrtRatioAtTick(tick)
	if err != nil {
		t.Fatalf(
			"GetSqrtRatioAtTick(%d) error = %v",
			tick,
			err,
		)
	}

	return sqrtPriceX96
}

func reconstructionOwner(
	t *testing.T,
	suffix byte,
) domain.Address {
	t.Helper()

	return mustReconstructionAddress(
		t,
		fmt.Sprintf(
			"0x%040x",
			suffix,
		),
	)
}

func reconstructionCursor(
	blockNumber uint64,
	transactionIndex uint32,
	logIndex uint32,
) domain.EventCursor {
	return domain.EventCursor{
		BlockNumber:      blockNumber,
		TransactionIndex: transactionIndex,
		LogIndex:         logIndex,
	}
}

func assertReconstructionBigIntEqual(
	t *testing.T,
	actual *big.Int,
	expected *big.Int,
) {
	t.Helper()

	if actual == nil {
		t.Fatal(
			"actual big.Int is nil",
		)
	}

	if expected == nil {
		t.Fatal(
			"expected big.Int is nil",
		)
	}

	if actual.Cmp(expected) != 0 {
		t.Fatalf(
			"value = %s, want %s",
			actual,
			expected,
		)
	}
}

func assertSnapshotStateEqual(
	t *testing.T,
	actual domain.PoolSnapshot,
	expected domain.PoolSnapshot,
) {
	t.Helper()

	if actual.PoolAddress() != expected.PoolAddress() {
		t.Fatalf(
			"pool address = %s, want %s",
			actual.PoolAddress(),
			expected.PoolAddress(),
		)
	}

	if actual.CurrentTick() != expected.CurrentTick() {
		t.Fatalf(
			"current tick = %d, want %d",
			actual.CurrentTick(),
			expected.CurrentTick(),
		)
	}

	assertReconstructionBigIntEqual(
		t,
		actual.SqrtPriceX96(),
		expected.SqrtPriceX96(),
	)

	assertReconstructionBigIntEqual(
		t,
		actual.ActiveLiquidity(),
		expected.ActiveLiquidity(),
	)

	actualTicks := actual.Ticks()
	expectedTicks := expected.Ticks()

	if len(actualTicks) != len(expectedTicks) {
		t.Fatalf(
			"tick count = %d, want %d",
			len(actualTicks),
			len(expectedTicks),
		)
	}

	for index := range expectedTicks {
		actualTick := actualTicks[index]
		expectedTick := expectedTicks[index]

		if actualTick.Index() != expectedTick.Index() {
			t.Fatalf(
				"tick[%d].index = %d, want %d",
				index,
				actualTick.Index(),
				expectedTick.Index(),
			)
		}

		assertReconstructionBigIntEqual(
			t,
			actualTick.LiquidityGross(),
			expectedTick.LiquidityGross(),
		)

		assertReconstructionBigIntEqual(
			t,
			actualTick.LiquidityNet(),
			expectedTick.LiquidityNet(),
		)
	}

	actualPositions := actual.Positions()
	expectedPositions := expected.Positions()

	if len(actualPositions) !=
		len(expectedPositions) {
		t.Fatalf(
			"position count = %d, want %d",
			len(actualPositions),
			len(expectedPositions),
		)
	}

	for index := range expectedPositions {
		actualPosition := actualPositions[index]
		expectedPosition := expectedPositions[index]

		if actualPosition.Key() !=
			expectedPosition.Key() {
			t.Fatalf(
				"position[%d].key = %s, want %s",
				index,
				actualPosition.Key(),
				expectedPosition.Key(),
			)
		}

		assertReconstructionBigIntEqual(
			t,
			actualPosition.Liquidity(),
			expectedPosition.Liquidity(),
		)
	}
}
