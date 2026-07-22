package cli

import (
	"context"
	"fmt"
	"io"
)

const (
	exitSuccess = 0
	exitFailure = 1
	exitUsage   = 2
)

// Run executes the command-line application and returns a process exit code.
func Run(
	ctx context.Context,
	args []string,
	stdout io.Writer,
	stderr io.Writer,
) int {
	if len(args) == 0 {
		writeUsage(stderr)
		return exitUsage
	}

	switch args[0] {
	case "help", "-h", "--help":
		writeUsage(stdout)
		return exitSuccess

	case "version":
		return runVersion(args[1:], stdout, stderr)

	case "config-check":
		return runConfigCheck(ctx, args[1:], stdout, stderr)

	default:
		fmt.Fprintf(stderr, "unknown command %q\n\n", args[0])
		writeUsage(stderr)

		return exitUsage
	}
}

func writeUsage(writer io.Writer) {
	fmt.Fprintln(writer, "CLMM-LSIS research CLI")
	fmt.Fprintln(writer)
	fmt.Fprintln(writer, "Usage:")
	fmt.Fprintln(writer, "  clmm-lsis <command> [options]")
	fmt.Fprintln(writer)
	fmt.Fprintln(writer, "Commands:")
	fmt.Fprintln(writer, "  version       Print build version information")
	fmt.Fprintln(writer, "  config-check  Validate a study configuration")
	fmt.Fprintln(writer, "  help          Print this help message")
}
