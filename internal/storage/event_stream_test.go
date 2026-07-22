package storage_test

import (
	"errors"
	"testing"

	"github.com/rajabinekoo/clmm-lsis/internal/domain"
	"github.com/rajabinekoo/clmm-lsis/internal/storage"
)

func TestBuildOrderedEventStreamInterleavesLiquidityAndSwaps(
	t *testing.T,
) {
	t.Parallel()

	pool := newStorageTestPool(t)

	owner := storageAddressString(1)
	sender := storageAddressString(2)
	recipient := storageAddressString(3)

	events, err := storage.BuildOrderedEventStream(
		[]storage.LegacyLPActionRecord{
			{
				ID:              "mint",
				PoolAddress:     pool.Address.String(),
				Action:          storage.LegacyLPActionMint,
				TransactionHash: storageHashString(10),
				BlockNumber:     100,
				LogIndex:        2,
				Owner:           &owner,
				Sender:          &sender,
				TickLower:       -100,
				TickUpper:       100,
				LiquidityDelta:  "1000",
			},
			{
				ID:              "burn",
				PoolAddress:     pool.Address.String(),
				Action:          storage.LegacyLPActionBurn,
				TransactionHash: storageHashString(12),
				BlockNumber:     100,
				LogIndex:        8,
				Owner:           &owner,
				TickLower:       -100,
				TickUpper:       100,
				LiquidityDelta:  "-250",
			},
		},
		[]storage.SwapRecord{
			{
				PoolAddress:      pool.Address.String(),
				BlockNumber:      100,
				BlockHash:        storageHashString(100),
				TransactionHash:  storageHashString(11),
				TransactionIndex: 4,
				LogIndex:         5,
				Sender:           sender,
				Recipient:        recipient,
				Amount0Raw:       "1000",
				Amount1Raw:       "-900",
				SqrtPriceX96:     mustStorageSqrtPrice(t, -10).String(),
				ActiveLiquidity:  "1000",
				Tick:             -10,
			},
		},
	)
	if err != nil {
		t.Fatalf(
			"BuildOrderedEventStream() error = %v",
			err,
		)
	}

	if len(events) != 3 {
		t.Fatalf(
			"event count = %d, want 3",
			len(events),
		)
	}

	expectedTypes := []domain.PoolEventType{
		domain.PoolEventMint,
		domain.PoolEventSwap,
		domain.PoolEventBurn,
	}

	expectedLogs := []uint32{
		2,
		5,
		8,
	}

	for index := range expectedTypes {
		if events[index].Type() !=
			expectedTypes[index] {
			t.Fatalf(
				"events[%d].Type() = %s, want %s",
				index,
				events[index].Type(),
				expectedTypes[index],
			)
		}

		if events[index].Cursor().LogIndex !=
			expectedLogs[index] {
			t.Fatalf(
				"events[%d].LogIndex = %d, want %d",
				index,
				events[index].Cursor().LogIndex,
				expectedLogs[index],
			)
		}
	}

	if events[0].BlockHashKnown() {
		t.Fatal(
			"legacy mint unexpectedly has known block hash",
		)
	}

	if !events[1].BlockHashKnown() {
		t.Fatal(
			"stored swap block hash is unknown",
		)
	}
}

func TestBuildOrderedEventStreamRejectsDuplicateLogCursor(
	t *testing.T,
) {
	t.Parallel()

	pool := newStorageTestPool(t)

	owner := storageAddressString(1)
	sender := storageAddressString(2)
	recipient := storageAddressString(3)

	_, err := storage.BuildOrderedEventStream(
		[]storage.LegacyLPActionRecord{
			{
				ID:              "mint",
				PoolAddress:     pool.Address.String(),
				Action:          storage.LegacyLPActionMint,
				TransactionHash: storageHashString(10),
				BlockNumber:     100,
				LogIndex:        5,
				Owner:           &owner,
				Sender:          &sender,
				TickLower:       -100,
				TickUpper:       100,
				LiquidityDelta:  "1000",
			},
		},
		[]storage.SwapRecord{
			{
				PoolAddress:      pool.Address.String(),
				BlockNumber:      100,
				BlockHash:        storageHashString(100),
				TransactionHash:  storageHashString(11),
				TransactionIndex: 2,
				LogIndex:         5,
				Sender:           sender,
				Recipient:        recipient,
				Amount0Raw:       "1000",
				Amount1Raw:       "-900",
				SqrtPriceX96:     mustStorageSqrtPrice(t, -10).String(),
				ActiveLiquidity:  "1000",
				Tick:             -10,
			},
		},
	)

	if !errors.Is(
		err,
		storage.ErrDuplicateEventCursor,
	) {
		t.Fatalf(
			"BuildOrderedEventStream() error = %v, want ErrDuplicateEventCursor",
			err,
		)
	}
}

func TestLegacyBurnConversionRejectsPositiveDelta(
	t *testing.T,
) {
	t.Parallel()

	pool := newStorageTestPool(t)
	owner := storageAddressString(1)

	_, err := (storage.LegacyLPActionRecord{
		ID:              "invalid-burn",
		PoolAddress:     pool.Address.String(),
		Action:          storage.LegacyLPActionBurn,
		TransactionHash: storageHashString(1),
		BlockNumber:     100,
		LogIndex:        1,
		Owner:           &owner,
		TickLower:       -100,
		TickUpper:       100,
		LiquidityDelta:  "250",
	}).DomainEvent()

	if !errors.Is(
		err,
		storage.ErrInvalidLiquiditySign,
	) {
		t.Fatalf(
			"DomainEvent() error = %v, want ErrInvalidLiquiditySign",
			err,
		)
	}
}
