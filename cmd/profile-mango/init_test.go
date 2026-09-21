package main

import (
	"bytes"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"gitlab.com/ariel-frischer/profile-mango/pkg/profilemango"
)

const expectedStarterProfile = `apiVersion: profilemango.dev/v1alpha1
kind: PolicyProfile
metadata:
  name: default
  description: Default profile scaffolded by profile-mango init
spec:
  routeRef: local
`

const expectedStarterBindings = `# Route identity only. Credentials remain target-owned and are never stored here.
routes:
  local:
    provider: openai
    transport: native
    authentication: oauth
    model: gpt-5.6
    effort: high
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
		"default directory": {
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

	profile, profileDiagnostics := profilemango.ParseProfile(firstFiles["profiles/default/profile.yaml"])
	if profileDiagnostics.HasErrors() {
		t.Fatalf("generated profile diagnostics = %v", profileDiagnostics.Sorted())
	}
	bindings, bindingDiagnostics := profilemango.ParseBindings(firstFiles["bindings/local.example.yaml"])
	if bindingDiagnostics.HasErrors() {
		t.Fatalf("generated binding diagnostics = %v", bindingDiagnostics.Sorted())
	}
	if _, found := bindings.Routes[profile.Spec.RouteRef]; !found {
		t.Fatalf("routeRef %q is absent from generated bindings", profile.Spec.RouteRef)
	}
	if strings.Contains(string(firstFiles["bindings/local.example.yaml"]), "token") || strings.Contains(string(firstFiles["bindings/local.example.yaml"]), "secret") {
		t.Fatal("generated bindings contain token or secret guidance")
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

func TestInitAndConfigInitRemainDistinctCommands(t *testing.T) {
	initCommand, _, err := rootCmd.Find([]string{"init"})
	if err != nil {
		t.Fatalf("find root init: %v", err)
	}
	if initCommand == nil || initCommand.Use != "init [directory]" {
		t.Fatalf("root init command = %#v, want Use init [directory]", initCommand)
	}
	configInit, _, err := rootCmd.Find([]string{"config", "init"})
	if err != nil {
		t.Fatalf("find config init: %v", err)
	}
	if configInit == nil || configInit == initCommand || configInit.Use != "init" {
		t.Fatalf("config init command = %#v, want distinct Use init command", configInit)
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
	rootCmd.SetArgs(args)
	configPathOverride = ""
	if flag := rootCmd.PersistentFlags().Lookup("config"); flag != nil {
		_ = flag.Value.Set("")
		flag.Changed = false
	}
	err := rootCmd.Execute()
	return out.String(), err
}

func assertStarterTree(t *testing.T, destination string) {
	t.Helper()
	expected := map[string][]byte{
		"profiles/default/profile.yaml": []byte(expectedStarterProfile),
		"bindings/local.example.yaml":   []byte(expectedStarterBindings),
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
