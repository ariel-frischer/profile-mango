package main

import (
	"bytes"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"gitlab.com/ariel-frischer/profile-mango/internal/profilehome"
	"gitlab.com/ariel-frischer/profile-mango/pkg/profilemango"
)

const expectedReadmeLogo = `🥭 █▀█ █▀█ █▀█ █▀▀ █ █   █▀▀   ─   █▀▄▀█ ▄▀█ █▄ █ █▀▀ █▀█
█▀▀ █▀▄ █▄█ █▀  █ █▄▄ ██▄       █ ▀ █ █▀█ █ ▀█ █▄█ █▄█`

const expectedStarterProfile = `# The profile name defaults to its folder name (default).
description: Default profile scaffolded by mango init
route: local
`

const expectedStarterBindings = `# Route identity only. Credentials remain target-owned and are never stored here.
# transport defaults to native and authentication to oauth when omitted.
routes:
  local:
    provider: openai
    model: gpt-6-sol
    effort: high
    # Each listed agent gets the route above with these fields replaced.
    targets:
      claude-code:
        provider: anthropic
        model: claude-sonnet-5
`

const expectedBindingsGitignore = `# Keep machine-local bindings out of version control.
local.yaml
`

func TestInitCommandScaffoldsDefaultAndExplicitDestinations(t *testing.T) {
	tests := map[string]struct {
		args        []string
		chdir       string
		destination string
	}{
		"explicit current directory": {
			args:        []string{"."},
			chdir:       t.TempDir(),
			destination: ".",
		},
		"explicit directory": {
			args:        []string{filepath.Join("nested", "project")},
			chdir:       t.TempDir(),
			destination: filepath.Join("nested", "project"),
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			t.Chdir(test.chdir)
			if _, err := executeCommandResult(t, append([]string{"init"}, test.args...)...); err != nil {
				t.Fatalf("init failed: %v", err)
			}
			assertStarterTree(t, test.destination)
		})
	}
}

func TestInitCommandDefaultsToEffectiveHome(t *testing.T) {
	root := t.TempDir()
	tests := map[string]struct {
		args            []string
		environmentHome string
		destination     string
	}{
		"environment": {
			args:            []string{"init"},
			environmentHome: filepath.Join(root, "environment"),
			destination:     filepath.Join(root, "environment"),
		},
		"flag": {
			args:            []string{"--home", filepath.Join(root, "flag"), "init"},
			environmentHome: filepath.Join(root, "environment-unused"),
			destination:     filepath.Join(root, "flag"),
		},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			t.Setenv(profilehome.EnvHome, test.environmentHome)
			if _, err := executeCommandResult(t, test.args...); err != nil {
				t.Fatalf("init failed: %v", err)
			}
			assertStarterTree(t, test.destination)
		})
	}
}

func TestInitBindingsAreImmediatelyUsableByValidate(t *testing.T) {
	destination := filepath.Join(t.TempDir(), "project")
	if _, err := executeCommandResult(t, "init", destination); err != nil {
		t.Fatalf("init failed: %v", err)
	}
	profile := filepath.Join(destination, "profiles", "default", "profile.yaml")
	bindings := filepath.Join(destination, "bindings", "local.yaml")
	output, err := executeCommandResult(t, "validate", profile, "--bindings", bindings)
	if err != nil {
		t.Fatalf("validate immediately after init failed with no manual copy: %v\n%s", err, output)
	}
}

func TestWriteInitLogoNonTerminalIsPlain(t *testing.T) {
	var output bytes.Buffer
	if err := writeInitLogo(&output); err != nil {
		t.Fatalf("writeInitLogo: %v", err)
	}
	if got := output.String(); got != expectedReadmeLogo+"\n" {
		t.Fatalf("logo = %q, want %q", got, expectedReadmeLogo+"\n")
	}
}

