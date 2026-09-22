package installfs

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestSnapshotFileRejectsUnsafeEntries(t *testing.T) {
	root := t.TempDir()
	regular := filepath.Join(root, "regular")
	if err := os.WriteFile(regular, []byte("content"), 0o644); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(root, "link")
	if err := os.Symlink(regular, link); err != nil {
		t.Fatal(err)
	}
	if _, err := SnapshotFile(link); err == nil {
		t.Fatal("symlink was accepted")
	}
	hardlink := filepath.Join(root, "hardlink")
	if err := os.Link(regular, hardlink); err != nil {
		t.Fatal(err)
	}
	if _, err := SnapshotFile(hardlink); err == nil {
		t.Fatal("hardlink was accepted")
	}
	directory := filepath.Join(root, "directory")
	if err := os.Mkdir(directory, 0o755); err != nil {
		t.Fatal(err)
	}
	if _, err := SnapshotFile(directory); err == nil {
		t.Fatal("directory was accepted")
	}
}

func TestApplyRollsBackAppliedFilesAfterFault(t *testing.T) {
	root := t.TempDir()
	first := filepath.Join(root, "first")
	second := filepath.Join(root, "second")
	writeTestFile(t, first, "old-first")
	writeTestFile(t, second, "old-second")
	firstBefore, err := SnapshotFile(first)
	if err != nil {
		t.Fatal(err)
	}
	secondBefore, err := SnapshotFile(second)
	if err != nil {
		t.Fatal(err)
	}
	_, err = Apply([]Change{
		{Path: first, Before: firstBefore, Content: []byte("new-first")},
		{Path: second, Before: secondBefore, Content: []byte("new-second")},
	}, ApplyOptions{PlanID: "fault-plan", Backup: true, FaultAfter: 1})
	if err == nil {
		t.Fatal("fault was not reported")
	}
	assertTestFile(t, first, "old-first")
	assertTestFile(t, second, "old-second")
}

func TestApplyRollsBackAppliedDeleteAfterFault(t *testing.T) {
	root := t.TempDir()
	deleted := filepath.Join(root, "a-delete")
	updated := filepath.Join(root, "b-update")
	writeTestFile(t, deleted, "old-delete")
	writeTestFile(t, updated, "old-update")
	deletedBefore, err := SnapshotFile(deleted)
	if err != nil {
		t.Fatal(err)
	}
	updatedBefore, err := SnapshotFile(updated)
	if err != nil {
		t.Fatal(err)
	}
	_, err = Apply([]Change{
		{Path: deleted, Before: deletedBefore, Delete: true},
		{Path: updated, Before: updatedBefore, Content: []byte("new-update")},
	}, ApplyOptions{PlanID: "delete-fault-plan", Backup: true, FaultAfter: 1})
	if err == nil {
		t.Fatal("fault was not reported")
	}
	assertTestFile(t, deleted, "old-delete")
	assertTestFile(t, updated, "old-update")
}

func TestRecoverRestoresIncompleteJournal(t *testing.T) {
	root := t.TempDir()
	first := filepath.Join(root, "first")
	second := filepath.Join(root, "second")
	writeTestFile(t, first, "old-first")
	writeTestFile(t, second, "old-second")
	firstBefore, err := SnapshotFile(first)
	if err != nil {
		t.Fatal(err)
	}
	secondBefore, err := SnapshotFile(second)
	if err != nil {
		t.Fatal(err)
	}
	_, err = Apply([]Change{
		{Path: first, Before: firstBefore, Content: []byte("new-first")},
		{Path: second, Before: secondBefore, Content: []byte("new-second")},
	}, ApplyOptions{PlanID: "recovery-plan", Backup: true, FaultAfter: 1, LeaveJournal: true})
	if err == nil {
		t.Fatal("fault was not reported")
	}
	journal := JournalPath(first, "recovery-plan")
	if _, err := os.Stat(journal); err != nil {
		t.Fatalf("journal missing: %v", err)
	}
	if err := Recover(journal); err != nil {
		t.Fatal(err)
	}
	assertTestFile(t, first, "old-first")
	assertTestFile(t, second, "old-second")
}

