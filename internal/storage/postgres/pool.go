package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/rajabinekoo/clmm-lsis/internal/domain"
	"github.com/rajabinekoo/clmm-lsis/internal/storage"
)

func (r *Repository) LoadPoolRecord(
	ctx context.Context,
	address domain.Address,
) (storage.LegacyPoolRecord, error) {
	if address.IsZero() {
		return storage.LegacyPoolRecord{}, fmt.Errorf(
			"load pool record: pool address is required",
		)
	}

	var (
		record storage.LegacyPoolRecord

		token0Decimals int64
		token1Decimals int64

		feeTier      int64
		tickSpacing  int64
		createdBlock int64
	)

	err := r.db.QueryRowContext(
		ctx,
		`
			SELECT
				BTRIM(address),
				BTRIM(token0_address),
				BTRIM(token1_address),
				token0_decimals,
				token1_decimals,
				fee_tier,
				tick_spacing,
				created_block
			FROM pools
			WHERE address = $1
		`,
		address.String(),
	).Scan(
		&record.Address,
		&record.Token0Address,
		&record.Token1Address,
		&token0Decimals,
		&token1Decimals,
		&feeTier,
		&tickSpacing,
		&createdBlock,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return storage.LegacyPoolRecord{}, fmt.Errorf(
				"%w: pool %s",
				storage.ErrRecordNotFound,
				address,
			)
		}

		return storage.LegacyPoolRecord{}, fmt.Errorf(
			"load pool %s: %w",
			address,
			err,
		)
	}

	if token0Decimals < 0 ||
		token0Decimals > 255 {
		return storage.LegacyPoolRecord{}, fmt.Errorf(
			"pool %s token0 decimals are invalid: %d",
			address,
			token0Decimals,
		)
	}

	if token1Decimals < 0 ||
		token1Decimals > 255 {
		return storage.LegacyPoolRecord{}, fmt.Errorf(
			"pool %s token1 decimals are invalid: %d",
			address,
			token1Decimals,
		)
	}

	fee, err := checkedUint32(
		"pool fee tier",
		feeTier,
	)
	if err != nil {
		return storage.LegacyPoolRecord{}, err
	}

	spacing, err := checkedInt32(
		"pool tick spacing",
		tickSpacing,
	)
	if err != nil {
		return storage.LegacyPoolRecord{}, err
	}

	created, err := checkedUint64(
		"pool created block",
		createdBlock,
	)
	if err != nil {
		return storage.LegacyPoolRecord{}, err
	}

	record.Token0Decimals =
		uint8(token0Decimals)

	record.Token1Decimals =
		uint8(token1Decimals)

	record.FeeTier = fee
	record.TickSpacing = spacing
	record.CreatedBlock = created

	return record, nil
}
