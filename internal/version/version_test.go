package version

import (
	"runtime/debug"
	"testing"
)

func TestIsDevBuild(t *testing.T) {
	tests := map[string]struct {
		version string
		want    bool
	}{
		"dev build":     {version: "dev", want: true},
		"release build": {version: "1.0.0", want: false},
		"pre-release":   {version: "1.0.0-rc1", want: false},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			orig := Version
			defer func() { Version = orig }()

			Version = tt.version
			if got := IsDevBuild(); got != tt.want {
				t.Errorf("IsDevBuild() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestFromBuildInfo(t *testing.T) {
	defaults := Info{Version: "dev", Commit: "unknown", BuildDate: "unknown"}
	stamped := func(version string, settings ...debug.BuildSetting) *debug.BuildInfo {
		return &debug.BuildInfo{Main: debug.Module{Version: version}, Settings: settings}
	}
	rev := debug.BuildSetting{Key: "vcs.revision", Value: "0123456789abcdef0123456789abcdef01234567"}
	when := debug.BuildSetting{Key: "vcs.time", Value: "2026-09-26T12:00:00Z"}
	tests := map[string]struct {
		current Info
		info    *debug.BuildInfo
		want    Info
	}{
		"nil build info keeps defaults": {current: defaults, info: nil, want: defaults},
		"module install uses module version": {
			current: defaults, info: stamped("v0.1.0"),
			want: Info{Version: "v0.1.0", Commit: "unknown", BuildDate: "unknown"},
		},
		"pseudo-version with vcs stamps": {
			current: defaults, info: stamped("v0.0.0-20260926120000-0123456789ab", rev, when),
			want: Info{Version: "v0.0.0-20260926120000-0123456789ab", Commit: "0123456", BuildDate: "2026-09-26T12:00:00Z"},
		},
		"devel checkout keeps dev but fills vcs": {
			current: defaults, info: stamped("(devel)", rev, when),
			want: Info{Version: "dev", Commit: "0123456", BuildDate: "2026-09-26T12:00:00Z"},
		},
		"short revision kept whole": {
			current: defaults, info: stamped("", debug.BuildSetting{Key: "vcs.revision", Value: "abc"}),
			want: Info{Version: "dev", Commit: "abc", BuildDate: "unknown"},
		},
		"ldflags take precedence": {
			current: Info{Version: "v0.2.0", Commit: "fedcba9", BuildDate: "2026-01-01T00:00:00Z"},
			info:    stamped("v0.1.0", rev, when),
			want:    Info{Version: "v0.2.0", Commit: "fedcba9", BuildDate: "2026-01-01T00:00:00Z"},
		},
		"partial ldflags commit kept": {
			current: Info{Version: "dev", Commit: "fedcba9", BuildDate: "unknown"},
			info:    stamped("(devel)", rev, when),
			want:    Info{Version: "dev", Commit: "fedcba9", BuildDate: "2026-09-26T12:00:00Z"},
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			if got := FromBuildInfo(tt.current, tt.info); got != tt.want {
				t.Errorf("FromBuildInfo() = %+v, want %+v", got, tt.want)
			}
		})
	}
}
