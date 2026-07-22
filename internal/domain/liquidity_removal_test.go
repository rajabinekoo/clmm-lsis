package domain_test

import (
	"math/big"
	"testing"

	"github.com/rajabinekoo/clmm-lsis/internal/domain"
)

func TestLiquidityRemovalPartial(
	t *testing.T,
) {
	t.Parallel()

	removal, err := domain.NewLiquidityRemoval(
		big.NewInt(1_000),
		big.NewInt(250),
	)
	if err != nil {
		t.Fatalf("NewLiquidityRemoval() error = %v", err)
	}

	if removal.IsFullRemoval() {
		t.Fatal("IsFullRemoval() = true, want false")
	}

	if removal.RemainingLiquidity().Cmp(big.NewInt(750)) != 0 {
		t.Fatalf(
			"RemainingLiquidity() = %s, want 750",
			removal.RemainingLiquidity(),
		)
	}

	expectedFraction := big.NewRat(1, 4)

	if removal.Fraction().Cmp(expectedFraction) != 0 {
		t.Fatalf(
			"Fraction() = %s, want %s",
			removal.Fraction(),
			expectedFraction,
		)
	}
}

func TestLiquidityRemovalFull(
	t *testing.T,
) {
	t.Parallel()

	removal, err := domain.NewLiquidityRemoval(
		big.NewInt(1_000),
		big.NewInt(1_000),
	)
	if err != nil {
		t.Fatalf("NewLiquidityRemoval() error = %v", err)
	}

	if !removal.IsFullRemoval() {
		t.Fatal("IsFullRemoval() = false, want true")
	}

	if removal.RemainingLiquidity().Sign() != 0 {
		t.Fatalf(
			"RemainingLiquidity() = %s, want 0",
			removal.RemainingLiquidity(),
		)
	}
}

func TestLiquidityRemovalRejectsExcessAmount(
	t *testing.T,
) {
	t.Parallel()

	_, err := domain.NewLiquidityRemoval(
		big.NewInt(1_000),
		big.NewInt(1_001),
	)
	if err == nil {
		t.Fatal("NewLiquidityRemoval() expected error")
	}
}
