package cli

import (
	"context"
	"flag"
	"fmt"
	"io"
	"time"

	"github.com/rajabinekoo/clmm-lsis/internal/config"
	"github.com/rajabinekoo/clmm-lsis/internal/database"
	postgresstorage "github.com/rajabinekoo/clmm-lsis/internal/storage/postgres"
)

// runDBMigrateSwaps applies only the additive swap-indexing schema.
//
// The command is intentionally separate from application startup. Research
// commands must never mutate database structure implicitly.
func runDBMigrateSwaps(
	parentContext context.Context,
	args []string,
	stdout io.Writer,
	stderr io.Writer,
) int {
	flags := flag.NewFlagSet(
		"db-migrate-swaps",
		flag.ContinueOnError,
	)

	flags.SetOutput(stderr)

	configPath := flags.String(
		"config",
		"configs/study.example.json",
		"path to the study configuration",
	)

	timeout := flags.Duration(
		"timeout",
		2*time.Minute,
		"maximum migration duration",
	)

	if err := flags.Parse(args); err != nil {
		return exitUsage
	}

	if flags.NArg() != 0 {
		fmt.Fprintln(
			stderr,
			"db-migrate-swaps does not accept positional arguments",
		)

		return exitUsage
	}

	if *timeout <= 0 {
		fmt.Fprintln(
			stderr,
			"db-migrate-swaps timeout must be greater than zero",
		)

		return exitUsage
	}

	ctx, cancel := context.WithTimeout(
		parentContext,
		*timeout,
	)
	defer cancel()

	cfg, err := config.Load(*configPath)
	if err != nil {
		fmt.Fprintf(
			stderr,
			"load configuration: %v\n",
			err,
		)

		return exitFailure
	}

	databaseURL, err :=
		config.LoadDatabaseURL(cfg)
	if err != nil {
		fmt.Fprintf(
			stderr,
			"load database environment: %v\n",
			err,
		)

		return exitFailure
	}

	db, err := database.OpenPostgres(
		ctx,
		databaseURL,
		database.DefaultPostgresOptions(),
	)
	if err != nil {
		fmt.Fprintf(
			stderr,
			"connect to database: %v\n",
			err,
		)

		return exitFailure
	}
	defer db.Close()

	repository, err :=
		postgresstorage.NewRepository(db)
	if err != nil {
		fmt.Fprintf(
			stderr,
			"create database repository: %v\n",
			err,
		)

		return exitFailure
	}

	before, err := repository.InspectSchema(ctx)
	if err != nil {
		fmt.Fprintf(
			stderr,
			"inspect schema before migration: %v\n",
			err,
		)

		return exitFailure
	}

	if !before.LegacyReady() {
		fmt.Fprintf(
			stderr,
			"refusing migration because legacy schema is incomplete: %v\n",
			before.MissingLegacyTables(),
		)

		return exitFailure
	}

	if before.AdditiveReady() {
		fmt.Fprintln(
			stdout,
			"swap schema already present; no migration required",
		)

		return exitSuccess
	}

	fmt.Fprintln(
		stdout,
		"applying additive swap schema...",
	)

	if err := repository.EnsureSwapSchema(ctx); err != nil {
		fmt.Fprintf(
			stderr,
			"apply additive swap schema: %v\n",
			err,
		)

		return exitFailure
	}

	after, err := repository.InspectSchema(ctx)
	if err != nil {
		fmt.Fprintf(
			stderr,
			"inspect schema after migration: %v\n",
			err,
		)

		return exitFailure
	}

	fmt.Fprintf(
		stdout,
		"swap schema ready: pool_swaps=%s pool_swap_index_ranges=%s\n",
		statusWord(after.PoolSwapsTable),
		statusWord(after.SwapIndexRangesTable),
	)

	fmt.Fprintln(
		stdout,
		"legacy tables were not modified",
	)

	return exitSuccess
}
