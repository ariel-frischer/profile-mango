package main

import (
	"fmt"
	"os"
	"testing"

	"github.com/ariel-frischer/profile-mango/internal/statehome"
	"github.com/ariel-frischer/profile-mango/internal/testenv"
	"github.com/ariel-frischer/profile-mango/pkg/install"
)

// TestMain isolates every test from the real user home and refuses to run if any
// default target config path could still resolve outside the sandbox.
func TestMain(m *testing.M) {
	root, cleanup, err := testenv.Sandbox()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	// Never run real agent commands from install tests; tests that cover
	// detection install their own detector over fake binaries.
	installVersionDetector = nil
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
	if state, err := statehome.Resolve(); err != nil || !testenv.Within(root, state) {
		return fmt.Errorf("test home guard: state directory %q escapes sandbox %s: %v", state, root, err)
	}
	registry := install.DefaultRegistry()
	for _, target := range registry.Targets() {
		path, err := registry.DefaultConfigPath(target, install.OSPathEnv())
		if err != nil {
			return fmt.Errorf("test home guard: resolve %s: %w", target.String(), err)
		}
		if !testenv.Within(root, path) {
			return fmt.Errorf("test home guard: %s default %s escapes sandbox %s", target.String(), path, root)
		}
	}
	return nil
}
