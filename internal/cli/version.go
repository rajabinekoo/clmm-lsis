package cli

import (
	"flag"
	"fmt"
	"io"

	"github.com/rajabinekoo/clmm-lsis/internal/version"
)

func runVersion(
	args []string,
	stdout io.Writer,
	stderr io.Writer,
) int {
	flags := flag.NewFlagSet("version", flag.ContinueOnError)
	flags.SetOutput(stderr)

	if err := flags.Parse(args); err != nil {
		return exitUsage
	}

	if flags.NArg() != 0 {
		fmt.Fprintln(stderr, "version does not accept positional arguments")
		return exitUsage
	}

	fmt.Fprintln(stdout, version.String())

	return exitSuccess
}
