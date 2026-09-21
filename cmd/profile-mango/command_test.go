package main

import (
	"bytes"
	"path/filepath"
	"strings"
	"testing"
)

func executeCommand(t *testing.T, args ...string) string {
	t.Helper()

	var out bytes.Buffer
	rootCmd.SetOut(&out)
	rootCmd.SetErr(&out)
	rootCmd.SetArgs(args)

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("profile-mango %s failed: %v\n%s", strings.Join(args, " "), err, out.String())
	}
	return out.String()
}

func TestVersionCommandSmoke(t *testing.T) {
	executeCommand(t, "version")
}

func TestVersionAliasSmoke(t *testing.T) {
	executeCommand(t, "v")
}

func TestHelpCommandSmoke(t *testing.T) {
	executeCommand(t, "help")
}

func TestRootCommandOmitsRemovedConfigSurface(t *testing.T) {
	for _, command := range rootCmd.Commands() {
		if command.Name() == "config" {
			t.Fatal("root command still registers removed config command")
		}
	}
	if flag := rootCmd.PersistentFlags().Lookup("config"); flag != nil {
		t.Fatal("root command still registers removed --config flag")
	}
	if flag := rootCmd.PersistentFlags().Lookup("no-color"); flag == nil {
		t.Fatal("root command omitted preserved --no-color flag")
	}
}

func TestAgentsCheckHelpSmoke(t *testing.T) {
	executeCommand(t, "agents", "check", "--help")
}

func TestCompletionCommandSmoke(t *testing.T) {
	for _, shell := range []string{"bash", "zsh", "fish", "powershell"} {
		t.Run(shell, func(t *testing.T) {
			executeCommand(t, "completion", shell)
		})
	}
}

func TestValidateCommandSmoke(t *testing.T) {
	profile := filepath.Join("..", "..", "pkg", "profilemango", "testdata", "fixtures", "route-only", "profile.yaml")
	bindings := filepath.Join("..", "..", "pkg", "profilemango", "testdata", "fixtures", "bindings.yaml")
	out := executeCommand(t, "validate", profile, "--bindings", bindings, "--json")
	if !strings.Contains(out, `"valid": true`) {
		t.Fatalf("validate output = %q", out)
	}
}
