package uniswapv3_test

import (
	"math/big"
	"testing"

	"github.com/rajabinekoo/clmm-lsis/internal/uniswapv3"
)

func TestGetNextSqrtPriceFromAmount0Add(
	t *testing.T,
) {
	t.Parallel()

	q96 := uniswapv3.Q96()
	liquidity := big.NewInt(1_000_000_000_000_000_000)
	amount := big.NewInt(1_000_000_000_000_000_000)

	actual, err :=
		uniswapv3.GetNextSqrtPriceFromAmount0RoundingUp(
			q96,
			liquidity,
			amount,
			true,
		)
	if err != nil {
		t.Fatalf(
			"GetNextSqrtPriceFromAmount0RoundingUp() error = %v",
			err,
		)
	}

	expected := new(big.Int).Quo(
		q96,
		big.NewInt(2),
	)

	assertBigIntEqual(
		t,
		actual,
		expected,
	)
}

func TestGetNextSqrtPriceFromAmount0Remove(
	t *testing.T,
) {
	t.Parallel()

	q96 := uniswapv3.Q96()
	liquidity := big.NewInt(2_000_000_000_000_000_000)
	amount := big.NewInt(1_000_000_000_000_000_000)

	actual, err :=
		uniswapv3.GetNextSqrtPriceFromAmount0RoundingUp(
			q96,
			liquidity,
			amount,
			false,
		)
	if err != nil {
		t.Fatalf(
			"GetNextSqrtPriceFromAmount0RoundingUp() error = %v",
			err,
		)
	}

	expected := new(big.Int).Mul(
		q96,
		big.NewInt(2),
	)

	assertBigIntEqual(
		t,
		actual,
		expected,
	)
}

func TestGetNextSqrtPriceFromAmount0UsesOverflowFallback(
	t *testing.T,
) {
	t.Parallel()

	amount := new(big.Int).Lsh(
		big.NewInt(1),
		200,
	)

	actual, err :=
		uniswapv3.GetNextSqrtPriceFromAmount0RoundingUp(
			uniswapv3.Q96(),
			big.NewInt(1_000_000_000_000_000_000),
			amount,
			true,
		)
	if err != nil {
		t.Fatalf(
			"GetNextSqrtPriceFromAmount0RoundingUp() error = %v",
			err,
		)
	}

	assertBigIntEqual(
		t,
		actual,
		big.NewInt(1),
	)
}

func TestGetNextSqrtPriceFromAmount1Add(
	t *testing.T,
) {
	t.Parallel()

	q96 := uniswapv3.Q96()

	actual, err :=
		uniswapv3.GetNextSqrtPriceFromAmount1RoundingDown(
			q96,
			big.NewInt(1_000_000_000_000_000_000),
			big.NewInt(1_000_000_000_000_000_000),
			true,
		)
	if err != nil {
		t.Fatalf(
			"GetNextSqrtPriceFromAmount1RoundingDown() error = %v",
			err,
		)
	}

	expected := new(big.Int).Mul(
		q96,
		big.NewInt(2),
	)

	assertBigIntEqual(
		t,
		actual,
		expected,
	)
}

func TestGetNextSqrtPriceFromAmount1Remove(
	t *testing.T,
) {
	t.Parallel()

	q96 := uniswapv3.Q96()

	current := new(big.Int).Mul(
		q96,
		big.NewInt(2),
	)

	actual, err :=
		uniswapv3.GetNextSqrtPriceFromAmount1RoundingDown(
			current,
			big.NewInt(1_000_000_000_000_000_000),
			big.NewInt(1_000_000_000_000_000_000),
			false,
		)
	if err != nil {
		t.Fatalf(
			"GetNextSqrtPriceFromAmount1RoundingDown() error = %v",
			err,
		)
	}

	assertBigIntEqual(
		t,
		actual,
		q96,
	)
}

