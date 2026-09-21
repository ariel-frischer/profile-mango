// Package profilehome resolves the user-owned profile-mango home directory.
package profilehome

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const EnvHome = "PROFILE_MANGO_HOME"

// Resolve returns the effective absolute home path.
func Resolve(override string) (string, error) {
	return resolve(override, os.Getenv(EnvHome), os.UserHomeDir)
}

func resolve(override, environment string, userHome func() (string, error)) (string, error) {
	if path := strings.TrimSpace(override); path != "" {
		return absolute(path)
	}
	if path := strings.TrimSpace(environment); path != "" {
		return absolute(path)
	}
	home, err := userHome()
	if err != nil {
		return "", fmt.Errorf("resolving user home for profile-mango: %w", err)
	}
	if strings.TrimSpace(home) == "" {
		return "", fmt.Errorf("resolving user home for profile-mango: empty path")
	}
	return absolute(filepath.Join(home, ".profile-mango"))
}

func absolute(path string) (string, error) {
	resolved, err := filepath.Abs(filepath.Clean(path))
	if err != nil {
		return "", fmt.Errorf("resolving profile-mango home %q: %w", path, err)
	}
	return resolved, nil
}
