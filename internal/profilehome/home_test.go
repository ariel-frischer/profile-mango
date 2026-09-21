package profilehome

import (
	"errors"
	"path/filepath"
	"strings"
	"testing"
)

func TestResolvePrecedence(t *testing.T) {
	root := t.TempDir()
	tests := map[string]struct {
		override    string
		environment string
		userHome    func() (string, error)
		want        string
	}{
		"flag override": {
			override:    filepath.Join(root, "flag", "..", "selected"),
			environment: filepath.Join(root, "environment"),
			userHome:    func() (string, error) { return filepath.Join(root, "user"), nil },
			want:        filepath.Join(root, "selected"),
		},
		"environment override": {
			environment: filepath.Join(root, "environment"),
			userHome:    func() (string, error) { return filepath.Join(root, "user"), nil },
			want:        filepath.Join(root, "environment"),
		},
		"user home default": {
			userHome: func() (string, error) { return filepath.Join(root, "user"), nil },
			want:     filepath.Join(root, "user", ".profile-mango"),
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			got, err := resolve(test.override, test.environment, test.userHome)
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
