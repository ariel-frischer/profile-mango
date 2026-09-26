package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ariel-frischer/profile-mango/internal/profilehome"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

func executeCommand(t *testing.T, args ...string) string {
	t.Helper()

	var out bytes.Buffer
	rootCmd.SetOut(&out)
	rootCmd.SetErr(&out)
	resetHomeFlag(t)
	resetColorFlag(t)
	resetNonInteractiveFlag(t)
	resetSubcommandFlags(rootCmd)
	rootCmd.SetArgs(args)

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("mango %s failed: %v\n%s", strings.Join(args, " "), err, out.String())
	}
	return out.String()
}

// resetColorFlag clears the package-global noColor flag and, since tests share
// rootCmd and its globals across the package, also restores it with
// t.Cleanup so a later test never observes a value this test left behind.
func resetColorFlag(t *testing.T) {
	t.Helper()
	reset := func() {
		noColor = false
		if flag := rootCmd.PersistentFlags().Lookup("no-color"); flag != nil {
			_ = flag.Value.Set("false")
			flag.Changed = false
		}
	}
	reset()
	t.Cleanup(reset)
}

// resetHomeFlag clears the package-global homePathOverride flag and restores
// it with t.Cleanup for the same cross-test isolation reason as resetColorFlag.
func resetHomeFlag(t *testing.T) {
	t.Helper()
	reset := func() {
		homePathOverride = ""
		if flag := rootCmd.PersistentFlags().Lookup("home"); flag != nil {
			_ = flag.Value.Set("")
			flag.Changed = false
		}
	}
	reset()
	t.Cleanup(reset)
}

// resetNonInteractiveFlag clears the package-global nonInteractive flag and
// restores it with t.Cleanup for the same cross-test isolation reason as
// resetColorFlag; without this, a test that sets --non-interactive can leak
// the global into an unrelated later test regardless of run order.
func resetNonInteractiveFlag(t *testing.T) {
	t.Helper()
	reset := func() {
		nonInteractive = false
		if flag := rootCmd.PersistentFlags().Lookup("non-interactive"); flag != nil {
			_ = flag.Value.Set("false")
			flag.Changed = false
		}
	}
	reset()
	t.Cleanup(reset)
}

// resetSubcommandFlags recursively resets every flag on command and its
// descendants to its registered default, including --help. Every test in
// this package shares the singleton rootCmd tree, and pflag never resets a
// flag that a later cobra.Execute() call does not re-specify on its own
// argv; without this, one test's --target, --apply, --yes, --profile, or
// similar subcommand flag value can silently leak into a later test (the
// same class of bug fixed for the package-global nonInteractive var in
// ap-uuz.11).
func resetSubcommandFlags(command *cobra.Command) {
	reset := func(flag *pflag.Flag) {
		if value, ok := flag.Value.(pflag.SliceValue); ok {
			_ = value.Replace(nil)
		} else {
			_ = flag.Value.Set(flag.DefValue)
		}
		flag.Changed = false
	}
	command.Flags().VisitAll(reset)
	command.PersistentFlags().VisitAll(reset)
	for _, child := range command.Commands() {
		resetSubcommandFlags(child)
	}
}

func TestVersionCommandSmoke(t *testing.T) {
	executeCommand(t, "version")
}

func TestVersionAliasSmoke(t *testing.T) {
	executeCommand(t, "ver")
}

func TestRootCommandUsesMangoIdentity(t *testing.T) {
	if rootCmd.Use != "mango" {
		t.Fatalf("root command use = %q, want mango", rootCmd.Use)
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
	if strings.Contains(output, "jcode-fork") {
		t.Fatalf("render --help still mentions jcode-fork:\n%s", output)
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
	if !strings.Contains(output, "mango ") {
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
	if !strings.Contains(err.Error(), "cp ") || !strings.Contains(err.Error(), "local.example.yaml") || !strings.Contains(err.Error(), "mango init") {
		t.Fatalf("missing bindings error lacks an exact fix: %v", err)
	}
}
