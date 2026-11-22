// Package ops holds sinkctl operations as typed input -> output funcs.
// CLI commands and MCP tools are thin adapters over this package.
package ops

import (
	"runtime"

	"github.com/kidk/sinkctl/internal/version"
)

type VersionInfo struct {
	Version   string `json:"version" yaml:"version"`
	Commit    string `json:"commit" yaml:"commit"`
	Date      string `json:"date" yaml:"date"`
	GoVersion string `json:"go_version" yaml:"go_version"`
	Platform  string `json:"platform" yaml:"platform"`
}

func Version() VersionInfo {
	return VersionInfo{
		Version:   version.Version,
		Commit:    version.Commit,
		Date:      version.Date,
		GoVersion: runtime.Version(),
		Platform:  runtime.GOOS + "/" + runtime.GOARCH,
	}
}

func (v VersionInfo) Table() ([]string, [][]string) {
	return []string{"VERSION", "COMMIT", "DATE", "GO", "PLATFORM"},
		[][]string{{v.Version, v.Commit, v.Date, v.GoVersion, v.Platform}}
}
