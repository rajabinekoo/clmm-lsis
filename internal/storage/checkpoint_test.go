package storage_test

import (
	"errors"
	"fmt"
	"math/big"
	"testing"

	"github.com/rajabinekoo/clmm-lsis/internal/domain"
	"github.com/rajabinekoo/clmm-lsis/internal/storage"
	"github.com/rajabinekoo/clmm-lsis/internal/uniswapv3"
)

func TestBuildLegacyCheckpointReconstructsPositionsAndTicks(
	t *testing.T,
) {
	t.Parallel()

	pool := newStorageTestPool(t)

	ownerA := storageAddressString(1)
	ownerB := storageAddressString(2)
	sender := storageAddressString(3)

	actions := []storage.LegacyLPActionRecord{
		{
			ID:              "mint-a",
			PoolAddress:     pool.Address.String(),
			Action:          storage.LegacyLPActionMint,
			TransactionHash: storageHashString(1),
			BlockNumber:     100,
			LogIndex:        1,
			Owner:           &ownerA,
			Sender:          &sender,
			TickLower:       -100,
			TickUpper:       100,
			LiquidityDelta:  "1000",
		},
		{
			ID:              "mint-b",
			PoolAddress:     pool.Address.String(),
			Action:          storage.LegacyLPActionMint,
			TransactionHash: storageHashString(2),
			BlockNumber:     101,
			LogIndex:        2,
			Owner:           &ownerB,
			Sender:          &sender,
			TickLower:       -200,
			TickUpper:       -100,
			LiquidityDelta:  "500",
		},
		{
			ID:              "burn-a",
			PoolAddress:     pool.Address.String(),
			Action:          storage.LegacyLPActionBurn,
			TransactionHash: storageHashString(3),
			BlockNumber:     102,
			LogIndex:        3,
			Owner:           &ownerA,
			TickLower:       -100,
			TickUpper:       100,
			LiquidityDelta:  "-250",
		},
	}

	currentTick := int32(-50)

	snapshot, err := storage.BuildLegacyCheckpoint(
		pool,
		storage.LegacyPoolSnapshotRecord{
			PoolAddress: pool.Address.String(),
			BlockNumber: 102,

			SqrtPriceX96: mustStorageSqrtPrice(
				t,
				currentTick,
			).String(),

			CurrentTick: &currentTick,

			// Only owner A's remaining 750 liquidity is active at tick -50.
			ActiveLiquidity: "750",
		},
		actions,
	)
	if err != nil {
		t.Fatalf(
			"BuildLegacyCheckpoint() error = %v",
			err,
		)
	}

	if snapshot.Reference().Boundary() !=
		domain.SnapshotBlockEnd {
		t.Fatalf(
			"boundary = %s, want block_end",
			snapshot.Reference().Boundary(),
		)
	}

	if snapshot.Reference().BlockNumber() != 102 {
		t.Fatalf(
			"block = %d, want 102",
			snapshot.Reference().BlockNumber(),
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

	sharedTick, exists := snapshot.Tick(-100)
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

	assertStorageBigIntEqual(
		t,
		snapshot.ActiveLiquidity(),
		big.NewInt(750),
	)

	ownerAAddress, err :=
		domain.ParseAddress(ownerA)
	if err != nil {
		t.Fatalf(
			"ParseAddress(ownerA) error = %v",
			err,
		)
	}

	positionA, exists := snapshot.Position(
		domain.CorePositionKey{
			PoolAddress: pool.Address,
			Owner:       ownerAAddress,
			TickLower:   -100,
			TickUpper:   100,
		},
	)

	if !exists {
		t.Fatal(
			"position A not found",
		)
	}

	assertStorageBigIntEqual(
		t,
		positionA.Liquidity(),
		big.NewInt(750),
	)
}

func TestBuildLegacyCheckpointRejectsMissingOwner(
	t *testing.T,
) {
	t.Parallel()

	pool := newStorageTestPool(t)
	currentTick := int32(0)

	_, err := storage.BuildLegacyCheckpoint(
		pool,
		storage.LegacyPoolSnapshotRecord{
			PoolAddress:     pool.Address.String(),
			BlockNumber:     100,
			SqrtPriceX96:    mustStorageSqrtPrice(t, 0).String(),
			CurrentTick:     &currentTick,
			ActiveLiquidity: "1000",
		},
		[]storage.LegacyLPActionRecord{
			{
				ID:              "missing-owner",
				PoolAddress:     pool.Address.String(),
				Action:          storage.LegacyLPActionMint,
				TransactionHash: storageHashString(1),
				BlockNumber:     100,
				LogIndex:        1,
				TickLower:       -100,
				TickUpper:       100,
				LiquidityDelta:  "1000",
			},
		},
	)

	if !errors.Is(
		err,
		storage.ErrMissingPositionOwner,
	) {
		t.Fatalf(
			"BuildLegacyCheckpoint() error = %v, want ErrMissingPositionOwner",
			err,
		)
	}
}

func TestBuildLegacyCheckpointRejectsScalarLiquidityMismatch(
	t *testing.T,
) {
	t.Parallel()

	pool := newStorageTestPool(t)
	currentTick := int32(0)

	owner := storageAddressString(1)

	_, err := storage.BuildLegacyCheckpoint(
		pool,
		storage.LegacyPoolSnapshotRecord{
			PoolAddress:     pool.Address.String(),
			BlockNumber:     100,
			SqrtPriceX96:    mustStorageSqrtPrice(t, 0).String(),
			CurrentTick:     &currentTick,
			ActiveLiquidity: "999",
		},
		[]storage.LegacyLPActionRecord{
			{
				ID:              "mint",
				PoolAddress:     pool.Address.String(),
				Action:          storage.LegacyLPActionMint,
				TransactionHash: storageHashString(1),
				BlockNumber:     100,
				LogIndex:        1,
				Owner:           &owner,
				TickLower:       -100,
				TickUpper:       100,
				LiquidityDelta:  "1000",
			},
		},
	)

	if !errors.Is(
		err,
		storage.ErrCheckpointMismatch,
	) {
		t.Fatalf(
			"BuildLegacyCheckpoint() error = %v, want ErrCheckpointMismatch",
			err,
		)
	}
}

func newStorageTestPool(
	t *testing.T,
) domain.Pool {
	t.Helper()

	address, err := domain.ParseAddress(
		"0xaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
	)
	if err != nil {
		t.Fatalf(
			"ParseAddress() error = %v",
			err,
		)
	}

	pool := domain.Pool{
		Name:        "storage_test_pool",
		Address:     address,
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
			"pool.Validate() error = %v",
			err,
		)
	}

	return pool
}

func storageAddressString(
	suffix uint64,
) string {
	return fmt.Sprintf(
		"0x%040x",
		suffix,
	)
}

func storageHashString(
	suffix uint64,
) string {
	return fmt.Sprintf(
		"0x%064x",
		suffix,
	)
}

func mustStorageSqrtPrice(
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

func assertStorageBigIntEqual(
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
