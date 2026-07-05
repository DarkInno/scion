package version

import (
	"runtime/debug"
	"strings"
)

var (
	// Version and Commit are overridden by release builds with -ldflags.
	Version = "0.1.0-dev"
	Commit  = "dev"
)

func Current() string {
	if Version != "" && !strings.HasSuffix(Version, "-dev") {
		return Version
	}
	if info, ok := debug.ReadBuildInfo(); ok {
		if info.Main.Version != "" && info.Main.Version != "(devel)" {
			return info.Main.Version
		}
	}
	return Version
}

func CurrentCommit() string {
	if Commit != "" && Commit != "dev" {
		return Commit
	}
	if info, ok := debug.ReadBuildInfo(); ok {
		for _, setting := range info.Settings {
			if setting.Key == "vcs.revision" && setting.Value != "" {
				return setting.Value
			}
		}
	}
	return Commit
}
