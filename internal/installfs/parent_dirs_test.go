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

func TestApplyRollbackKeepsCreatedParentDirectories(t *testing.T) {
	root := t.TempDir()
	config := filepath.Join(root, "config.json")
	nested := filepath.Join(root, "a-profiles", "coding.json")
	writeTestFile(t, config, "old")
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
	if _, err := os.Lstat(nested); !os.IsNotExist(err) {
		t.Fatalf("created file survived rollback: %v", err)
	}
	if info, err := os.Stat(filepath.Dir(nested)); err != nil || !info.IsDir() {
		t.Fatalf("created directory was removed: %v", err)
	}
}

func TestApplyRefusesSymlinkedParentDirectory(t *testing.T) {
	root := t.TempDir()
	outside := t.TempDir()
	if err := os.Symlink(outside, filepath.Join(root, "profiles")); err != nil {
		t.Fatal(err)
	}
	if _, err := SnapshotFile(filepath.Join(root, "profiles", "missing", "coding.json")); err == nil {
		t.Fatal("symlinked parent was accepted")
	}
}
