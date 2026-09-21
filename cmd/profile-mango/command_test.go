package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gitlab.com/ariel-frischer/profile-mango/internal/profilehome"
)

func executeCommand(t *testing.T, args ...string) string {
	t.Helper()

	var out bytes.Buffer
	rootCmd.SetOut(&out)
	rootCmd.SetErr(&out)
	resetHomeFlag()
	resetColorFlag()
	rootCmd.SetArgs(args)

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("profile-mango %s failed: %v\n%s", strings.Join(args, " "), err, out.String())
	}
	return out.String()
}

func resetColorFlag() {
	noColor = false
	if flag := rootCmd.PersistentFlags().Lookup("no-color"); flag != nil {
		_ = flag.Value.Set("false")
		flag.Changed = false
	}
}

func resetHomeFlag() {
	homePathOverride = ""
	if flag := rootCmd.PersistentFlags().Lookup("home"); flag != nil {
		_ = flag.Value.Set("")
		flag.Changed = false
	}
}

func TestVersionCommandSmoke(t *testing.T) {
	executeCommand(t, "version")
}

func TestVersionAliasSmoke(t *testing.T) {
	executeCommand(t, "v")
}

func TestHelpCommandSmoke(t *testing.T) {
	output := executeCommand(t, "help")
	if strings.Contains(output, "\x1b[") {
		t.Fatalf("buffered help contains ANSI escapes")
	}
}

func TestNoColorFlagOutputIsPlain(t *testing.T) {
	output := executeCommand(t, "--no-color", "version")
	if strings.Contains(output, "\x1b[") {
		t.Fatalf("--no-color output contains ANSI escapes")
	}
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
	if flag := rootCmd.PersistentFlags().Lookup("home"); flag == nil {
		t.Fatal("root command omitted --home flag")
	}
}

func TestHomeCommandPrecedenceAndNoMutation(t *testing.T) {
	root := t.TempDir()
	environmentHome := filepath.Join(root, "environment")
	flagHome := filepath.Join(root, "flag")
	t.Setenv(profilehome.EnvHome, environmentHome)

	tests := map[string]struct {
		args []string
		want string
	}{
		"environment": {args: []string{"home"}, want: environmentHome},
		"flag":        {args: []string{"--home", flagHome, "home"}, want: flagHome},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			out := executeCommand(t, test.args...)
			if strings.TrimSpace(out) != test.want {
				t.Fatalf("home = %q, want %q", strings.TrimSpace(out), test.want)
			}
			if _, err := os.Stat(test.want); !os.IsNotExist(err) {
				t.Fatalf("home command mutated %s: %v", test.want, err)
			}
		})
	}
}

func TestAgentsCheckHelpSmoke(t *testing.T) {
	executeCommand(t, "agents", "check", "--help")
}

func TestCompletionCommandSmoke(t *testing.T) {
	for _, shell := range []string{"bash", "zsh", "fish", "powershell"} {
		t.Run(shell, func(t *testing.T) {
			output := executeCommand(t, "completion", shell)
			if strings.Contains(output, "\x1b[") {
				t.Fatalf("completion output contains ANSI escapes")
			}
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
	if strings.Contains(out, "\x1b[") {
		t.Fatalf("validate JSON contains ANSI escapes: %q", out)
	}
}
