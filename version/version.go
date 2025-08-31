package version

import (
	"os"
	"regexp"
	"runtime/debug"
)

var semver = regexp.MustCompile(`^v\d+\.\d+\.\d+(-[0-9A-Za-z.-]+)?$`)

// IsRelease returns true if the binary looks like it was built from a semver tag
// (e.g. "v1.2.3"), false if it's "(devel)" or a pseudo-version.
func IsRelease() bool {
	if os.Getenv("WHATEVER_ENV") == "dev" {
		return false
	}
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return false
	}
	return semver.MatchString(info.Main.Version)
}

// Version returns the raw version string reported by Go.
//   - "v1.2.3" when built from a semver tag (release)
//   - "(devel)" when run locally with `go run .`
//   - "v0.0.0-20250831-abcdef123456" for pseudo-versions (dev)
func Version() string {
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return "(unknown)"
	}
	return info.Main.Version
}