func TestInitCommandOutputIncludesLogo(t *testing.T) {
	destination := filepath.Join(t.TempDir(), "project")

	output, err := executeCommandResult(t, "init", destination)
	if err != nil {
		t.Fatalf("init failed: %v", err)
	}
	want := expectedReadmeLogo + "\nCreated profile scaffold in " + destination + "\n"
	if output != want {
		t.Fatalf("init output = %q, want %q", output, want)
	}
}

func TestInitCommandFailureOmitsLogo(t *testing.T) {
	destination := t.TempDir()
	if err := os.MkdirAll(filepath.Join(destination, "profiles", "default"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(destination, "profiles", "default", "profile.yaml"), []byte("existing"), 0o644); err != nil {
		t.Fatal(err)
	}

	output, err := executeCommandResult(t, "init", destination)
	if err == nil {
		t.Fatal("init succeeded despite conflict")
	}
	if strings.Contains(output, "█▀█") || strings.Contains(output, "Created profile scaffold") {
		t.Fatalf("failed init emitted success output: %q", output)
	}
}

func TestInitCommandContentIsDeterministicAndStrictlyValid(t *testing.T) {
	first := t.TempDir()
	second := t.TempDir()
	for _, destination := range []string{first, second} {
		if _, err := executeCommandResult(t, "init", destination); err != nil {
			t.Fatalf("init %s failed: %v", destination, err)
		}
	}

	firstFiles := readStarterFiles(t, first)
	secondFiles := readStarterFiles(t, second)
	if !reflect.DeepEqual(firstFiles, secondFiles) {
		t.Fatalf("generated files differ:\nfirst=%v\nsecond=%v", firstFiles, secondFiles)
	}

	profile, profileDiagnostics := profilemango.ParseProfileAt(firstFiles["profiles/default/profile.yaml"], "default")
	if len(profileDiagnostics) > 0 || profile.Name != "default" {
		t.Fatalf("generated profile diagnostics = %v", profileDiagnostics.Sorted())
	}
	bindings, bindingDiagnostics := profilemango.ParseBindings(firstFiles["bindings/local.example.yaml"])
	if bindingDiagnostics.HasErrors() {
		t.Fatalf("generated binding diagnostics = %v", bindingDiagnostics.Sorted())
	}
	if _, found := bindings.Routes[profile.Route]; !found {
		t.Fatalf("route %q is absent from generated bindings", profile.Route)
	}
	if strings.Contains(string(firstFiles["bindings/local.example.yaml"]), "token") || strings.Contains(string(firstFiles["bindings/local.example.yaml"]), "secret") {
		t.Fatal("generated bindings contain token or secret guidance")
	}
	if !reflect.DeepEqual(firstFiles["bindings/local.yaml"], firstFiles["bindings/local.example.yaml"]) {
		t.Fatalf("bindings/local.yaml = %q, want a copy of the example", firstFiles["bindings/local.yaml"])
	}
	localBindings, localDiagnostics := profilemango.ParseBindings(firstFiles["bindings/local.yaml"])
	if localDiagnostics.HasErrors() {
		t.Fatalf("generated local binding diagnostics = %v", localDiagnostics.Sorted())
	}
	if _, found := localBindings.Routes[profile.Route]; !found {
		t.Fatalf("route %q is absent from generated bindings/local.yaml", profile.Route)
	}
	if strings.Contains(string(firstFiles["bindings/local.yaml"]), "gpt-5.6") {
		t.Fatal("generated bindings reference the retired gpt-5.6 model")
	}
}

func TestInitCommandRejectsConflictsWithoutMutation(t *testing.T) {
	tests := map[string]struct {
		conflict string
		content  []byte
	}{
		"destination file":        {conflict: "project", content: []byte("destination")},
		"profiles parent file":    {conflict: "project/profiles", content: []byte("profiles")},
		"profile directory file":  {conflict: "project/profiles/default", content: []byte("default")},
		"profile file":            {conflict: "project/profiles/default/profile.yaml", content: []byte("profile")},
		"bindings parent file":    {conflict: "project/bindings", content: []byte("bindings")},
		"example binding file":    {conflict: "project/bindings/local.example.yaml", content: []byte("binding")},
		"local binding file":      {conflict: "project/bindings/local.yaml", content: []byte("binding")},
		"bindings gitignore file": {conflict: "project/bindings/.gitignore", content: []byte("ignore")},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			root := t.TempDir()
			conflictPath := filepath.Join(root, test.conflict)
			if err := os.MkdirAll(filepath.Dir(conflictPath), 0o755); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(conflictPath, test.content, 0o644); err != nil {
				t.Fatal(err)
			}
			before := snapshotTree(root)

			_, err := executeCommandResult(t, "init", filepath.Join(root, "project"))
			if err == nil {
				t.Fatal("init succeeded despite a destination conflict")
			}
			if after := snapshotTree(root); !reflect.DeepEqual(after, before) {
				t.Fatalf("destination changed after conflict:\nbefore=%v\nafter=%v", before, after)
			}
		})
	}
}