func TestGetAmount0DeltaKnownValue(
	t *testing.T,
) {
	t.Parallel()

	q96 := uniswapv3.Q96()

	doubleQ96 := new(big.Int).Mul(
		q96,
		big.NewInt(2),
	)

	actual, err := uniswapv3.GetAmount0Delta(
		q96,
		doubleQ96,
		big.NewInt(1_000_000_000_000_000_000),
		false,
	)
	if err != nil {
		t.Fatalf(
			"GetAmount0Delta() error = %v",
			err,
		)
	}

	assertBigIntEqual(
		t,
		actual,
		big.NewInt(500_000_000_000_000_000),
	)
}

func TestGetAmount1DeltaKnownValue(
	t *testing.T,
) {
	t.Parallel()

	q96 := uniswapv3.Q96()

	doubleQ96 := new(big.Int).Mul(
		q96,
		big.NewInt(2),
	)

	actual, err := uniswapv3.GetAmount1Delta(
		q96,
		doubleQ96,
		big.NewInt(1_000_000_000_000_000_000),
		false,
	)
	if err != nil {
		t.Fatalf(
			"GetAmount1Delta() error = %v",
			err,
		)
	}

	assertBigIntEqual(
		t,
		actual,
		big.NewInt(1_000_000_000_000_000_000),
	)
}

func TestAmountDeltasAcceptReversedPriceOrder(
	t *testing.T,
) {
	t.Parallel()

	q96 := uniswapv3.Q96()

	doubleQ96 := new(big.Int).Mul(
		q96,
		big.NewInt(2),
	)

	forward, err := uniswapv3.GetAmount0Delta(
		q96,
		doubleQ96,
		big.NewInt(1_000_000),
		true,
	)
	if err != nil {
		t.Fatalf(
			"GetAmount0Delta(forward) error = %v",
			err,
		)
	}

	reversed, err := uniswapv3.GetAmount0Delta(
		doubleQ96,
		q96,
		big.NewInt(1_000_000),
		true,
	)
	if err != nil {
		t.Fatalf(
			"GetAmount0Delta(reversed) error = %v",
			err,
		)
	}

	assertBigIntEqual(
		t,
		forward,
		reversed,
	)
}

func TestGetNextSqrtPriceFromInputReturnsCurrentPriceForZeroInput(
	t *testing.T,
) {
	t.Parallel()

	current := uniswapv3.Q96()

	actual, err := uniswapv3.GetNextSqrtPriceFromInput(
		current,
		big.NewInt(1_000_000),
		new(big.Int),
		true,
	)
	if err != nil {
		t.Fatalf(
			"GetNextSqrtPriceFromInput() error = %v",
			err,
		)
	}

	assertBigIntEqual(
		t,
		actual,
		current,
	)
}

func TestGetNextSqrtPriceRejectsZeroLiquidity(
	t *testing.T,
) {
	t.Parallel()

	_, err := uniswapv3.GetNextSqrtPriceFromInput(
		uniswapv3.Q96(),
		new(big.Int),
		big.NewInt(1),
		true,
	)
	if err == nil {
		t.Fatal(
			"GetNextSqrtPriceFromInput() expected error",
		)
	}
}

func TestToken0RemovalRejectsExcessiveAmount(
	t *testing.T,
) {
	t.Parallel()

	_, err :=
		uniswapv3.GetNextSqrtPriceFromAmount0RoundingUp(
			uniswapv3.Q96(),
			big.NewInt(1_000),
			big.NewInt(1_000),
			false,
		)
	if err == nil {
		t.Fatal(
			"GetNextSqrtPriceFromAmount0RoundingUp() expected error",
		)
	}
}

func assertBigIntEqual(
	t *testing.T,
	actual *big.Int,
	expected *big.Int,
) {
	t.Helper()

	if actual.Cmp(expected) != 0 {
		t.Fatalf(
			"value = %s, want %s",
			actual,
			expected,
		)
	}
}
