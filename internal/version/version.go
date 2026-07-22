package version

import "fmt"

var (
	Version   = "development"
	Commit    = "unknown"
	BuildDate = "unknown"
)

func String() string {
	return fmt.Sprintf(
		"%s (commit=%s, built=%s)",
		Version,
		Commit,
		BuildDate,
	)
}
