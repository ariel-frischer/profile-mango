package install

import (
	"fmt"
	"os"
	"testing"

	"gitlab.com/ariel-frischer/profile-mango/internal/testenv"
)

// TestMain isolates every test from the real user home and refuses to run if any
// default target config path could still resolve outside the sandbox.
func TestMain(m *testing.M) {
	root, cleanup, err := testenv.Sandbox()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	code := 1
	if err := guardDefaultPaths(root); err != nil {
		fmt.Fprintln(os.Stderr, err)
	} else {
		code = m.Run()
	}
	cleanup()
	os.Exit(code)
}

func guardDefaultPaths(root string) error {
	registry := DefaultRegistry()
	for _, target := range registry.Targets() {
		path, err := registry.DefaultConfigPath(target, OSPathEnv())
		if err != nil {
			return fmt.Errorf("test home guard: resolve %s: %w", target.String(), err)
		}
		if !testenv.Within(root, path) {
			return fmt.Errorf("test home guard: %s default %s escapes sandbox %s", target.String(), path, root)
		}
	}
	return nil
}
