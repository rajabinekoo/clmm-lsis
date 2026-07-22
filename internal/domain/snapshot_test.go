package domain_test

import (
	"math/big"
	"testing"

	"github.com/rajabinekoo/clmm-lsis/internal/domain"
)

func TestNewPoolSnapshotValidatesDerivedAccounting(
	t *testing.T,
) {
	t.Parallel()

	pool := mustAddress(
		t,
		"0xaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
	)

	ownerA := mustAddress(
		t,
		"0x1111111111111111111111111111111111111111",
	)

	ownerB := mustAddress(
		t,
		"0x2222222222222222222222222222222222222222",
	)

	ownerC := mustAddress(
		t,
		"0x3333333333333333333333333333333333333333",
	)

	positions := []domain.CorePosition{
		mustPosition(
			t,
			domain.CorePositionKey{
				PoolAddress: pool,
				Owner:       ownerA,
				TickLower:   100,
				TickUpper:   200,
			},
			400,
		),
		mustPosition(
			t,
			domain.CorePositionKey{
				PoolAddress: pool,
				Owner:       ownerB,
				TickLower:   140,
				TickUpper:   170,
			},
			300,
		),
		mustPosition(
			t,
			domain.CorePositionKey{
				PoolAddress: pool,
				Owner:       ownerC,
				TickLower:   50,
				TickUpper:   300,
			},
			200,
		),
	}

	ticks := []domain.TickState{
		mustTick(t, 50, 200, 200),
		mustTick(t, 100, 400, 400),
		mustTick(t, 140, 300, 300),
		mustTick(t, 170, 300, -300),
		mustTick(t, 200, 400, -400),
		mustTick(t, 300, 200, -200),
	}

	reference, err := domain.NewBlockEndSnapshotReference(1_000)
	if err != nil {
		t.Fatalf(
			"NewBlockEndSnapshotReference() error = %v",
			err,
		)
	}

	snapshot, err := domain.NewPoolSnapshot(
		pool,
		reference,
		big.NewInt(792281625142643375),
		150,
		big.NewInt(900),
		ticks,
		positions,
	)
	if err != nil {
		t.Fatalf("NewPoolSnapshot() error = %v", err)
	}

	if len(snapshot.ActivePositions()) != 3 {
		t.Fatalf(
			"len(ActivePositions()) = %d, want 3",
			len(snapshot.ActivePositions()),
		)
	}

	if snapshot.ActiveLiquidity().Cmp(big.NewInt(900)) != 0 {
		t.Fatalf(
			"ActiveLiquidity() = %s, want 900",
			snapshot.ActiveLiquidity(),
		)
	}
}

func TestNewPoolSnapshotRejectsIncorrectActiveLiquidity(
	t *testing.T,
) {
	t.Parallel()

	pool := mustAddress(
		t,
		"0xaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
	)

	owner := mustAddress(
		t,
		"0x1111111111111111111111111111111111111111",
	)

	position := mustPosition(
		t,
		domain.CorePositionKey{
			PoolAddress: pool,
			Owner:       owner,
			TickLower:   100,
			TickUpper:   200,
		},
		400,
	)

	ticks := []domain.TickState{
		mustTick(t, 100, 400, 400),
		mustTick(t, 200, 400, -400),
	}

	reference, err := domain.NewBlockEndSnapshotReference(1_000)
	if err != nil {
		t.Fatalf(
			"NewBlockEndSnapshotReference() error = %v",
			err,
		)
	}

	_, err = domain.NewPoolSnapshot(
		pool,
		reference,
		big.NewInt(792281625142643375),
		150,
		big.NewInt(399),
		ticks,
		[]domain.CorePosition{position},
	)
	if err == nil {
		t.Fatal("NewPoolSnapshot() expected accounting error")
	}
}

func TestSnapshotAccessorsDoNotExposeMutableBigInts(
	t *testing.T,
) {
	t.Parallel()

	pool := mustAddress(
		t,
		"0xaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
	)

	owner := mustAddress(
		t,
		"0x1111111111111111111111111111111111111111",
	)

	position := mustPosition(
		t,
		domain.CorePositionKey{
			PoolAddress: pool,
			Owner:       owner,
			TickLower:   100,
			TickUpper:   200,
		},
		400,
	)

	ticks := []domain.TickState{
		mustTick(t, 100, 400, 400),
		mustTick(t, 200, 400, -400),
	}

	reference, err := domain.NewBlockEndSnapshotReference(1_000)
	if err != nil {
		t.Fatalf(
			"NewBlockEndSnapshotReference() error = %v",
			err,
		)
	}

	snapshot, err := domain.NewPoolSnapshot(
		pool,
		reference,
		big.NewInt(792281625142643375),
		150,
		big.NewInt(400),
		ticks,
		[]domain.CorePosition{position},
	)
	if err != nil {
		t.Fatalf("NewPoolSnapshot() error = %v", err)
	}

	exposed := snapshot.ActiveLiquidity()
	exposed.SetInt64(1)

	if snapshot.ActiveLiquidity().Cmp(big.NewInt(400)) != 0 {
		t.Fatal(
			"mutating returned ActiveLiquidity changed snapshot state",
		)
	}
}

func mustPosition(
	t *testing.T,
	key domain.CorePositionKey,
	liquidity int64,
) domain.CorePosition {
	t.Helper()

	position, err := domain.NewCorePosition(
		key,
		big.NewInt(liquidity),
	)
	if err != nil {
		t.Fatalf("NewCorePosition() error = %v", err)
	}

	return position
}

func mustTick(
	t *testing.T,
	index int32,
	gross int64,
	net int64,
) domain.TickState {
	t.Helper()

	tick, err := domain.NewTickState(
		index,
		big.NewInt(gross),
		big.NewInt(net),
	)
	if err != nil {
		t.Fatalf("NewTickState() error = %v", err)
	}

	return tick
}
