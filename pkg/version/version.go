// Package version provides version information for the application.
package version

import "runtime"

// These variables are set during build time via ldflags
var (
	Version   = "dev"
	BuildTime = "unknown"
	GitCommit = "unknown"
	GoVersion = runtime.Version()
)

// Info returns a map with all version information.
func Info() map[string]string {
	return map[string]string{
		"version":   Version,
		"buildTime": BuildTime,
		"gitCommit": GitCommit,
		"goVersion": GoVersion,
	}
}
