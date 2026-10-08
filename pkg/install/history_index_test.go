package install

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ariel-frischer/profile-mango/internal/installfs"
)

func TestJournalIndexFailureLeavesTargetsUnchanged(t *testing.T) {
	test := undoTargetCases()["codex"]
	config, plan, root := planAtDefault(t, test, &test.original)
	plan.stateDir = filepath.Join(root, "state")
	writeInstallTestFile(t, filepath.Join(plan.stateDir, "targets"), "not a directory")
	report, err := ApplyPlan(plan, ApplyOptions{ExpectedPlanID: plan.PlanID})
	if err == nil || report.Status != StatusNotAttempted {
		t.Fatalf("apply with an unwritable journal index: report=%#v err=%v", report, err)
	}
	assertInstallTestFile(t, config, test.original)
	for _, path := range []string{config + ".profile-mango.manifest.json", installfs.LockPath(config), historyTransactions(plan.stateDir)} {
		if _, err := os.Lstat(path); !os.IsNotExist(err) {
			t.Fatalf("failed index left %s: %v", path, err)
		}
	}
}

func TestUndoSkipsReferenceOfUncommittedInstall(t *testing.T) {
	test := undoTargetCases()["codex"]
	config, first, _ := planAtDefault(t, test, &test.original)
	if _, err := ApplyPlan(first, ApplyOptions{ExpectedPlanID: first.PlanID}); err != nil {
		t.Fatal(err)
	}
	request, _ := testRequest(t, DefaultRegistry())
	writeInstallTestFile(t, request.BindingsPath, strings.Replace(test.bindings, "gpt-5.6", "gpt-5.6-mini", 1))
	request.Override, request.Default = true, true
	request.Targets = []TargetRequest{{Target: test.target(), ConfigPath: config}}
	second, err := BuildPlan(request)
	if err != nil || second.Status != StatusReady {
		t.Fatalf("second install status=%s err=%v", second.Status, err)
	}
	lock := installfs.LockPath(config)
	writeInstallTestFile(t, lock, "held by another installer\n")
	if _, err := ApplyPlan(second, ApplyOptions{ExpectedPlanID: second.PlanID}); err == nil {
		t.Fatal("install under a held lock accepted")
	}
	if err := os.Remove(lock); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(historyRefPath(testStateDir, config, second.PlanID)); err != nil {
		t.Fatalf("uncommitted install left no reference to skip: %v", err)
	}
	named := UndoRequest{Target: test.target(), ConfigPath: config, OriginalPlanID: second.PlanID, StateDir: testStateDir}
	if _, err := BuildUndoPlan(named); err == nil || !strings.Contains(err.Error(), "never committed") {
		t.Fatalf("undo of the uncommitted install: %v", err)
	}
	undone := applyUndo(t, UndoRequest{Target: test.target(), ConfigPath: config, StateDir: testStateDir})
	if undone.OriginalPlanID != first.PlanID {
		t.Fatalf("undo selected %s, want committed install %s", undone.OriginalPlanID, first.PlanID)
	}
	assertInstallTestFile(t, config, test.original)
}
