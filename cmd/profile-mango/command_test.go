package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"gitlab.com/ariel-frischer/profile-mango/internal/profilehome"
)

func executeCommand(t *testing.T, args ...string) string {
	t.Helper()

	var out bytes.Buffer
	rootCmd.SetOut(&out)
	rootCmd.SetErr(&out)
	resetHomeFlag()
	resetColorFlag()
	resetNonInteractiveFlag()
	resetHelpFlags(rootCmd)
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

func resetNonInteractiveFlag() {
	nonInteractive = false
	if flag := rootCmd.PersistentFlags().Lookup("non-interactive"); flag != nil {
		_ = flag.Value.Set("false")
		flag.Changed = false
	}
}

func resetHelpFlags(command *cobra.Command) {
	if flag := command.Flags().Lookup("help"); flag != nil {
		_ = flag.Value.Set("false")
		flag.Changed = false
	}
	if flag := command.PersistentFlags().Lookup("help"); flag != nil {
		_ = flag.Value.Set("false")
		flag.Changed = false
	}
	for _, child := range command.Commands() {
		resetHelpFlags(child)
	}
}

func TestVersionCommandSmoke(t *testing.T) {
	executeCommand(t, "version")
}

func TestVersionAliasSmoke(t *testing.T) {
	executeCommand(t, "ver")
}

func TestRootCommandUsesMangoIdentity(t *testing.T) {
	if rootCmd.Use != "profile-mango" {
		t.Fatalf("root command use = %q, want profile-mango", rootCmd.Use)
	}
}

func TestCommandAliasesResolveAndRenderHelp(t *testing.T) {
	tests := map[string]struct {
		path  []string
		alias string
	}{
		"validate":     {path: []string{"validate"}, alias: "v"},
		"version":      {path: []string{"version"}, alias: "ver"},
		"home":         {path: []string{"home"}, alias: "h"},
		"init":         {path: []string{"init"}, alias: "new"},
		"agents":       {path: []string{"agents"}, alias: "a"},
		"agents check": {path: []string{"agents", "check"}, alias: "c"},
		"render":       {path: []string{"render"}, alias: "r"},
		"install":      {path: []string{"install"}, alias: "i"},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			command := commandAtPath(t, test.path...)
			if !containsString(command.Aliases, test.alias) {
				t.Fatalf("%s aliases = %v, want %q", strings.Join(test.path, " "), command.Aliases, test.alias)
			}
			findPath := append(append([]string{}, test.path[:len(test.path)-1]...), test.alias)
			found, _, err := rootCmd.Find(findPath)
			if err != nil {
				t.Fatalf("find alias %q: %v", strings.Join(findPath, " "), err)
			}
			if found != command {
				t.Fatalf("alias %q resolved to %s, want %s", test.alias, found.CommandPath(), command.CommandPath())
			}

			args := append(findPath, "--help")
			output := executeCommand(t, args...)
			if !strings.Contains(output, "Aliases:") || !strings.Contains(output, test.alias) {
				t.Fatalf("help for alias %q omitted alias listing:\n%s", test.alias, output)
			}
		})
	}
}

func commandAtPath(t *testing.T, path ...string) *cobra.Command {
	t.Helper()
	command := rootCmd
	for _, name := range path {
		var found *cobra.Command
		for _, candidate := range command.Commands() {
			if candidate.Name() == name {
				found = candidate
				break
			}
		}
		if found == nil {
			t.Fatalf("command %q not found below %s", name, command.CommandPath())
		}
		command = found
	}
	return command
}

func containsString(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
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

// bannedHelpJargon lists internal-sounding phrasing that must never reach the
// public --help surface (Bead ap-uuz.6).
var bannedHelpJargon = []string{
	"capability-aware target artifacts",
	"hash-bound consent",
	"inert, version-qualified candidate",
	"adapter-approved narrow ownership override",
	"bounded",
}

func TestHelpSurfaceHasNoJargon(t *testing.T) {
	var walk func(*cobra.Command)
	walk = func(command *cobra.Command) {
		text := strings.Join([]string{
			command.Short, command.Long,
			command.LocalFlags().FlagUsages(),
			command.PersistentFlags().FlagUsages(),
		}, "\n")
		for _, term := range bannedHelpJargon {
			if strings.Contains(text, term) {
				t.Fatalf("%s help contains jargon %q", command.CommandPath(), term)
			}
		}
		for _, sub := range command.Commands() {
			walk(sub)
		}
	}
	walk(rootCmd)
}

func TestRootHelpHidesAgentsCommand(t *testing.T) {
	output := executeCommand(t, "--help")
	if strings.Contains(output, "\n  agents") {
		t.Fatalf("root help still lists hidden agents command:\n%s", output)
	}
}

func TestAgentsCommandIsHiddenButStillWorks(t *testing.T) {
	for _, command := range rootCmd.Commands() {
		if command.Name() == "agents" && !command.Hidden {
			t.Fatal("agents command is no longer hidden")
		}
	}
	executeCommand(t, "agents", "check", "--help")
}

func TestRenderHelpHidesArielJcodeTarget(t *testing.T) {
	output := executeCommand(t, "render", "--help")
	if strings.Contains(output, "ariel-jcode") {
		t.Fatalf("render --help still mentions ariel-jcode:\n%s", output)
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

func TestNonInteractiveFlagIsGlobal(t *testing.T) {
	if flag := rootCmd.PersistentFlags().Lookup("non-interactive"); flag == nil {
		t.Fatal("root command omitted --non-interactive flag")
	}
	output, err := executeCommandResult(t, "--non-interactive", "version", "--plain")
	if err != nil {
		t.Fatalf("non-interactive version failed: %v", err)
	}
	if !strings.Contains(output, "profile-mango ") {
		t.Fatalf("version output = %q", output)
	}
}

func TestNonInteractiveInstallStillRequiresExplicitConsent(t *testing.T) {
	_, err := executeCommandResult(t, "--non-interactive", "install", "route-only", "--target", "codex@0.154.0", "--apply")
	if err == nil || !strings.Contains(err.Error(), "--non-interactive apply requires --yes --expect-plan") {
		t.Fatalf("error = %v, want explicit consent error", err)
	}
}

func TestInstallConsentFlagsRemainExplicit(t *testing.T) {
	cmd := newInstallCmd()
	for _, name := range []string{"apply", "yes", "expect-plan"} {
		if flag := cmd.Flags().Lookup(name); flag == nil {
			t.Fatalf("install command omitted --%s", name)
		}
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

func TestValidateCommandMissingBindingsNamesFix(t *testing.T) {
	profile := filepath.Join("..", "..", "pkg", "profilemango", "testdata", "fixtures", "route-only", "profile.yaml")
	missing := filepath.Join(t.TempDir(), "bindings", "local.yaml")
	_, err := executeCommandResult(t, "validate", profile, "--bindings", missing)
	if err == nil {
		t.Fatal("validate succeeded despite missing bindings")
	}
	if !strings.Contains(err.Error(), "cp ") || !strings.Contains(err.Error(), "local.example.yaml") || !strings.Contains(err.Error(), "profile-mango init") {
		t.Fatalf("missing bindings error lacks an exact fix: %v", err)
	}
}
