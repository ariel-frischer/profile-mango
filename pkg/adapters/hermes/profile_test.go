package hermes

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestValidProfileName(t *testing.T) {
	tests := map[string]bool{
		"coding":                true,
		"code_review-2":         true,
		"9lives":                true,
		strings.Repeat("a", 64): true,
		strings.Repeat("a", 65): false,
		"":                      false,
		"_lead":                 false,
		"-lead":                 false,
		"Coding":                false, // hermes_cli/profiles.py _PROFILE_ID_RE is lowercase-only
		"a.b":                   false,
		"a/b":                   false,
		"has space":             false,
		"hermes":                false, // reserved
		"test":                  false,
		"tmp":                   false,
		"root":                  false,
		"sudo":                  false,
		"default":               true, // validate_profile_name special-cases "default" to pass
	}
	for name, want := range tests {
		if got := ValidProfileName(name); got != want {
			t.Fatalf("ValidProfileName(%q) = %v, want %v", name, got, want)
		}
	}
}

func TestIsDefaultProfile(t *testing.T) {
	tests := map[string]bool{"default": true, "Default": true, "DEFAULT": true, "coding": false, "": false}
	for name, want := range tests {
		if got := IsDefaultProfile(name); got != want {
			t.Fatalf("IsDefaultProfile(%q) = %v, want %v", name, got, want)
		}
	}
}

func TestProfileConfigPath(t *testing.T) {
	home := filepath.Join(string(filepath.Separator), "home", "user", ".hermes")
	canonical := filepath.Join(home, "config.yaml")
	tests := map[string]struct {
		config string
		name   string
		want   string
		err    string
	}{
		"named profile":     {config: canonical, name: "coding", want: filepath.Join(home, "profiles", "coding", "config.yaml")},
		"unclean path":      {config: home + "/./config.yaml", name: "dev", want: filepath.Join(home, "profiles", "dev", "config.yaml")},
		"relocated config":  {config: filepath.Join(home, "cfg", "config.yaml"), name: "coding", want: filepath.Join(home, "cfg", "profiles", "coding", "config.yaml")},
		"invalid name":      {config: canonical, name: "../x", err: "profile names"},
		"reserved name":     {config: canonical, name: "hermes", err: "profile names"},
		"default name":      {config: canonical, name: "default", want: canonical},
		"default any case":  {config: canonical, name: "Default", want: canonical},
		"default relocated": {config: filepath.Join(home, "cfg", "other.yaml"), name: "default", want: filepath.Join(home, "cfg", "other.yaml")},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			got, err := ProfileConfigPath(tc.config, tc.name)
			if tc.err != "" {
				if err == nil || !strings.Contains(err.Error(), tc.err) {
					t.Fatalf("error = %v, want %q", err, tc.err)
				}
				return
			}
			if err != nil || got != tc.want {
				t.Fatalf("path = %q, %v; want %q", got, err, tc.want)
			}
		})
	}
}
