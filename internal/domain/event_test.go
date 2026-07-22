package domain_test

import (
	"math/big"
	"testing"

	"github.com/rajabinekoo/clmm-lsis/internal/domain"
)

func TestNewSwapEventAcceptsOppositeSignedAmounts(
	t *testing.T,
) {
	t.Parallel()

	sender := mustAddress(
		t,
		"0x1111111111111111111111111111111111111111",
	)

	recipient := mustAddress(
		t,
		"0x2222222222222222222222222222222222222222",
	)

	event, err := domain.NewSwapEvent(
		sender,
		recipient,
		big.NewInt(1_000_000),
		big.NewInt(-500_000),
		big.NewInt(792281625142643375),
		big.NewInt(10_000),
		100,
	)
	if err != nil {
		t.Fatalf("NewSwapEvent() error = %v", err)
	}

	if !event.ZeroForOne() {
		t.Fatal("ZeroForOne() = false, want true")
	}
}

func TestNewSwapEventRejectsSameSignedAmounts(
	t *testing.T,
) {
	t.Parallel()

	sender := mustAddress(
		t,
		"0x1111111111111111111111111111111111111111",
	)

	recipient := mustAddress(
		t,
		"0x2222222222222222222222222222222222222222",
	)

	_, err := domain.NewSwapEvent(
		sender,
		recipient,
		big.NewInt(1_000_000),
		big.NewInt(500_000),
		big.NewInt(792281625142643375),
		big.NewInt(10_000),
		100,
	)
	if err == nil {
		t.Fatal("NewSwapEvent() expected error")
	}
}

func TestNewBurnEventRejectsInvalidRange(
	t *testing.T,
) {
	t.Parallel()

	owner := mustAddress(
		t,
		"0x3333333333333333333333333333333333333333",
	)

	_, err := domain.NewBurnEvent(
		owner,
		200,
		100,
		big.NewInt(500),
	)
	if err == nil {
		t.Fatal("NewBurnEvent() expected error")
	}
}

func mustAddress(
	t *testing.T,
	value string,
) domain.Address {
	t.Helper()

	address, err := domain.ParseAddress(value)
	if err != nil {
		t.Fatalf("ParseAddress(%q) error = %v", value, err)
	}

	return address
}
