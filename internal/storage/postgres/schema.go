package postgres

import (
	"context"
	"fmt"
	"sort"

	"github.com/rajabinekoo/clmm-lsis/internal/storage"
)

// SchemaReport describes which legacy and additive tables currently exist.
type SchemaReport struct {
	PoolsTable              bool
	LPActionsTable          bool
	PoolSnapshotsTable      bool
	IndexerCheckpointsTable bool

	PoolSwapsTable       bool
	SwapIndexRangesTable bool
}

func (r SchemaReport) LegacyReady() bool {
	return r.PoolsTable &&
		r.LPActionsTable &&
		r.PoolSnapshotsTable &&
		r.IndexerCheckpointsTable
}

func (r SchemaReport) AdditiveReady() bool {
	return r.PoolSwapsTable &&
		r.SwapIndexRangesTable
}

func (r SchemaReport) MissingLegacyTables() []string {
	missing := make([]string, 0, 4)

	if !r.PoolsTable {
		missing = append(
			missing,
			"pools",
		)
	}

	if !r.LPActionsTable {
		missing = append(
			missing,
			"lp_actions",
		)
	}

	if !r.PoolSnapshotsTable {
		missing = append(
			missing,
			"pool_snapshots",
		)
	}

	if !r.IndexerCheckpointsTable {
		missing = append(
			missing,
			"indexer_checkpoints",
		)
	}

	sort.Strings(missing)

	return missing
}

func (r SchemaReport) MissingAdditiveTables() []string {
	missing := make([]string, 0, 2)

	if !r.PoolSwapsTable {
		missing = append(
			missing,
			"pool_swaps",
		)
	}

	if !r.SwapIndexRangesTable {
		missing = append(
			missing,
			"pool_swap_index_ranges",
		)
	}

	sort.Strings(missing)

	return missing
}

func (r *Repository) InspectSchema(
	ctx context.Context,
) (SchemaReport, error) {
	pools, err := r.tableExists(
		ctx,
		"pools",
	)
	if err != nil {
		return SchemaReport{}, err
	}

	lpActions, err := r.tableExists(
		ctx,
		"lp_actions",
	)
	if err != nil {
		return SchemaReport{}, err
	}

	poolSnapshots, err :=
		r.tableExists(
			ctx,
			"pool_snapshots",
		)
	if err != nil {
		return SchemaReport{}, err
	}

	indexerCheckpoints, err :=
		r.tableExists(
			ctx,
			"indexer_checkpoints",
		)
	if err != nil {
		return SchemaReport{}, err
	}

	poolSwaps, err := r.tableExists(
		ctx,
		"pool_swaps",
	)
	if err != nil {
		return SchemaReport{}, err
	}

	swapIndexRanges, err :=
		r.tableExists(
			ctx,
			"pool_swap_index_ranges",
		)
	if err != nil {
		return SchemaReport{}, err
	}

	return SchemaReport{
		PoolsTable: pools,

		LPActionsTable: lpActions,

		PoolSnapshotsTable: poolSnapshots,

		IndexerCheckpointsTable: indexerCheckpoints,

		PoolSwapsTable: poolSwaps,

		SwapIndexRangesTable: swapIndexRanges,
	}, nil
}

func (r *Repository) RequireLegacySchema(
	ctx context.Context,
) error {
	report, err := r.InspectSchema(ctx)
	if err != nil {
		return err
	}

	if report.LegacyReady() {
		return nil
	}

	return fmt.Errorf(
		"%w: missing legacy tables: %v",
		storage.ErrSchemaIncompatible,
		report.MissingLegacyTables(),
	)
}

func (r *Repository) RequireSwapSchema(
	ctx context.Context,
) error {
	report, err := r.InspectSchema(ctx)
	if err != nil {
		return err
	}

	if report.AdditiveReady() {
		return nil
	}

	return fmt.Errorf(
		"%w: missing additive tables: %v",
		storage.ErrSchemaIncompatible,
		report.MissingAdditiveTables(),
	)
}

func (r *Repository) tableExists(
	ctx context.Context,
	tableName string,
) (bool, error) {
	relationName :=
		r.schema + "." + tableName

	var exists bool

	err := r.db.QueryRowContext(
		ctx,
		`
			SELECT to_regclass($1) IS NOT NULL
		`,
		relationName,
	).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf(
			"inspect PostgreSQL table %s: %w",
			relationName,
			err,
		)
	}

	return exists, nil
}
