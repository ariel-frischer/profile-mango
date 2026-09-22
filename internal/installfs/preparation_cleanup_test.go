package installfs

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestApplyPreparationFailureCleansOwnedBackups(t *testing.T) {
	tests := map[string]struct {
		block func(t *testing.T, root string, changes []Change, options ApplyOptions) func()
		want  string
	}{
		"later backup collision": {
			block: blockLaterBackup,
			want:  "backup already exists",
		},
		"initial journal write failure": {
			block: blockJournalPath,
			want:  "rename",
		},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			root, changes, options := preparationChanges(t)
			removeBlocker := test.block(t, root, changes, options)
			firstBefore, secondBefore := changes[0].Before, changes[1].Before

			if _, err := Apply(changes, options); err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("Apply error = %v, want %q", err, test.want)
			}
			assertSnapshotUnchanged(t, firstBefore)
			assertSnapshotUnchanged(t, secondBefore)
			if _, err := os.Stat(BackupPath(changes[0].Path, options.PlanID)); !os.IsNotExist(err) {
				t.Fatalf("first backup remains after preparation failure: %v", err)
			}
			if name == "later backup collision" {
				assertTestFile(t, BackupPath(changes[1].Path, options.PlanID), "external-backup")
			} else if info, err := os.Stat(options.JournalPath); err != nil || !info.IsDir() {
				t.Fatalf("journal blocker was not preserved: info=%v err=%v", info, err)
			}
			removeBlocker()

			result, err := Apply(changes, options)
			if err != nil || result.Status != "committed" {
				t.Fatalf("same-plan retry result=%#v err=%v", result, err)
			}
			assertTestFile(t, changes[0].Path, "new-first")
			assertTestFile(t, changes[1].Path, "new-second")
		})
	}
}

func TestCleanupPreparedBackupsPreservesChangedArtifacts(t *testing.T) {
	tests := map[string]struct {
		mutate func(t *testing.T, path string)
	}{
		"replacement": {
			mutate: replaceBackup,
		},
		"modified bytes": {
			mutate: func(t *testing.T, path string) {
				if err := os.WriteFile(path, []byte("modified"), 0o644); err != nil {
					t.Fatal(err)
				}
			},
		},
		"changed mode": {
			mutate: func(t *testing.T, path string) {
				if err := os.Chmod(path, 0o600); err != nil {
					t.Fatal(err)
				}
			},
		},
		"symlink": {
			mutate: replaceWithSymlink,
		},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "backup")
			writeTestFile(t, path, "original")
			before, err := SnapshotFile(path)
			if err != nil {
				t.Fatal(err)
			}
			test.mutate(t, path)

			err = cleanupPreparedBackups([]preparedBackup{{path: path, snapshot: before, verified: true}})
			if err == nil {
				t.Fatal("changed backup was removed without an error")
			}
			if _, statErr := os.Lstat(path); statErr != nil {
				t.Fatalf("changed backup was not preserved: %v", statErr)
			}
		})
	}
}

func TestCleanupPreparedBackupsReportsUnverifiedArtifact(t *testing.T) {
	path := filepath.Join(t.TempDir(), "backup")
	writeTestFile(t, path, "uncertain")

	err := cleanupPreparedBackups([]preparedBackup{{path: path}})
	if err == nil || !strings.Contains(err.Error(), "unverified") {
		t.Fatalf("cleanup error = %v, want unverified ownership", err)
	}
	assertTestFile(t, path, "uncertain")
}

func TestPreparationFailureJoinsOriginalAndCleanupErrors(t *testing.T) {
	path := filepath.Join(t.TempDir(), "backup")
	writeTestFile(t, path, "original")
	before, err := SnapshotFile(path)
	if err != nil {
		t.Fatal(err)
	}
	originalErr := errors.New("original preparation failure")
	cleanupErr := errors.New("cleanup blocked")

	err = preparationFailureWith(originalErr, []preparedBackup{{path: path, snapshot: before, verified: true}}, func(string) error {
		return cleanupErr
	})
	if !errors.Is(err, originalErr) || !errors.Is(err, cleanupErr) {
		t.Fatalf("joined error = %v, want both causes", err)
	}
	assertTestFile(t, path, "original")
}

func preparationChanges(t *testing.T) (string, []Change, ApplyOptions) {
	t.Helper()
	root := t.TempDir()
	first := filepath.Join(root, "target")
	second := filepath.Join(root, "manifest")
	writeTestFile(t, first, "old-target")
	writeTestFile(t, second, "old-manifest")
	beforeFirst, err := SnapshotFile(first)
	if err != nil {
		t.Fatal(err)
	}
	beforeSecond, err := SnapshotFile(second)
	if err != nil {
		t.Fatal(err)
	}
	changes := []Change{
		{Path: first, Before: beforeFirst, Content: []byte("new-first")},
		{Path: second, Before: beforeSecond, Content: []byte("new-second")},
	}
	return root, changes, ApplyOptions{
		PlanID:      "preparation-retry",
		Backup:      true,
		JournalPath: JournalPath(first, "preparation-retry"),
	}
}

func blockLaterBackup(t *testing.T, _ string, changes []Change, options ApplyOptions) func() {
	t.Helper()
	path := BackupPath(changes[1].Path, options.PlanID)
	writeTestFile(t, path, "external-backup")
	return func() { _ = os.Remove(path) }
}

func blockJournalPath(t *testing.T, _ string, _ []Change, options ApplyOptions) func() {
	t.Helper()
	path := options.JournalPath
	if err := os.Mkdir(path, 0o755); err != nil {
		t.Fatal(err)
	}
	return func() { _ = os.RemoveAll(path) }
}

func assertSnapshotUnchanged(t *testing.T, want Snapshot) {
	t.Helper()
	got, err := SnapshotFile(want.Path)
	if err != nil {
		t.Fatal(err)
	}
	if !got.Equal(want) || !bytes.Equal(got.Content, want.Content) {
		t.Fatalf("%s changed: got=%#v want=%#v", want.Path, got, want)
	}
}

func replaceBackup(t *testing.T, path string) {
	t.Helper()
	temporary := filepath.Join(filepath.Dir(path), "replacement")
	writeTestFile(t, temporary, "original")
	if err := os.Rename(temporary, path); err != nil {
		t.Fatal(err)
	}
}

func replaceWithSymlink(t *testing.T, path string) {
	t.Helper()
	target := filepath.Join(filepath.Dir(path), "target")
	writeTestFile(t, target, "symlink-target")
	if err := os.Remove(path); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(target, path); err != nil {
		t.Fatal(err)
	}
}
