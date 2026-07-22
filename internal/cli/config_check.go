package cli

import (
	"context"
	"flag"
	"fmt"
	"io"

	"github.com/rajabinekoo/clmm-lsis/internal/config"
)

func runConfigCheck(
	ctx context.Context,
	args []string,
	stdout io.Writer,
	stderr io.Writer,
) int {
	flags := flag.NewFlagSet("config-check", flag.ContinueOnError)
	flags.SetOutput(stderr)

	configPath := flags.String(
		"config",
		"configs/study.example.json",
		"path to the study configuration",
	)

	requireEnvironment := flags.Bool(
		"require-env",
		false,
		"also validate required runtime environment variables",
	)

	if err := flags.Parse(args); err != nil {
		return exitUsage
	}

	if flags.NArg() != 0 {
		fmt.Fprintln(
			stderr,
			"config-check does not accept positional arguments",
		)

		return exitUsage
	}

	select {
	case <-ctx.Done():
		fmt.Fprintf(stderr, "config-check canceled: %v\n", ctx.Err())
		return exitFailure
	default:
	}

	cfg, err := config.Load(*configPath)
	if err != nil {
		fmt.Fprintf(stderr, "configuration is invalid: %v\n", err)
		return exitFailure
	}

	if *requireEnvironment {
		if _, err := config.LoadRuntimeSecrets(cfg); err != nil {
			fmt.Fprintf(
				stderr,
				"runtime environment is invalid: %v\n",
				err,
			)

			return exitFailure
		}
	}

	fmt.Fprintf(
		stdout,
		"configuration valid: schema=%d chain=%d pools=%d primary-k=%d primary-horizon=%d\n",
		cfg.SchemaVersion,
		cfg.ChainID,
		len(cfg.Pools),
		cfg.StructuralStudy.PrimaryPositionLimit,
		cfg.WithdrawalStudy.PrimaryHorizonBlocks,
	)

	return exitSuccess
}
