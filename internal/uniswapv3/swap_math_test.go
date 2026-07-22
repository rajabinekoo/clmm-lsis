package uniswapv3_test

import (
	"math/big"
	"testing"

	"github.com/rajabinekoo/clmm-lsis/internal/uniswapv3"
)

const testFeePips uint32 = 3000

func TestComputeSwapStepExactInputZeroForOneReachesTarget(
	t *testing.T,
) {
	t.Parallel()

	current := uniswapv3.Q96()

	target := new(big.Int).Quo(
		current,
		big.NewInt(2),
	)

	liquidity := big.NewInt(1_000_000_000_000_000_000)

	// This gross amount contains exactly enough input and fee to move from
	// Q96 to Q96/2.
	amountRemaining := big.NewInt(1_003_009_027_081_243_732)

	result, err := uniswapv3.ComputeSwapStepExactInput(
		current,
		target,
		liquidity,
		amountRemaining,
		testFeePips,
		true,
	)
	if err != nil {
		t.Fatalf(
			"ComputeSwapStepExactInput() error = %v",
			err,
		)
	}

	if !result.ReachedTarget() {
		t.Fatal(
			"ReachedTarget() = false, want true",
		)
	}

	assertBigIntEqual(
		t,
		result.SqrtPriceNextX96(),
		target,
	)

	assertBigIntEqual(
		t,
		result.AmountIn(),
		big.NewInt(1_000_000_000_000_000_000),
	)

	assertBigIntEqual(
		t,
		result.AmountOut(),
		big.NewInt(500_000_000_000_000_000),
	)

	assertBigIntEqual(
		t,
		result.FeeAmount(),
		big.NewInt(3_009_027_081_243_732),
	)

	assertBigIntEqual(
		t,
		result.TotalInputConsumed(),
		amountRemaining,
	)
}

func TestComputeSwapStepExactInputOneForZeroReachesTarget(
	t *testing.T,
) {
	t.Parallel()

	current := uniswapv3.Q96()

	target := new(big.Int).Mul(
		current,
		big.NewInt(2),
	)

	liquidity := big.NewInt(1_000_000_000_000_000_000)

	amountRemaining := big.NewInt(1_003_009_027_081_243_732)

	result, err := uniswapv3.ComputeSwapStepExactInput(
		current,
		target,
		liquidity,
		amountRemaining,
		testFeePips,
		false,
	)
	if err != nil {
		t.Fatalf(
			"ComputeSwapStepExactInput() error = %v",
			err,
		)
	}

	if !result.ReachedTarget() {
		t.Fatal(
			"ReachedTarget() = false, want true",
		)
	}

	assertBigIntEqual(
		t,
		result.SqrtPriceNextX96(),
		target,
	)

	assertBigIntEqual(
		t,
		result.AmountIn(),
		big.NewInt(1_000_000_000_000_000_000),
	)

	assertBigIntEqual(
		t,
		result.AmountOut(),
		big.NewInt(500_000_000_000_000_000),
	)

	assertBigIntEqual(
		t,
		result.FeeAmount(),
		big.NewInt(3_009_027_081_243_732),
	)
}

func TestComputeSwapStepExactInputPartialZeroForOne(
	t *testing.T,
) {
	t.Parallel()

	current := uniswapv3.Q96()

	target := new(big.Int).Quo(
		current,
		big.NewInt(2),
	)

	result, err := uniswapv3.ComputeSwapStepExactInput(
		current,
		target,
		big.NewInt(1_000_000_000_000_000_000),
		big.NewInt(100_000_000_000_000_000),
		testFeePips,
		true,
	)
	if err != nil {
		t.Fatalf(
			"ComputeSwapStepExactInput() error = %v",
			err,
		)
	}

	if result.ReachedTarget() {
		t.Fatal(
			"ReachedTarget() = true, want false",
		)
	}

	assertBigIntEqual(
		t,
		result.SqrtPriceNextX96(),
		mustTestBigInt(
			t,
			"72045250990510446115798809072",
		),
	)

	assertBigIntEqual(
		t,
		result.AmountIn(),
		big.NewInt(99_700_000_000_000_000),
	)

	assertBigIntEqual(
		t,
		result.AmountOut(),
		big.NewInt(90_661_089_388_014_913),
	)

	assertBigIntEqual(
		t,
		result.FeeAmount(),
		big.NewInt(300_000_000_000_000),
	)

	assertBigIntEqual(
		t,
		result.TotalInputConsumed(),
		big.NewInt(100_000_000_000_000_000),
	)
}

