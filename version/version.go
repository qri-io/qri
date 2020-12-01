package version

import "fmt"

var (
	// Version is set by govvv at build time
	Version = "n/a"
	// GitCommit is set by govvv at build time
	GitCommit = "n/a"
	// GitBranch  is set by govvv at build time
	GitBranch = "n/a"
	// GitState  is set by govvv at build time
	GitState = "n/a"
	// GitSummary is set by govvv at build time
	GitSummary = "n/a"
	// BuildDate  is set by govvv at build time
	BuildDate = "n/a"
	// GolangVersion is set by govvv at build time
	GolangVersion = "n/a"
)

// Map returns a summary of build info as a string map
func Map() map[string]string {
	if Version == "n/a" {
		return map[string]string{
			"error": `This qri binary was not built with version information. Please build using 'make install' instead of 'go install'`,
		}
	}

	return map[string]string{
		"version":       Version,
		"gitCommit":     GitCommit,
		"gitBranch":     GitBranch,
		"gitState":      GitState,
		"gitSummary":    GitSummary,
		"buildDate":     BuildDate,
		"golangVersion": GolangVersion,
	}
}

// Summary prints a summary of all build info.
func Summary() string {
	if Version == "n/a" {
		return `Warning! This qri binary was not built with version information. Please build using 'make install' instead of 'go install'`
	}

	return fmt.Sprintf(
		"version:\t%s\nbuild date:\t%s\ngit summary:\t%s\ngit branch:\t%s\ngit commit:\t%s\ngit state:\t%s\ngolang version:\t%s",
		Version,
		BuildDate,
		GitSummary,
		GitBranch,
		GitCommit,
		GitState,
		GolangVersion,
	)
}
