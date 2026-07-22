package storage_test

import (
	"math/big"
	"testing"

	"github.com/rajabinekoo/clmm-lsis/internal/domain"
	"github.com/rajabinekoo/clmm-lsis/internal/storage"
)

func TestBuildCheckpointFromAggregatedPositions(
	t *testing.T,
) {
	t.Parallel()

	pool := newStorageTestPool(t)
	currentTick := int32(-50)

	ownerA := storageAddressString(1)
	ownerB := storageAddressString(2)

	snapshot, err :=
		storage.BuildCheckpointFromAggregatedPositions(
			pool,
			storage.LegacyPoolSnapshotRecord{
				PoolAddress: pool.Address.String(),
				BlockNumber: 1_000,

				SqrtPriceX96: mustStorageSqrtPrice(
					t,
					currentTick,
				).String(),

				CurrentTick: &currentTick,

				ActiveLiquidity: "750",
			},
			[]storage.AggregatedPositionRecord{
				{
					Owner:     ownerA,
					TickLower: -100,
					TickUpper: 100,
					Liquidity: "750",
				},
				{
					Owner:     ownerB,
					TickLower: -200,
					TickUpper: -100,
					Liquidity: "500",
				},
			},
		)
	if err != nil {
		t.Fatalf(
			"BuildCheckpointFromAggregatedPositions() error = %v",
			err,
		)
	}

	if len(snapshot.Positions()) != 2 {
		t.Fatalf(
			"position count = %d, want 2",
			len(snapshot.Positions()),
		)
	}

	if len(snapshot.Ticks()) != 3 {
		t.Fatalf(
			"tick count = %d, want 3",
			len(snapshot.Ticks()),
		)
	}

	assertStorageBigIntEqual(
		t,
		snapshot.ActiveLiquidity(),
		big.NewInt(750),
	)

	sharedTick, exists :=
		snapshot.Tick(-100)

	if !exists {
		t.Fatal(
			"shared tick -100 not found",
		)
	}

	assertStorageBigIntEqual(
		t,
		sharedTick.LiquidityGross(),
		big.NewInt(1_250),
	)

	assertStorageBigIntEqual(
		t,
		sharedTick.LiquidityNet(),
		big.NewInt(250),
	)

	ownerAddress, err :=
		domain.ParseAddress(ownerA)
	if err != nil {
		t.Fatalf(
			"ParseAddress() error = %v",
			err,
		)
	}

	position, exists := snapshot.Position(
		domain.CorePositionKey{
			PoolAddress: pool.Address,
			Owner:       ownerAddress,
			TickLower:   -100,
			TickUpper:   100,
		},
	)

	if !exists {
		t.Fatal(
			"owner A position not found",
		)
	}

	assertStorageBigIntEqual(
		t,
		position.Liquidity(),
		big.NewInt(750),
	)
}

func TestBuildCheckpointFromAggregatedPositionsRejectsDuplicateKey(
	t *testing.T,
) {
	t.Parallel()

	pool := newStorageTestPool(t)
	currentTick := int32(0)
	owner := storageAddressString(1)

	_, err :=
		storage.BuildCheckpointFromAggregatedPositions(
			pool,
			storage.LegacyPoolSnapshotRecord{
				PoolAddress:     pool.Address.String(),
				BlockNumber:     1_000,
				SqrtPriceX96:    mustStorageSqrtPrice(t, 0).String(),
				CurrentTick:     &currentTick,
				ActiveLiquidity: "2",
			},
			[]storage.AggregatedPositionRecord{
				{
					Owner:     owner,
					TickLower: -100,
					TickUpper: 100,
					Liquidity: "1",
				},
				{
					Owner:     owner,
					TickLower: -100,
					TickUpper: 100,
					Liquidity: "1",
				},
			},
		)
	if err == nil {
		t.Fatal(
			"BuildCheckpointFromAggregatedPositions() expected duplicate error",
		)
	}
}
