package installfs

import (
	"os"
	"path/filepath"
	"testing"
)

func TestRecoveryRejectsAlteredBackupBeforeAnyRestore(t *testing.T) {
	tests := map[string]struct{ alter func(string) error }{
		"changed bytes": {alter: func(path string) error { return os.WriteFile(path, []byte("corrupt"), 0o644) }},
		"missing":       {alter: os.Remove},
		"changed mode":  {alter: func(path string) error { return os.Chmod(path, 0o600) }},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			changes := recoveryChanges(t)
			options := ApplyOptions{PlanID: "backup-integrity", Backup: true, FaultAfter: 2, LeaveJournal: true}
			result, err := Apply(changes, options)
			if err == nil || result.Status != "recovery-required" {
				t.Fatalf("expected interrupted transaction, result=%#v err=%v", result, err)
			}
			if err := test.alter(BackupPath(changes[0].Path, options.PlanID)); err != nil {
				t.Fatal(err)
			}
			if err := Recover(result.JournalPath); err == nil {
				t.Fatal("recovery accepted an altered backup")
			}
			assertTestFile(t, changes[0].Path, "new-a")
			assertTestFile(t, changes[1].Path, "new-b")
			assertTestFile(t, changes[2].Path, "old-c")
		})
	}
}

func recoveryChanges(t *testing.T) []Change {
	t.Helper()
	root := t.TempDir()
	var changes []Change
	for _, name := range []string{"a", "b", "c"} {
		path := filepath.Join(root, name)
		writeTestFile(t, path, "old-"+name)
		before, err := SnapshotFile(path)
		if err != nil {
			t.Fatal(err)
		}
		changes = append(changes, Change{Path: path, Before: before, Content: []byte("new-" + name)})
	}
	return changes
}

func TestRollbackRejectsAlteredBackupBeforeAnyRestore(t *testing.T) {
	changes := recoveryChanges(t)
	options := ApplyOptions{PlanID: "rollback-integrity", Backup: true, FaultAfter: 2, LeaveJournal: true}
	result, err := Apply(changes, options)
	if err == nil || result.Status != "recovery-required" {
		t.Fatalf("expected interrupted transaction, result=%#v err=%v", result, err)
	}
	journal, err := readJournal(result.JournalPath)
	if err != nil {
		t.Fatal(err)
	}
	writeTestFile(t, journal.Entries[0].BackupPath, "corrupt")
	if err := rollback(changes, &journal); err == nil {
		t.Fatal("rollback accepted an altered backup")
	}
	assertTestFile(t, changes[0].Path, "new-a")
	assertTestFile(t, changes[1].Path, "new-b")
	assertTestFile(t, changes[2].Path, "old-c")
}
