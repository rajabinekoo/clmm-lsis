package storage

import (
	"fmt"

	"github.com/rajabinekoo/clmm-lsis/internal/domain"
)

// DomainEvent converts one append-only pool_swaps row into a SwapEvent.
func (r SwapRecord) DomainEvent() (
	domain.PoolEvent,
	error,
) {
	poolAddress, err :=
		parseRequiredStorageAddress(
			"swap pool_address",
			r.PoolAddress,
		)
	if err != nil {
		return domain.PoolEvent{}, err
	}

	blockHash, err :=
		parseRequiredStorageHash(
			"swap block_hash",
			r.BlockHash,
		)
	if err != nil {
		return domain.PoolEvent{}, err
	}

	transactionHash, err :=
		parseRequiredStorageHash(
			"swap transaction_hash",
			r.TransactionHash,
		)
	if err != nil {
		return domain.PoolEvent{}, err
	}

	sender, err :=
		parseRequiredStorageAddress(
			"swap sender",
			r.Sender,
		)
	if err != nil {
		return domain.PoolEvent{}, err
	}

	recipient, err :=
		parseRequiredStorageAddress(
			"swap recipient",
			r.Recipient,
		)
	if err != nil {
		return domain.PoolEvent{}, err
	}

	amount0, err :=
		parseStorageBigInt(
			"swap amount0_raw",
			r.Amount0Raw,
		)
	if err != nil {
		return domain.PoolEvent{}, err
	}

	amount1, err :=
		parseStorageBigInt(
			"swap amount1_raw",
			r.Amount1Raw,
		)
	if err != nil {
		return domain.PoolEvent{}, err
	}

	sqrtPriceX96, err :=
		parseStorageBigInt(
			"swap sqrt_price_x96",
			r.SqrtPriceX96,
		)
	if err != nil {
		return domain.PoolEvent{}, err
	}

	activeLiquidity, err :=
		parseStorageBigInt(
			"swap active_liquidity",
			r.ActiveLiquidity,
		)
	if err != nil {
		return domain.PoolEvent{}, err
	}

	payload, err := domain.NewSwapEvent(
		sender,
		recipient,
		amount0,
		amount1,
		sqrtPriceX96,
		activeLiquidity,
		r.Tick,
	)
	if err != nil {
		return domain.PoolEvent{}, fmt.Errorf(
			"convert stored swap at block %d log %d: %w",
			r.BlockNumber,
			r.LogIndex,
			err,
		)
	}

	event, err := domain.NewPoolEvent(
		poolAddress,
		domain.EventCursor{
			BlockNumber:      r.BlockNumber,
			TransactionIndex: r.TransactionIndex,
			LogIndex:         r.LogIndex,
		},
		blockHash,
		transactionHash,
		payload,
	)
	if err != nil {
		return domain.PoolEvent{}, fmt.Errorf(
			"convert stored swap at block %d log %d: %w",
			r.BlockNumber,
			r.LogIndex,
			err,
		)
	}

	return event, nil
}
