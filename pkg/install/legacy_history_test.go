package install

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ariel-frischer/profile-mango/internal/installfs"
)

// applyLegacyPlan commits plan the way releases before the Mango state directory did,
// with the journal and backups beside the files they protect.
func applyLegacyPlan(t *testing.T, plan Plan) {
	t.Helper()
	var changes []installfs.Change
	var anchors []string
	for _, target := range sortedTargetPlans(plan.Targets) {
		changes = append(changes, target.changes...)
		anchors = append(anchors, target.ConfigPath)
	}
	if _, err := installfs.Apply(changes, installfs.ApplyOptions{PlanID: plan.PlanID, Backup: plan.Backup, Anchors: anchors}); err != nil {
		t.Fatal(err)
	}
}

// assertNoHistoryBesideTargets fails if any installer history file sits under root.
func assertNoHistoryBesideTargets(t *testing.T, root string) {
	t.Helper()
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() && path == testStateDir {
			return filepath.SkipDir
		}
		if name := entry.Name(); strings.Contains(name, ".profile-mango.") && !strings.HasSuffix(name, ".profile-mango.manifest.json") {
			t.Errorf("installer history beside targets: %s", path)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestInstallAndUndoKeepHistoryInStateDir(t *testing.T) {
	for name, test := range undoTargetCases() {
		t.Run(name, func(t *testing.T) {
			config, plan, root := planAtDefault(t, test, &test.original)
			if _, err := ApplyPlan(plan, ApplyOptions{ExpectedPlanID: plan.PlanID}); err != nil {
				t.Fatal(err)
			}
			assertNoHistoryBesideTargets(t, root)
			assertInstallTestFile(t, installedBackup(t, plan.PlanID, config), test.original)
			applyUndo(t, UndoRequest{Target: test.target(), ConfigPath: config, StateDir: testStateDir})
			assertInstallTestFile(t, config, test.original)
			assertNoHistoryBesideTargets(t, root)
		})
	}
}

func TestApplyRequiresStateDir(t *testing.T) {
	test := undoTargetCases()["codex"]
	config, plan, _ := planAtDefault(t, test, &test.original)
	plan.stateDir = ""
	if _, err := ApplyPlan(plan, ApplyOptions{ExpectedPlanID: plan.PlanID}); err == nil {
		t.Fatal("install without a state directory accepted")
	}
	assertInstallTestFile(t, config, test.original)
}

func TestUndoReadsLegacyHistoryBesideTarget(t *testing.T) {
	test := undoTargetCases()["codex"]
	config, plan, _ := planAtDefault(t, test, &test.original)
	applyLegacyPlan(t, plan)
	for _, path := range []string{installfs.JournalPath(config, plan.PlanID), installfs.BackupPath(config, plan.PlanID)} {
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("legacy history missing: %v", err)
		}
	}
	undone := applyUndo(t, UndoRequest{Target: test.target(), ConfigPath: config, StateDir: testStateDir})
	if undone.OriginalPlanID != plan.PlanID {
		t.Fatalf("undo selected %s, want legacy install %s", undone.OriginalPlanID, plan.PlanID)
	}
	assertInstallTestFile(t, config, test.original)
	if _, err := BuildUndoPlan(UndoRequest{Target: test.target(), ConfigPath: config, StateDir: testStateDir}); err == nil {
		t.Fatal("second undo of the legacy install accepted")
	}
}

func TestReleaseRestoresAdoptedFileFromLegacyBackup(t *testing.T) {
	request, _, agents := globalTestRequest(t, "    AGENTS.md: instructions.md\n")
	writeInstallTestFile(t, agents, globalTestOriginal)
	plan, err := BuildPlan(request)
	if err != nil || plan.Status != StatusReady {
		t.Fatalf("install plan status=%s err=%v", plan.Status, err)
	}
	applyLegacyPlan(t, plan)
	assertInstallTestFile(t, installfs.BackupPath(agents, plan.PlanID), globalTestOriginal)
	assertInstallTestFile(t, agents, globalTestWork)
	request.ProfileName, request.Release = "plain", true
	applySwitchTestPlan(t, request)
	assertInstallTestFile(t, agents, globalTestOriginal)
}
