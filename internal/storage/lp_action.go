package storage

import (
	"fmt"
	"math/big"

	"github.com/rajabinekoo/clmm-lsis/internal/domain"
)

// DomainEvent converts one legacy lp_actions row into a domain event.
//
// Legacy events use TransactionIndex=0 when the original row was not enriched
// with transaction metadata. This does not affect ordering because EventCursor
// orders logs by block number and global log index.
func (r LegacyLPActionRecord) DomainEvent() (
	domain.PoolEvent,
	error,
) {
	poolAddress, err :=
		parseRequiredStorageAddress(
			"lp action pool_address",
			r.PoolAddress,
		)
	if err != nil {
		return domain.PoolEvent{}, err
	}

	transactionHash, err :=
		parseRequiredStorageHash(
			"lp action transaction hash",
			r.TransactionHash,
		)
	if err != nil {
		return domain.PoolEvent{}, err
	}

	blockHash, err :=
		parseOptionalStorageHash(
			"lp action block hash",
			r.BlockHash,
		)
	if err != nil {
		return domain.PoolEvent{}, err
	}

	owner, exists, err :=
		parseOptionalStorageAddress(
			"lp action owner",
			r.Owner,
		)
	if err != nil {
		return domain.PoolEvent{}, err
	}

	if !exists {
		return domain.PoolEvent{}, fmt.Errorf(
			"%w: action id=%s block=%d log=%d",
			ErrMissingPositionOwner,
			r.ID,
			r.BlockNumber,
			r.LogIndex,
		)
	}

	liquidityDelta, err :=
		parseStorageBigInt(
			"lp action liquidity_delta",
			r.LiquidityDelta,
		)
	if err != nil {
		return domain.PoolEvent{}, err
	}

	transactionIndex := uint32(0)

	if r.TransactionIndex != nil {
		transactionIndex = *r.TransactionIndex
	}

	cursor := domain.EventCursor{
		BlockNumber:      r.BlockNumber,
		TransactionIndex: transactionIndex,
		LogIndex:         r.LogIndex,
	}

	var payload domain.PoolEventPayload

	switch r.Action {
	case LegacyLPActionMint:
		if liquidityDelta.Sign() <= 0 {
			return domain.PoolEvent{}, fmt.Errorf(
				"%w: mint action %s has delta %s",
				ErrInvalidLiquiditySign,
				r.ID,
				liquidityDelta,
			)
		}

		sender, exists, err :=
			parseOptionalStorageAddress(
				"lp action sender",
				r.Sender,
			)
		if err != nil {
			return domain.PoolEvent{}, err
		}

		if !exists {
			return domain.PoolEvent{}, fmt.Errorf(
				"%w: mint action %s sender is missing",
				ErrInvalidLegacyRecord,
				r.ID,
			)
		}

		mint, err := domain.NewMintEvent(
			sender,
			owner,
			r.TickLower,
			r.TickUpper,
			liquidityDelta,
		)
		if err != nil {
			return domain.PoolEvent{}, fmt.Errorf(
				"convert legacy mint %s: %w",
				r.ID,
				err,
			)
		}

		payload = mint

	case LegacyLPActionBurn:
		if liquidityDelta.Sign() >= 0 {
			return domain.PoolEvent{}, fmt.Errorf(
				"%w: burn action %s has delta %s",
				ErrInvalidLiquiditySign,
				r.ID,
				liquidityDelta,
			)
		}

		removedLiquidity := new(big.Int).Abs(
			liquidityDelta,
		)

		burn, err := domain.NewBurnEvent(
			owner,
			r.TickLower,
			r.TickUpper,
			removedLiquidity,
		)
		if err != nil {
			return domain.PoolEvent{}, fmt.Errorf(
				"convert legacy burn %s: %w",
				r.ID,
				err,
			)
		}

		payload = burn

	default:
		return domain.PoolEvent{}, fmt.Errorf(
			"%w: unsupported lp action type %d for action %s",
			ErrInvalidLegacyRecord,
			r.Action,
			r.ID,
		)
	}

	event, err := domain.NewPoolEvent(
		poolAddress,
		cursor,
		blockHash,
		transactionHash,
		payload,
	)
	if err != nil {
		return domain.PoolEvent{}, fmt.Errorf(
			"convert legacy lp action %s: %w",
			r.ID,
			err,
		)
	}

	return event, nil
}
