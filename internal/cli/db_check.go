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

func runDBCheck(
	parentContext context.Context,
	args []string,
	stdout io.Writer,
	stderr io.Writer,
) int {
	flags := flag.NewFlagSet(
		"db-check",
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
		"maximum duration of the database compatibility check",
	)

	if err := flags.Parse(args); err != nil {
		return exitUsage
	}

	if flags.NArg() != 0 {
		fmt.Fprintln(
			stderr,
			"db-check does not accept positional arguments",
		)

		return exitUsage
	}

	if *timeout <= 0 {
		fmt.Fprintln(
			stderr,
			"db-check timeout must be greater than zero",
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

	schema, err := repository.InspectSchema(ctx)
	if err != nil {
		fmt.Fprintf(
			stderr,
			"inspect database schema: %v\n",
			err,
		)

		return exitFailure
	}

	fmt.Fprintf(
		stdout,
		"legacy schema: pools=%s lp_actions=%s pool_snapshots=%s indexer_checkpoints=%s\n",
		statusWord(schema.PoolsTable),
		statusWord(schema.LPActionsTable),
		statusWord(schema.PoolSnapshotsTable),
		statusWord(schema.IndexerCheckpointsTable),
	)

	if schema.PoolSwapsTable {
		fmt.Fprintln(
			stdout,
			"additive schema: pool_swaps=present",
		)
	} else {
		fmt.Fprintln(
			stdout,
			"additive schema: pool_swaps=missing (expected before swap migration)",
		)
	}

	if !schema.LegacyReady() {
		fmt.Fprintf(
			stderr,
			"database schema is missing required legacy tables: %v\n",
			schema.MissingLegacyTables(),
		)

		return exitFailure
	}

	failedPools := 0

	for _, poolConfig := range cfg.Pools {
		pool, err := poolConfig.DomainPool()
		if err != nil {
			fmt.Fprintf(
				stderr,
				"pool %s configuration error: %v\n",
				poolConfig.Name,
				err,
			)

			failedPools++

			continue
		}

		record, err := repository.LoadPoolRecord(
			ctx,
			pool.Address,
		)
		if err != nil {
			fmt.Fprintf(
				stderr,
				"pool %s metadata error: %v\n",
				pool.Name,
				err,
			)

			failedPools++

			continue
		}

		if err := record.ValidateAgainst(pool); err != nil {
			fmt.Fprintf(
				stderr,
				"pool %s metadata mismatch: %v\n",
				pool.Name,
				err,
			)

			failedPools++

			continue
		}

		stats, err :=
			repository.LoadLegacyPoolStats(
				ctx,
				pool.Address,
				poolConfig.StructuralReferenceBlock,
			)
		if err != nil {
			fmt.Fprintf(
				stderr,
				"pool %s statistics error: %v\n",
				pool.Name,
				err,
			)

			failedPools++

			continue
		}

		checkpoint, err :=
			repository.LoadLatestCheckpoint(
				ctx,
				pool,
				poolConfig.StructuralReferenceBlock,
			)
		if err != nil {
			fmt.Fprintf(
				stderr,
				"pool %s checkpoint error: %v\n",
				pool.Name,
				err,
			)

			failedPools++

			continue
		}

		fmt.Fprintf(
			stdout,
			"pool %-16s metadata=ok actions=%d missing_owners=%d snapshots=%d checkpoint_block=%d positions=%d ticks=%d\n",
			pool.Name,
			stats.LPActionCount,
			stats.MissingOwnerCount,
			stats.SnapshotCount,
			checkpoint.Reference().BlockNumber(),
			len(checkpoint.Positions()),
			len(checkpoint.Ticks()),
		)
	}

	if failedPools > 0 {
		fmt.Fprintf(
			stderr,
			"database compatibility failed for %d pool(s)\n",
			failedPools,
		)

		return exitFailure
	}

	fmt.Fprintf(
		stdout,
		"database compatibility valid: pools=%d\n",
		len(cfg.Pools),
	)

	return exitSuccess
}

func statusWord(
	value bool,
) string {
	if value {
		return "ok"
	}

	return "missing"
}