func TestInitCommandPreflightsLaterParentConflicts(t *testing.T) {
	root := t.TempDir()
	destination := filepath.Join(root, "project")
	if err := os.MkdirAll(destination, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(destination, "profiles"), []byte("not a directory"), 0o644); err != nil {
		t.Fatal(err)
	}

	before := snapshotTree(root)
	_, err := executeCommandResult(t, "init", destination)
	if err == nil {
		t.Fatal("init succeeded with a non-directory parent")
	}
	if after := snapshotTree(root); !reflect.DeepEqual(after, before) {
		t.Fatalf("preflight conflict caused partial writes:\nbefore=%v\nafter=%v", before, after)
	}
}

func TestInitWriteRollsBackCreatedPathsAfterWriteFailure(t *testing.T) {
	root := t.TempDir()
	if err := os.Mkdir(filepath.Join(root, "bindings"), 0o755); err != nil {
		t.Fatal(err)
	}
	files := []starterFile{
		{path: "profiles/default/profile.yaml", content: []byte("first")},
		{path: "bindings", content: []byte("cannot overwrite directory")},
	}

	err := writeInitFiles(root, files)
	if err == nil {
		t.Fatal("writeInitFiles succeeded despite a write failure")
	}
	if _, statErr := os.Stat(filepath.Join(root, "profiles")); !os.IsNotExist(statErr) {
		t.Fatalf("created profile paths remain after rollback, stat error = %v", statErr)
	}
	if info, statErr := os.Stat(filepath.Join(root, "bindings")); statErr != nil || !info.IsDir() {
		t.Fatalf("pre-existing binding directory changed: info=%v err=%v", info, statErr)
	}
}

func TestInitCommandArgumentHandling(t *testing.T) {
	tests := map[string]struct {
		args       []string
		wantErr    bool
		wantRemain bool
	}{
		"zero arguments":     {args: []string{"init"}},
		"one argument":       {args: []string{"init", "project"}},
		"too many arguments": {args: []string{"init", "one", "two"}, wantErr: true, wantRemain: true},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			root := t.TempDir()
			t.Chdir(root)
			t.Setenv(profilehome.EnvHome, filepath.Join(root, "global-home"))
			before := snapshotTree(root)
			_, err := executeCommandResult(t, test.args...)
			if (err != nil) != test.wantErr {
				t.Fatalf("error = %v, wantErr %t", err, test.wantErr)
			}
			if test.wantRemain && !reflect.DeepEqual(snapshotTree(root), before) {
				t.Fatal("too many arguments caused filesystem mutation")
			}
		})
	}
}

type treeEntry struct {
	directory bool
	mode      fs.FileMode
	data      []byte
}

func executeCommandResult(t *testing.T, args ...string) (string, error) {
	t.Helper()
	var out bytes.Buffer
	rootCmd.SetOut(&out)
	rootCmd.SetErr(&out)
	resetHomeFlag(t)
	resetNonInteractiveFlag(t)
	resetSubcommandFlags(rootCmd)
	rootCmd.SetArgs(args)
	err := rootCmd.Execute()
	return out.String(), err
}

