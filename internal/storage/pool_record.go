package storage

import (
	"fmt"

	"github.com/rajabinekoo/clmm-lsis/internal/domain"
)

// ValidateAgainst verifies that one legacy pools row is compatible with the
// immutable pool metadata declared by the study configuration.
//
// Token addresses are validated syntactically. They are not compared with the
// configuration because the current study configuration intentionally stores
// symbols and decimals rather than token contract addresses.
func (r LegacyPoolRecord) ValidateAgainst(
	pool domain.Pool,
) error {
	if err := pool.Validate(); err != nil {
		return fmt.Errorf(
			"validate stored pool metadata: %w",
			err,
		)
	}

	address, err :=
		parseRequiredStorageAddress(
			"pools.address",
			r.Address,
		)
	if err != nil {
		return err
	}

	if address != pool.Address {
		return fmt.Errorf(
			"%w: stored pool address %s does not match configured address %s",
			ErrCheckpointMismatch,
			address,
			pool.Address,
		)
	}

	if _, err := parseRequiredStorageAddress(
		"pools.token0_address",
		r.Token0Address,
	); err != nil {
		return err
	}

	if _, err := parseRequiredStorageAddress(
		"pools.token1_address",
		r.Token1Address,
	); err != nil {
		return err
	}

	if r.Token0Decimals != pool.Token0.Decimals {
		return fmt.Errorf(
			"%w: token0 decimals stored=%d configured=%d",
			ErrCheckpointMismatch,
			r.Token0Decimals,
			pool.Token0.Decimals,
		)
	}

	if r.Token1Decimals != pool.Token1.Decimals {
		return fmt.Errorf(
			"%w: token1 decimals stored=%d configured=%d",
			ErrCheckpointMismatch,
			r.Token1Decimals,
			pool.Token1.Decimals,
		)
	}

	if r.FeeTier != pool.FeePips {
		return fmt.Errorf(
			"%w: fee tier stored=%d configured=%d",
			ErrCheckpointMismatch,
			r.FeeTier,
			pool.FeePips,
		)
	}

	if r.TickSpacing != pool.TickSpacing {
		return fmt.Errorf(
			"%w: tick spacing stored=%d configured=%d",
			ErrCheckpointMismatch,
			r.TickSpacing,
			pool.TickSpacing,
		)
	}

	if r.CreatedBlock == 0 {
		return fmt.Errorf(
			"%w: stored pool created block is zero",
			ErrInvalidLegacyRecord,
		)
	}

	return nil
}
