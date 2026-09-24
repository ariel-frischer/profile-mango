package openclaw

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestValidProfileName(t *testing.T) {
	tests := map[string]bool{
		"coding":                true,
		"Code_Review-2":         true,
		"9lives":                true,
		strings.Repeat("a", 64): true,
		strings.Repeat("a", 65): false,
		"":                      false,
		"_lead":                 false,
		"-lead":                 false,
		"a.b":                   false,
		"a/b":                   false,
		"has space":             false,
		"default":               true,
	}
	for name, want := range tests {
		if got := ValidProfileName(name); got != want {
			t.Fatalf("ValidProfileName(%q) = %v, want %v", name, got, want)
		}
	}
}

func TestProfileConfigPath(t *testing.T) {
	home := filepath.Join(string(filepath.Separator), "home", "user")
	canonical := filepath.Join(home, ".openclaw", "openclaw.json")
	tests := map[string]struct {
		config string
		name   string
		want   string
		err    string
	}{
		"default home":      {config: canonical, name: "coding", want: filepath.Join(home, ".openclaw-coding", "openclaw.json")},
		"unclean path":      {config: home + "/.openclaw/./openclaw.json", name: "dev", want: filepath.Join(home, ".openclaw-dev", "openclaw.json")},
		"relocated file":    {config: filepath.Join(home, "cfg", "openclaw.json"), name: "coding", err: "can only be derived"},
		"renamed file":      {config: filepath.Join(home, ".openclaw", "custom.json"), name: "coding", err: "can only be derived"},
		"relative path":     {config: filepath.Join(".openclaw", "openclaw.json"), name: "coding", err: "can only be derived"},
		"invalid name":      {config: canonical, name: "../x", err: "profile names"},
		"default name":      {config: canonical, name: "default", want: canonical},
		"default relocated": {config: filepath.Join(home, "cfg", "oc.json"), name: "Default", want: filepath.Join(home, "cfg", "oc.json")},
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
