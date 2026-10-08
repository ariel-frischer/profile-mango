// Package statehome resolves the private Mango state directory that keeps install and undo history.
package statehome

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const (
	// EnvStateDir overrides the state directory.
	EnvStateDir = "PROFILE_MANGO_STATE_DIR"
	envXDGState = "XDG_STATE_HOME"
)

// Resolve returns the absolute state directory: $PROFILE_MANGO_STATE_DIR, else
// $XDG_STATE_HOME/profile-mango, else <user-home>/.local/state/profile-mango.
// It never creates the directory.
func Resolve() (string, error) {
	return resolve(os.Getenv(EnvStateDir), os.Getenv(envXDGState), os.UserHomeDir)
}

func resolve(override, xdgState string, userHome func() (string, error)) (string, error) {
	if path := strings.TrimSpace(override); path != "" {
		return absolute(path)
	}
	// The XDG base directory spec ignores relative paths.
	if path := strings.TrimSpace(xdgState); filepath.IsAbs(path) {
		return absolute(filepath.Join(path, "profile-mango"))
	}
	home, err := userHome()
	if err != nil {
		return "", fmt.Errorf("resolving user home for profile-mango state: %w", err)
	}
	if strings.TrimSpace(home) == "" {
		return "", fmt.Errorf("resolving user home for profile-mango state: empty path")
	}
	return absolute(filepath.Join(home, ".local", "state", "profile-mango"))
}

func absolute(path string) (string, error) {
	resolved, err := filepath.Abs(filepath.Clean(path))
	if err != nil {
		return "", fmt.Errorf("resolving profile-mango state directory %q: %w", path, err)
	}
	return resolved, nil
}
