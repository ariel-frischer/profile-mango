package statehome

import (
	"errors"
	"path/filepath"
	"strings"
	"testing"
)

func TestResolvePrecedence(t *testing.T) {
	root := t.TempDir()
	user := func() (string, error) { return filepath.Join(root, "user"), nil }
	tests := map[string]struct {
		override, xdgState string
		want               string
	}{
		"override wins":        {override: filepath.Join(root, "x", "..", "state"), xdgState: filepath.Join(root, "xdg"), want: filepath.Join(root, "state")},
		"xdg state home":       {xdgState: filepath.Join(root, "xdg"), want: filepath.Join(root, "xdg", "profile-mango")},
		"relative xdg skipped": {xdgState: "relative", want: filepath.Join(root, "user", ".local", "state", "profile-mango")},
		"user home default":    {want: filepath.Join(root, "user", ".local", "state", "profile-mango")},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			got, err := resolve(test.override, test.xdgState, user)
			if err != nil {
				t.Fatalf("resolve: %v", err)
			}
			if got != test.want {
				t.Fatalf("path = %q, want %q", got, test.want)
			}
		})
	}
}

func TestResolveRejectsUnavailableHome(t *testing.T) {
	tests := map[string]func() (string, error){
		"lookup failure": func() (string, error) { return "", errors.New("unavailable") },
		"empty home":     func() (string, error) { return "", nil },
	}
	for name, userHome := range tests {
		t.Run(name, func(t *testing.T) {
			_, err := resolve("", "", userHome)
			if err == nil || !strings.Contains(err.Error(), "resolving user home") {
				t.Fatalf("error = %v", err)
			}
		})
	}
}
