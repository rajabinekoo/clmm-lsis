package postgres

import (
	"strings"
	"testing"
)

func TestSplitMigrationStatements(
	t *testing.T,
) {
	t.Parallel()

	migration := `
		CREATE TABLE first_table
		(
			id BIGINT PRIMARY KEY
		);

		-- statement-breakpoint

		CREATE TABLE second_table
		(
			id BIGINT PRIMARY KEY
		);
	`

	statements, err :=
		splitMigrationStatements(migration)
	if err != nil {
		t.Fatalf(
			"splitMigrationStatements() error = %v",
			err,
		)
	}

	if len(statements) != 2 {
		t.Fatalf(
			"statement count = %d, want 2",
			len(statements),
		)
	}

	if !strings.Contains(
		statements[0],
		"first_table",
	) {
		t.Fatalf(
			"first statement = %q",
			statements[0],
		)
	}

	if !strings.Contains(
		statements[1],
		"second_table",
	) {
		t.Fatalf(
			"second statement = %q",
			statements[1],
		)
	}
}

func TestSplitEmbeddedSwapMigration(
	t *testing.T,
) {
	t.Parallel()

	migration, err := swapMigrationFS.ReadFile(
		swapSchemaMigrationPath,
	)
	if err != nil {
		t.Fatalf(
			"ReadFile() error = %v",
			err,
		)
	}

	statements, err :=
		splitMigrationStatements(
			string(migration),
		)
	if err != nil {
		t.Fatalf(
			"splitMigrationStatements() error = %v",
			err,
		)
	}

	if len(statements) != 2 {
		t.Fatalf(
			"embedded migration statement count = %d, want 2",
			len(statements),
		)
	}

	if !strings.Contains(
		statements[0],
		"pool_swaps",
	) {
		t.Fatal(
			"first migration statement does not create pool_swaps",
		)
	}

	if !strings.Contains(
		statements[1],
		"pool_swap_index_ranges",
	) {
		t.Fatal(
			"second migration statement does not create pool_swap_index_ranges",
		)
	}
}
