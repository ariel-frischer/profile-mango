// Package testenv isolates test processes from the real user home and agent config locations.
package testenv

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// RelocationVars are cleared so documented agent and Mango relocations cannot point at real state.
var RelocationVars = []string{"CODEX_HOME", "HERMES_HOME", "PI_CODING_AGENT_DIR", "OPENCLAW_CONFIG_PATH", "PROFILE_MANGO_HOME", "PROFILE_MANGO_STATE_DIR"}

// Sandbox points HOME, XDG_CONFIG_HOME and XDG_STATE_HOME at a fresh temporary directory and
// clears relocation variables. It returns the sandbox root and a cleanup function.
func Sandbox() (string, func(), error) {
	root, err := os.MkdirTemp("", "profile-mango-test-home-")
	if err != nil {
		return "", nil, fmt.Errorf("create sandbox home: %w", err)
	}
	cleanup := func() { _ = os.RemoveAll(root) }
	settings := map[string]string{"HOME": filepath.Join(root, "home"), "XDG_CONFIG_HOME": filepath.Join(root, "xdg"), "XDG_STATE_HOME": filepath.Join(root, "xdg-state")}
	for _, name := range RelocationVars {
		settings[name] = ""
	}
	for name, value := range settings {
		if err := setOrUnset(name, value); err != nil {
			cleanup()
			return "", nil, err
		}
	}
	return root, cleanup, nil
}

func setOrUnset(name, value string) error {
	var err error
	if value == "" {
		err = os.Unsetenv(name)
	} else {
		err = os.Setenv(name, value)
	}
	if err != nil {
		return fmt.Errorf("isolate %s: %w", name, err)
	}
	return nil
}

// Within reports whether path is the sandbox root or beneath it.
func Within(root, path string) bool {
	rel, err := filepath.Rel(root, path)
	return err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}
