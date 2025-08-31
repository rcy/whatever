package version

import (
	"os"
	"runtime/debug"
)

func IsRelease() bool {
	return os.Getenv("WHATEVER_ENV") == "dev"
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
