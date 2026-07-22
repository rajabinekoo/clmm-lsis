package uniswapv3_test

import (
	"math/big"
	"testing"

	"github.com/rajabinekoo/clmm-lsis/internal/uniswapv3"
)

func TestMulDivUsesExactIntermediateProduct(
	t *testing.T,
) {
	t.Parallel()

	a := new(big.Int).Lsh(
		big.NewInt(1),
		200,
	)

	b := new(big.Int).Lsh(
		big.NewInt(1),
		100,
	)

	denominator := new(big.Int).Lsh(
		big.NewInt(1),
		50,
	)

	actual, err := uniswapv3.MulDiv(
		a,
		b,
		denominator,
	)
	if err != nil {
		t.Fatalf("MulDiv() error = %v", err)
	}

	expected := new(big.Int).Lsh(
		big.NewInt(1),
		250,
	)

	if actual.Cmp(expected) != 0 {
		t.Fatalf(
			"MulDiv() = %s, want %s",
			actual,
			expected,
		)
	}
}

func TestMulDivRoundsDown(
	t *testing.T,
) {
	t.Parallel()

	actual, err := uniswapv3.MulDiv(
		big.NewInt(10),
		big.NewInt(10),
		big.NewInt(6),
	)
	if err != nil {
		t.Fatalf("MulDiv() error = %v", err)
	}

	if actual.Cmp(big.NewInt(16)) != 0 {
		t.Fatalf(
			"MulDiv() = %s, want 16",
			actual,
		)
	}
}

func TestMulDivRoundingUp(
	t *testing.T,
) {
	t.Parallel()

	actual, err := uniswapv3.MulDivRoundingUp(
		big.NewInt(10),
		big.NewInt(10),
		big.NewInt(6),
	)
	if err != nil {
		t.Fatalf(
			"MulDivRoundingUp() error = %v",
			err,
		)
	}

	if actual.Cmp(big.NewInt(17)) != 0 {
		t.Fatalf(
			"MulDivRoundingUp() = %s, want 17",
			actual,
		)
	}
}

func TestMulDivRoundingUpDoesNotChangeExactResult(
	t *testing.T,
) {
	t.Parallel()

	actual, err := uniswapv3.MulDivRoundingUp(
		big.NewInt(10),
		big.NewInt(12),
		big.NewInt(6),
	)
	if err != nil {
		t.Fatalf(
			"MulDivRoundingUp() error = %v",
			err,
		)
	}

	if actual.Cmp(big.NewInt(20)) != 0 {
		t.Fatalf(
			"MulDivRoundingUp() = %s, want 20",
			actual,
		)
	}
}

func TestDivRoundingUp(
	t *testing.T,
) {
	t.Parallel()

	actual, err := uniswapv3.DivRoundingUp(
		big.NewInt(10),
		big.NewInt(6),
	)
	if err != nil {
		t.Fatalf(
			"DivRoundingUp() error = %v",
			err,
		)
	}

	if actual.Cmp(big.NewInt(2)) != 0 {
		t.Fatalf(
			"DivRoundingUp() = %s, want 2",
			actual,
		)
	}
}

func TestMulDivRejectsZeroDenominator(
	t *testing.T,
) {
	t.Parallel()

	_, err := uniswapv3.MulDiv(
		big.NewInt(1),
		big.NewInt(1),
		big.NewInt(0),
	)
	if err == nil {
		t.Fatal("MulDiv() expected error")
	}
}

func TestMulDivRejectsNegativeInput(
	t *testing.T,
) {
	t.Parallel()

	_, err := uniswapv3.MulDiv(
		big.NewInt(-1),
		big.NewInt(1),
		big.NewInt(1),
	)
	if err == nil {
		t.Fatal("MulDiv() expected error")
	}
}

func TestMulDivRejectsResultOutsideUint256(
	t *testing.T,
) {
	t.Parallel()

	maxUint256 := new(big.Int).Sub(
		new(big.Int).Lsh(
			big.NewInt(1),
			256,
		),
		big.NewInt(1),
	)

	_, err := uniswapv3.MulDiv(
		maxUint256,
		maxUint256,
		big.NewInt(1),
	)
	if err == nil {
		t.Fatal(
			"MulDiv() expected overflow error",
		)
	}
}
