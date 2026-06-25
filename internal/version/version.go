// Package version holds build-time metadata injected via -ldflags.
package version

import "fmt"

// Build-time variables, overridden by ldflags:
//
//	-X github.com/pmtop/pmtop/internal/version.Version=...
//	-X github.com/pmtop/pmtop/internal/version.Commit=...
//	-X github.com/pmtop/pmtop/internal/version.Date=...
var (
	Version = "dev"
	Commit  = "none"
	Date    = "unknown"
)

// String returns a human-readable version string.
func String() string {
	return fmt.Sprintf("pmtop %s (commit: %s, built: %s)", Version, Commit, Date)
}

// Short returns just the version number.
func Short() string {
	return Version
}
