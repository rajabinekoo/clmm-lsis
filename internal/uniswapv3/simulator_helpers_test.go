package uniswapv3_test

import (
	"fmt"
	"math/big"
	"sort"
	"testing"

	"github.com/rajabinekoo/clmm-lsis/internal/domain"
	"github.com/rajabinekoo/clmm-lsis/internal/uniswapv3"
)

const simulatorTestFeePips uint32 = 500

type simulatorPositionSpec struct {
	ownerSuffix byte

	tickLower int32
	tickUpper int32

	liquidity *big.Int
}

type derivedTestTick struct {
	gross *big.Int
	net   *big.Int
}

func newSimulatorTestPool(
	t *testing.T,
) domain.Pool {
	t.Helper()

	address := mustSimulatorAddress(
		t,
		"0xaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
	)

	pool := domain.Pool{
		Name:        "simulator_test_pool",
		Address:     address,
		FeePips:     simulatorTestFeePips,
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

func newSimulatorForTest(
	t *testing.T,
) (
	domain.Pool,
	uniswapv3.Simulator,
) {
	t.Helper()

	pool := newSimulatorTestPool(t)

	simulator, err := uniswapv3.NewSimulator(pool)
	if err != nil {
		t.Fatalf(
			"NewSimulator() error = %v",
			err,
		)
	}

	return pool, simulator
}

func buildSimulatorSnapshot(
	t *testing.T,
	pool domain.Pool,
	blockNumber uint64,
	currentTick int32,
	sqrtPriceX96 *big.Int,
	specs []simulatorPositionSpec,
) domain.PoolSnapshot {
	t.Helper()

	if len(specs) == 0 {
		t.Fatal(
			"buildSimulatorSnapshot() requires at least one position",
		)
	}

	positions := make(
		[]domain.CorePosition,
		0,
		len(specs),
	)

	tickAccounting := make(
		map[int32]*derivedTestTick,
	)

	activeLiquidity := new(big.Int)

	for index, spec := range specs {
		if spec.liquidity == nil {
			t.Fatalf(
				"position spec %d liquidity is nil",
				index,
			)
		}

		if spec.liquidity.Sign() <= 0 {
			t.Fatalf(
				"position spec %d liquidity must be positive",
				index,
			)
		}

		owner := mustSimulatorAddress(
			t,
			simulatorOwnerAddress(
				spec.ownerSuffix,
			),
		)

		key := domain.CorePositionKey{
			PoolAddress: pool.Address,
			Owner:       owner,
			TickLower:   spec.tickLower,
			TickUpper:   spec.tickUpper,
		}

		position, err := domain.NewCorePosition(
			key,
			spec.liquidity,
		)
		if err != nil {
			t.Fatalf(
				"NewCorePosition(spec=%d) error = %v",
				index,
				err,
			)
		}

		positions = append(
			positions,
			position,
		)

		if position.IsActiveAt(currentTick) {
			activeLiquidity.Add(
				activeLiquidity,
				spec.liquidity,
			)
		}

		lower := ensureDerivedTestTick(
			tickAccounting,
			spec.tickLower,
		)

		lower.gross.Add(
			lower.gross,
			spec.liquidity,
		)

		lower.net.Add(
			lower.net,
			spec.liquidity,
		)

		upper := ensureDerivedTestTick(
			tickAccounting,
			spec.tickUpper,
		)

		upper.gross.Add(
			upper.gross,
			spec.liquidity,
		)

		upper.net.Sub(
			upper.net,
			spec.liquidity,
		)
	}

	tickIndexes := make(
		[]int32,
		0,
		len(tickAccounting),
	)

	for index := range tickAccounting {
		tickIndexes = append(
			tickIndexes,
			index,
		)
	}

	sort.Slice(
		tickIndexes,
		func(i, j int) bool {
			return tickIndexes[i] < tickIndexes[j]
		},
	)

	ticks := make(
		[]domain.TickState,
		0,
		len(tickIndexes),
	)

	for _, index := range tickIndexes {
		accounting := tickAccounting[index]

		tick, err := domain.NewTickState(
			index,
			accounting.gross,
			accounting.net,
		)
		if err != nil {
			t.Fatalf(
				"NewTickState(index=%d) error = %v",
				index,
				err,
			)
		}

		ticks = append(
			ticks,
			tick,
		)
	}

	reference, err :=
		domain.NewBlockEndSnapshotReference(
			blockNumber,
		)
	if err != nil {
		t.Fatalf(
			"NewBlockEndSnapshotReference() error = %v",
			err,
		)
	}

	snapshot, err := domain.NewPoolSnapshot(
		pool.Address,
		reference,
		sqrtPriceX96,
		currentTick,
		activeLiquidity,
		ticks,
		positions,
	)
	if err != nil {
		t.Fatalf(
			"NewPoolSnapshot() error = %v",
			err,
		)
	}

	return snapshot
}

func ensureDerivedTestTick(
	ticks map[int32]*derivedTestTick,
	index int32,
) *derivedTestTick {
	tick := ticks[index]

	if tick != nil {
		return tick
	}

	tick = &derivedTestTick{
		gross: new(big.Int),
		net:   new(big.Int),
	}

	ticks[index] = tick

	return tick
}

func grossInputRequiredToReachTarget(
	t *testing.T,
	currentSqrtPriceX96 *big.Int,
	targetSqrtPriceX96 *big.Int,
	liquidity *big.Int,
	feePips uint32,
	zeroForOne bool,
) *big.Int {
	t.Helper()

	var (
		netInput *big.Int
		err      error
	)

	if zeroForOne {
		netInput, err = uniswapv3.GetAmount0Delta(
			targetSqrtPriceX96,
			currentSqrtPriceX96,
			liquidity,
			true,
		)
	} else {
		netInput, err = uniswapv3.GetAmount1Delta(
			currentSqrtPriceX96,
			targetSqrtPriceX96,
			liquidity,
			true,
		)
	}

	if err != nil {
		t.Fatalf(
			"compute net input required for target error = %v",
			err,
		)
	}

	feeComplement := new(big.Int).SetUint64(
		uint64(
			uniswapv3.FeeDenominatorPips -
				feePips,
		),
	)

	feeAmount, err :=
		uniswapv3.MulDivRoundingUp(
			netInput,
			new(big.Int).SetUint64(
				uint64(feePips),
			),
			feeComplement,
		)
	if err != nil {
		t.Fatalf(
			"compute gross target fee error = %v",
			err,
		)
	}

	return new(big.Int).Add(
		netInput,
		feeAmount,
	)
}

func mustSqrtPriceAtTick(
	t *testing.T,
	tick int32,
) *big.Int {
	t.Helper()

	value, err :=
		uniswapv3.GetSqrtRatioAtTick(tick)
	if err != nil {
		t.Fatalf(
			"GetSqrtRatioAtTick(%d) error = %v",
			tick,
			err,
		)
	}

	return value
}

func mustSimulatorAddress(
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

func simulatorOwnerAddress(
	suffix byte,
) string {
	return fmt.Sprintf(
		"0x%040x",
		suffix,
	)
}

func assertInt32SliceEqual(
	t *testing.T,
	actual []int32,
	expected []int32,
) {
	t.Helper()

	if len(actual) != len(expected) {
		t.Fatalf(
			"slice length = %d, want %d; actual=%v expected=%v",
			len(actual),
			len(expected),
			actual,
			expected,
		)
	}

	for index := range expected {
		if actual[index] != expected[index] {
			t.Fatalf(
				"slice[%d] = %d, want %d; actual=%v expected=%v",
				index,
				actual[index],
				expected[index],
				actual,
				expected,
			)
		}
	}
}
