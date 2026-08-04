package version

import (
	"fmt"
	"runtime"
	gotime "time"
)

// This will be set via ldflags at build time.
var (
	variant string = "<unset>"
	version string = "<unset>"
	hash    string = "<unset>"
	time    string = "<unset>"
	branch  string = "<unset>"
)

var (
	Info Build = Build{
		variant:   variant,
		version:   version,
		hash:      hash,
		time:      time,
		branch:    branch,
		goVersion: runtime.Version(),
	}
)

// Build contains information about the build environment.
type Build struct {
	// The variant of the current build.
	variant string

	// The version of the current build.
	version string

	// The git hash of the current build.
	hash string

	// The time of the current build.
	time string

	// The branch of the current build.
	branch string

	// The version of go.
	goVersion string
}

func (b Build) Variant() string {
	return b.variant
}

func (b Build) Version() string {
	return b.version
}

func (b Build) Hash() string {
	return b.hash
}

func (b Build) Time() string {
	return b.time
}

func (b Build) Branch() string {
	return b.branch
}

func (b Build) GoVersion() string {
	return b.goVersion
}

func (b Build) PrettyPrint(cfg Config) {
	if !cfg.Banner.Enabled {
		return
	}

	fmt.Print(b.banner(gotime.Now().Year())) //nolint:forbidigo
}

func (b Build) banner(year int) string {
	return fmt.Sprintf(
		"\nSigInsight\nVersion: %s (%s)\nCopyright %d SigInsight. All rights reserved.\n\n",
		b.version,
		b.variant,
		year,
	)
}