func TestComputeSwapStepExactInputWithZeroRemainingAmount(
	t *testing.T,
) {
	t.Parallel()

	current := uniswapv3.Q96()

	target := new(big.Int).Quo(
		current,
		big.NewInt(2),
	)

	result, err := uniswapv3.ComputeSwapStepExactInput(
		current,
		target,
		big.NewInt(1_000_000),
		new(big.Int),
		testFeePips,
		true,
	)
	if err != nil {
		t.Fatalf(
			"ComputeSwapStepExactInput() error = %v",
			err,
		)
	}

	if result.ReachedTarget() {
		t.Fatal(
			"ReachedTarget() = true, want false",
		)
	}

	assertBigIntEqual(
		t,
		result.SqrtPriceNextX96(),
		current,
	)

	assertBigIntEqual(
		t,
		result.TotalInputConsumed(),
		new(big.Int),
	)
}

func TestComputeSwapStepExactInputWithEqualTarget(
	t *testing.T,
) {
	t.Parallel()

	current := uniswapv3.Q96()

	result, err := uniswapv3.ComputeSwapStepExactInput(
		current,
		current,
		big.NewInt(1_000_000),
		big.NewInt(100_000),
		testFeePips,
		true,
	)
	if err != nil {
		t.Fatalf(
			"ComputeSwapStepExactInput() error = %v",
			err,
		)
	}

	if !result.ReachedTarget() {
		t.Fatal(
			"ReachedTarget() = false, want true",
		)
	}

	assertBigIntEqual(
		t,
		result.TotalInputConsumed(),
		new(big.Int),
	)
}

func TestComputeSwapStepRejectsInvalidTargetDirection(
	t *testing.T,
) {
	t.Parallel()

	current := uniswapv3.Q96()

	higherTarget := new(big.Int).Mul(
		current,
		big.NewInt(2),
	)

	_, err := uniswapv3.ComputeSwapStepExactInput(
		current,
		higherTarget,
		big.NewInt(1_000_000),
		big.NewInt(10_000),
		testFeePips,
		true,
	)
	if err == nil {
		t.Fatal(
			"ComputeSwapStepExactInput() expected direction error",
		)
	}
}

func TestComputeSwapStepRejectsInvalidFee(
	t *testing.T,
) {
	t.Parallel()

	current := uniswapv3.Q96()

	target := new(big.Int).Quo(
		current,
		big.NewInt(2),
	)

	_, err := uniswapv3.ComputeSwapStepExactInput(
		current,
		target,
		big.NewInt(1_000_000),
		big.NewInt(10_000),
		uniswapv3.FeeDenominatorPips,
		true,
	)
	if err == nil {
		t.Fatal(
			"ComputeSwapStepExactInput() expected fee error",
		)
	}
}

func TestSwapStepResultDoesNotExposeMutableIntegers(
	t *testing.T,
) {
	t.Parallel()

	current := uniswapv3.Q96()

	target := new(big.Int).Quo(
		current,
		big.NewInt(2),
	)

	result, err := uniswapv3.ComputeSwapStepExactInput(
		current,
		target,
		big.NewInt(1_000_000_000),
		big.NewInt(100_000),
		testFeePips,
		true,
	)
	if err != nil {
		t.Fatalf(
			"ComputeSwapStepExactInput() error = %v",
			err,
		)
	}

	original := result.AmountIn()

	exposed := result.AmountIn()
	exposed.SetInt64(0)

	assertBigIntEqual(
		t,
		result.AmountIn(),
		original,
	)
}

func mustTestBigInt(
	t *testing.T,
	value string,
) *big.Int {
	t.Helper()

	parsed, ok := new(big.Int).SetString(
		value,
		10,
	)
	if !ok {
		t.Fatalf(
			"invalid test integer %q",
			value,
		)
	}

	return parsed
}
