// Package version holds build metadata, set via -ldflags -X at release time.
package version

var (
	Version = "dev"
	Commit  = "none"
	Date    = "unknown"
)
