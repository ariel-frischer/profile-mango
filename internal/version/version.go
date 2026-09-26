// Package version holds profile-mango version information.
// Separate package to avoid import cycles.
package version

import "runtime/debug"

var (
	// Set via ldflags during build
	Version   = "dev"
	Commit    = "unknown"
	BuildDate = "unknown"
)

// Info is a snapshot of the reported build metadata.
type Info struct {
	Version   string
	Commit    string
	BuildDate string
}

func init() {
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return
	}
	resolved := FromBuildInfo(Info{Version: Version, Commit: Commit, BuildDate: BuildDate}, info)
	Version, Commit, BuildDate = resolved.Version, resolved.Commit, resolved.BuildDate
}

// FromBuildInfo fills metadata that ldflags left at defaults from Go build
// info, as embedded by `go install module@version` or `go build` in a VCS
// checkout. Ldflags values always take precedence.
func FromBuildInfo(current Info, info *debug.BuildInfo) Info {
	if info == nil || current.Version != "dev" {
		return current
	}
	if v := info.Main.Version; v != "" && v != "(devel)" {
		current.Version = v
	}
	settings := make(map[string]string, len(info.Settings))
	for _, s := range info.Settings {
		settings[s.Key] = s.Value
	}
	if rev := settings["vcs.revision"]; rev != "" && current.Commit == "unknown" {
		current.Commit = shortRevision(rev)
	}
	if t := settings["vcs.time"]; t != "" && current.BuildDate == "unknown" {
		current.BuildDate = t
	}
	return current
}

func shortRevision(rev string) string {
	if len(rev) > 7 {
		return rev[:7]
	}
	return rev
}

func IsDevBuild() bool {
	return Version == "dev"
}