func TestRecoverRestoresIncompleteDeleteJournal(t *testing.T) {
	root := t.TempDir()
	deleted := filepath.Join(root, "a-delete")
	updated := filepath.Join(root, "b-update")
	writeTestFile(t, deleted, "old-delete")
	writeTestFile(t, updated, "old-update")
	deletedBefore, err := SnapshotFile(deleted)
	if err != nil {
		t.Fatal(err)
	}
	updatedBefore, err := SnapshotFile(updated)
	if err != nil {
		t.Fatal(err)
	}
	result, err := Apply([]Change{
		{Path: deleted, Before: deletedBefore, Delete: true},
		{Path: updated, Before: updatedBefore, Content: []byte("new-update")},
	}, ApplyOptions{PlanID: "delete-recovery-plan", Backup: true, FaultAfter: 1, LeaveJournal: true})
	if err == nil || result.Status != "recovery-required" {
		t.Fatalf("expected interrupted transaction, result=%#v err=%v", result, err)
	}
	if _, err := os.Stat(deleted); !os.IsNotExist(err) {
		t.Fatalf("deleted path still exists before recovery: %v", err)
	}
	if err := Recover(result.JournalPath); err != nil {
		t.Fatal(err)
	}
	assertTestFile(t, deleted, "old-delete")
	assertTestFile(t, updated, "old-update")
}

func TestRecoverRejectsThirdPartyEdit(t *testing.T) {
	root := t.TempDir()
	first := filepath.Join(root, "first")
	second := filepath.Join(root, "second")
	writeTestFile(t, first, "old-first")
	writeTestFile(t, second, "old-second")
	firstBefore, err := SnapshotFile(first)
	if err != nil {
		t.Fatal(err)
	}
	secondBefore, err := SnapshotFile(second)
	if err != nil {
		t.Fatal(err)
	}
	_, err = Apply([]Change{
		{Path: first, Before: firstBefore, Content: []byte("new-first")},
		{Path: second, Before: secondBefore, Content: []byte("new-second")},
	}, ApplyOptions{PlanID: "guarded-recovery", Backup: true, FaultAfter: 1, LeaveJournal: true})
	if err == nil {
		t.Fatal("fault was not reported")
	}
	writeTestFile(t, first, "third-party")
	journal := JournalPath(first, "guarded-recovery")
	if err := Recover(journal); err == nil {
		t.Fatal("recovery overwrote third-party state")
	}
	assertTestFile(t, first, "third-party")
	assertTestFile(t, second, "old-second")
}

func TestRecoverRejectsThirdPartyRecreationOfDeletedFile(t *testing.T) {
	root := t.TempDir()
	deleted := filepath.Join(root, "a-delete")
	updated := filepath.Join(root, "b-update")
	writeTestFile(t, deleted, "old-delete")
	writeTestFile(t, updated, "old-update")
	deletedBefore, err := SnapshotFile(deleted)
	if err != nil {
		t.Fatal(err)
	}
	updatedBefore, err := SnapshotFile(updated)
	if err != nil {
		t.Fatal(err)
	}
	result, err := Apply([]Change{
		{Path: deleted, Before: deletedBefore, Delete: true},
		{Path: updated, Before: updatedBefore, Content: []byte("new-update")},
	}, ApplyOptions{PlanID: "delete-guarded-recovery", Backup: true, FaultAfter: 1, LeaveJournal: true})
	if err == nil || result.Status != "recovery-required" {
		t.Fatalf("expected interrupted transaction, result=%#v err=%v", result, err)
	}
	writeTestFile(t, deleted, "third-party")
	if err := Recover(result.JournalPath); err == nil {
		t.Fatal("recovery overwrote third-party recreation")
	}
	assertTestFile(t, deleted, "third-party")
	assertTestFile(t, updated, "old-update")
}

func TestApplyRejectsStaleSnapshotBeforeWrites(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "config")
	writeTestFile(t, path, "old")
	before, err := SnapshotFile(path)
	if err != nil {
		t.Fatal(err)
	}
	writeTestFile(t, path, "changed")
	_, err = Apply([]Change{{Path: path, Before: before, Content: []byte("new")}}, ApplyOptions{PlanID: "stale-plan", Backup: true})
	if !errors.Is(err, ErrStale) {
		t.Fatalf("error = %v, want stale error", err)
	}
	assertTestFile(t, path, "changed")
}

func writeTestFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func assertTestFile(t *testing.T, path, want string) {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != want {
		t.Fatalf("%s = %q, want %q", path, data, want)
	}
}