func assertStarterTree(t *testing.T, destination string) {
	t.Helper()
	expected := map[string][]byte{
		"profiles/default/profile.yaml": []byte(expectedStarterProfile),
		"bindings/local.example.yaml":   []byte(expectedStarterBindings),
		"bindings/local.yaml":           []byte(expectedStarterBindings),
		"bindings/.gitignore":           []byte(expectedBindingsGitignore),
	}
	actual := starterFileSnapshot(t, destination)
	if !reflect.DeepEqual(actual, expected) {
		t.Fatalf("starter files = %v, want %v", actual, expected)
	}
	for path := range expected {
		info, err := os.Stat(filepath.Join(destination, path))
		if err != nil {
			t.Fatal(err)
		}
		if info.Mode().Perm() != 0o644 {
			t.Errorf("%s mode = %o, want 644", path, info.Mode().Perm())
		}
	}
}

func starterFileSnapshot(t *testing.T, destination string) map[string][]byte {
	t.Helper()
	files := make(map[string][]byte)
	for path, entry := range snapshotTree(destination) {
		if !entry.directory {
			files[path] = entry.data
		}
	}
	return files
}

func readStarterFiles(t *testing.T, destination string) map[string][]byte {
	t.Helper()
	files := make(map[string][]byte)
	for _, path := range []string{
		"profiles/default/profile.yaml",
		"bindings/local.example.yaml",
		"bindings/local.yaml",
		"bindings/.gitignore",
	} {
		data, err := os.ReadFile(filepath.Join(destination, path))
		if err != nil {
			t.Fatal(err)
		}
		files[path] = data
	}
	return files
}

func snapshotTree(root string) map[string]treeEntry {
	entries := make(map[string]treeEntry)
	_ = filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		relative, relErr := filepath.Rel(root, path)
		if relErr != nil || relative == "." {
			return relErr
		}
		info, infoErr := entry.Info()
		if infoErr != nil {
			return infoErr
		}
		item := treeEntry{directory: entry.IsDir(), mode: info.Mode()}
		if !entry.IsDir() {
			item.data, _ = os.ReadFile(path)
		}
		entries[relative] = item
		return nil
	})
	return entries
}

func TestInitStarterPlansReadyForClaudeCodeAndCodex(t *testing.T) {
	root := t.TempDir()
	home := filepath.Join(root, "home")
	t.Setenv("HOME", home)
	t.Setenv("CODEX_HOME", "")
	t.Setenv("CLAUDE_CONFIG_DIR", "")
	doctorFakeBinaries(t, nil)
	withAgentDetection(t)
	for _, folder := range []string{".claude", ".codex"} {
		if err := os.MkdirAll(filepath.Join(home, folder), 0o700); err != nil {
			t.Fatal(err)
		}
	}
	destination := filepath.Join(root, "project")
	if _, err := executeCommandResult(t, "init", destination); err != nil {
		t.Fatalf("init failed: %v", err)
	}
	var output bytes.Buffer
	cmd := &cobra.Command{}
	cmd.SetOut(&output)
	cmd.SetErr(&output)
	options := installOptions{profiles: filepath.Join(destination, "profiles"), resourceRoot: destination, bindings: filepath.Join(destination, "bindings", "local.yaml"), targets: []string{"claude-code", "codex"}}
	if err := runInstall(cmd, "default", options); err != nil {
		t.Fatalf("install after init: %v\n%s", err, output.String())
	}
	for _, want := range []string{" (ready)\n", "claude-code@", `"claude-sonnet-5"`, `"gpt-6-sol"`} {
		if !strings.Contains(output.String(), want) {
			t.Fatalf("fresh init plan lacks %q:\n%s", want, output.String())
		}
	}
}
