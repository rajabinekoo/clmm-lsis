package uniswapv3_test

import (
	"math/big"
	"testing"

	"github.com/rajabinekoo/clmm-lsis/internal/uniswapv3"
)

func TestAddLiquidityDeltaAddsPositiveDelta(
	t *testing.T,
) {
	t.Parallel()

	actual, err := uniswapv3.AddLiquidityDelta(
		big.NewInt(1_000),
		big.NewInt(250),
	)
	if err != nil {
		t.Fatalf(
			"AddLiquidityDelta() error = %v",
			err,
		)
	}

	assertBigIntEqual(
		t,
		actual,
		big.NewInt(1_250),
	)
}

func TestAddLiquidityDeltaSubtractsNegativeDelta(
	t *testing.T,
) {
	t.Parallel()

	actual, err := uniswapv3.AddLiquidityDelta(
		big.NewInt(1_000),
		big.NewInt(-250),
	)
	if err != nil {
		t.Fatalf(
			"AddLiquidityDelta() error = %v",
			err,
		)
	}

	assertBigIntEqual(
		t,
		actual,
		big.NewInt(750),
	)
}

func TestAddLiquidityDeltaRejectsUnderflow(
	t *testing.T,
) {
	t.Parallel()

	_, err := uniswapv3.AddLiquidityDelta(
		big.NewInt(1_000),
		big.NewInt(-1_001),
	)
	if err == nil {
		t.Fatal(
			"AddLiquidityDelta() expected underflow error",
		)
	}
}

func TestApplyLiquidityNetZeroForOneNegatesTickNet(
	t *testing.T,
) {
	t.Parallel()

	// Crossing from right to left:
	//
	// current liquidity = 1,000
	// tick liquidity net = +400
	//
	// The effective delta is -400.
	actual, err := uniswapv3.ApplyLiquidityNet(
		big.NewInt(1_000),
		big.NewInt(400),
		true,
	)
	if err != nil {
		t.Fatalf(
			"ApplyLiquidityNet() error = %v",
			err,
		)
	}

	assertBigIntEqual(
		t,
		actual,
		big.NewInt(600),
	)
}

func TestApplyLiquidityNetOneForZeroAppliesTickNetDirectly(
	t *testing.T,
) {
	t.Parallel()

	// Crossing from left to right applies liquidityNet directly.
	actual, err := uniswapv3.ApplyLiquidityNet(
		big.NewInt(1_000),
		big.NewInt(-300),
		false,
	)
	if err != nil {
		t.Fatalf(
			"ApplyLiquidityNet() error = %v",
			err,
		)
	}

	assertBigIntEqual(
		t,
		actual,
		big.NewInt(700),
	)
}

func TestApplyLiquidityNetDoesNotMutateInputs(
	t *testing.T,
) {
	t.Parallel()

	current := big.NewInt(1_000)
	net := big.NewInt(400)

	_, err := uniswapv3.ApplyLiquidityNet(
		current,
		net,
		true,
	)
	if err != nil {
		t.Fatalf(
			"ApplyLiquidityNet() error = %v",
			err,
		)
	}

	assertBigIntEqual(
		t,
		current,
		big.NewInt(1_000),
	)

	assertBigIntEqual(
		t,
		net,
		big.NewInt(400),
	)
}
