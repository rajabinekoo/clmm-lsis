package postgres

import (
	"context"
	"embed"
	"fmt"
	"strings"

	"github.com/rajabinekoo/clmm-lsis/internal/storage"
)

const (
	swapSchemaMigrationPath = "migrations/000002_create_pool_swaps.sql"

	migrationStatementBreakpoint = "-- statement-breakpoint"
)

// swapMigrationFS embeds the additive schema into the CLI binary.
//
// The migration remains available even when the executable is run outside the
// repository working directory.
//
//go:embed migrations/000002_create_pool_swaps.sql
var swapMigrationFS embed.FS

// EnsureSwapSchema creates only the additive tables required for swap
// indexing.
//
// Existing legacy tables are checked before any write occurs. The migration
// contains no ALTER, DROP, DELETE, UPDATE or TRUNCATE statement.
func (r *Repository) EnsureSwapSchema(
	ctx context.Context,
) error {
	if err := r.RequireLegacySchema(ctx); err != nil {
		return fmt.Errorf(
			"apply swap schema: %w",
			err,
		)
	}

	migration, err := swapMigrationFS.ReadFile(
		swapSchemaMigrationPath,
	)
	if err != nil {
		return fmt.Errorf(
			"read embedded swap migration: %w",
			err,
		)
	}

	statements, err :=
		splitMigrationStatements(
			string(migration),
		)
	if err != nil {
		return fmt.Errorf(
			"parse swap migration: %w",
			err,
		)
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf(
			"apply swap schema: begin transaction: %w",
			err,
		)
	}

	defer func() {
		_ = tx.Rollback()
	}()

	for index, statement := range statements {
		if _, err := tx.ExecContext(
			ctx,
			statement,
		); err != nil {
			return fmt.Errorf(
				"apply swap schema statement %d: %w",
				index+1,
				err,
			)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf(
			"apply swap schema: commit transaction: %w",
			err,
		)
	}

	report, err := r.InspectSchema(ctx)
	if err != nil {
		return fmt.Errorf(
			"verify swap schema: %w",
			err,
		)
	}

	if !report.AdditiveReady() {
		return fmt.Errorf(
			"%w: swap schema migration completed but required tables are missing",
			storage.ErrSchemaIncompatible,
		)
	}

	return nil
}

func splitMigrationStatements(
	migration string,
) ([]string, error) {
	parts := strings.Split(
		migration,
		migrationStatementBreakpoint,
	)

	statements := make(
		[]string,
		0,
		len(parts),
	)

	for _, part := range parts {
		statement := strings.TrimSpace(part)

		if statement == "" {
			continue
		}

		statements = append(
			statements,
			statement,
		)
	}

	if len(statements) == 0 {
		return nil, fmt.Errorf(
			"migration contains no executable statements",
		)
	}

	return statements, nil
}
