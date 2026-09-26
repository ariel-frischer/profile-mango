package installfs

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestApplyCreatesMissingParentDirectories(t *testing.T) {
	root := t.TempDir()
	nested := filepath.Join(root, "a-profiles", "coding", "config.json")
	config := filepath.Join(root, "config.json")
	writeTestFile(t, config, "old")
	nestedBefore, err := SnapshotFile(nested)
	if err != nil || nestedBefore.Exists {
		t.Fatalf("SnapshotFile(missing parent) = %+v, %v", nestedBefore, err)
	}
	configBefore, err := SnapshotFile(config)
	if err != nil {
		t.Fatal(err)
	}
	result, err := Apply([]Change{
		{Path: nested, Before: nestedBefore, Content: []byte("profile")},
		{Path: config, Before: configBefore, Content: []byte("new")},
	}, ApplyOptions{PlanID: "parent-plan", Backup: true})
	if err != nil {
		t.Fatal(err)
	}
	assertTestFile(t, nested, "profile")
	if !strings.HasPrefix(result.JournalPath, config) {
		t.Fatalf("journal %s is not anchored at an existing directory", result.JournalPath)
	}
	info, err := os.Stat(filepath.Dir(nested))
	if err != nil || info.Mode().Perm() != 0o700 {
		t.Fatalf("created directory = %v, %v", info, err)
	}
}

// Rollback removes the directories the failed transaction created, but not ones it found.
func TestApplyRollbackRemovesOnlyCreatedParentDirectories(t *testing.T) {
	root := t.TempDir()
	config := filepath.Join(root, "config.json")
	existing := filepath.Join(root, "a-profiles")
	nested := filepath.Join(existing, "coding", "deeper", "profile.json")
	writeTestFile(t, config, "old")
	if err := os.Mkdir(existing, 0o700); err != nil {
		t.Fatal(err)
	}
	configBefore, err := SnapshotFile(config)
	if err != nil {
		t.Fatal(err)
	}
	nestedBefore, err := SnapshotFile(nested)
	if err != nil {
		t.Fatal(err)
	}
	_, err = Apply([]Change{
		{Path: config, Before: configBefore, Content: []byte("new")},
		{Path: nested, Before: nestedBefore, Content: []byte("profile")},
	}, ApplyOptions{PlanID: "parent-fault", Backup: true, FaultAfter: 1})
	if err == nil {
		t.Fatal("fault was not reported")
	}
	assertTestFile(t, config, "old")
	if _, err := os.Lstat(filepath.Join(existing, "coding")); !os.IsNotExist(err) {
		t.Fatalf("created directory survived rollback: %v", err)
	}
	if info, err := os.Stat(existing); err != nil || !info.IsDir() {
		t.Fatalf("pre-existing directory was removed: %v", err)
	}
}

// A delete removes the directories an earlier transaction created for the file while they
// are empty, keeping its backup outside them; content added there since keeps them.
func TestApplyDeleteRemovesEmptyCreatedParentDirectories(t *testing.T) {
	tests := map[string]struct {
		extra   string
		removed bool
	}{
		"empty after delete": {removed: true},
		"other file remains": {extra: "memory.db"},
		"nested dir remains": {extra: filepath.Join("skills", "note.md")},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			root := t.TempDir()
			config := filepath.Join(root, "config.yaml")
			dir := filepath.Join(root, "profiles", "coding")
			named := filepath.Join(dir, "config.yaml")
			writeTestFile(t, config, "main")
			created := applyCreate(t, config, named)
			if want := []string{dir, filepath.Dir(dir)}; strings.Join(created, "|") != strings.Join(want, "|") {
				t.Fatalf("journal created dirs = %v, want %v", created, want)
			}
			if test.extra != "" {
				if err := os.MkdirAll(filepath.Dir(filepath.Join(dir, test.extra)), 0o700); err != nil {
					t.Fatal(err)
				}
				writeTestFile(t, filepath.Join(dir, test.extra), "user state")
			}
			before, err := SnapshotFile(named)
			if err != nil {
				t.Fatal(err)
			}
			// The sibling sorts after the profile file, so this also checks the lock is not
			// anchored inside a directory the delete removes.
			sibling := filepath.Join(root, "zz-sibling")
			changes := []Change{{Path: named, Before: before, Delete: true, RemoveEmptyDirs: created}, {Path: sibling, Content: []byte("x")}}
			if _, err := Apply(changes, ApplyOptions{PlanID: "undo-plan", Backup: true, JournalPath: config + ".undo.json"}); err != nil {
				t.Fatal(err)
			}
			if _, err := os.Lstat(dir); os.IsNotExist(err) != test.removed {
				t.Fatalf("profile dir removed=%v, want %v", os.IsNotExist(err), test.removed)
			}
			assertTestFile(t, config+".undo.bak.0", "profile")
		})
	}
}

// applyCreate creates path beside an update to config and returns the directories it journaled.
func applyCreate(t *testing.T, config, path string) []string {
	t.Helper()
	before, err := SnapshotFile(path)
	if err != nil {
		t.Fatal(err)
	}
	configBefore, err := SnapshotFile(config)
	if err != nil {
		t.Fatal(err)
	}
	changes := []Change{{Path: config, Before: configBefore, Content: []byte("main2")}, {Path: path, Before: before, Content: []byte("profile")}}
	result, err := Apply(changes, ApplyOptions{PlanID: "create-plan", Backup: true})
	if err != nil {
		t.Fatal(err)
	}
	journal, err := readJournal(result.JournalPath)
	if err != nil {
		t.Fatal(err)
	}
	return journal.Entries[1].CreatedDirs
}
