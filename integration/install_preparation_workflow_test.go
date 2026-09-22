package integration_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"gitlab.com/ariel-frischer/profile-mango/internal/installfs"
	"gitlab.com/ariel-frischer/profile-mango/pkg/install"
)

func TestInstalledBinaryPreparationFailureRetry(t *testing.T) {
	tests := map[string]struct {
		blocker func(installWorkflow, install.Plan) string
	}{
		"later backup collision": {func(w installWorkflow, plan install.Plan) string {
			return installfs.BackupPath(w.config+".profile-mango.manifest.json", plan.PlanID)
		}},
		"initial journal obstruction": {func(w installWorkflow, plan install.Plan) string {
			return installfs.JournalPath(w.config, plan.PlanID)
		}},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			w, plan := preparationRetryWorkflow(t)
			blocker := test.blocker(w, plan)
			if err := os.Mkdir(blocker, 0o700); err != nil {
				t.Fatal(err)
			}
			assertPreparationFailurePreserved(t, w, plan, blocker)
			if err := os.Remove(blocker); err != nil {
				t.Fatal(err)
			}
			if retry := w.plan(t); retry.PlanID != plan.PlanID {
				t.Fatal("removing only external blocker changed plan ID")
			}
			applyPreparationPlan(t, w, plan)
		})
	}
}

func preparationRetryWorkflow(t *testing.T) (installWorkflow, install.Plan) {
	t.Helper()
	w := newInstallWorkflow(t, "opencode@1.18.31", "openai", "gpt-5.6",
		"{\"model\":\"openai/old\",\"keep\":true}\n", "")
	w.args = append(w.args, "--manifest", "opencode@1.18.31="+w.config+".profile-mango.manifest.json")
	applyPreparationPlan(t, w, w.plan(t))
	bindings := "routes:\n  primary:\n    provider: openai\n    transport: native\n    authentication: oauth\n    model: gpt-5.6-next\n    effort: high\n"
	if err := os.WriteFile(filepath.Join(w.root, "bindings.yaml"), []byte(bindings), 0o600); err != nil {
		t.Fatal(err)
	}
	plan := w.plan(t)
	if plan.Status != install.StatusReady || len(plan.Targets) != 1 || len(plan.Targets[0].Files) != 2 {
		t.Fatalf("expected ready config and manifest update: %#v", plan)
	}
	return w, plan
}

func assertPreparationFailurePreserved(t *testing.T, w installWorkflow, plan install.Plan, blocker string) {
	t.Helper()
	paths := []string{w.config, w.config + ".profile-mango.manifest.json"}
	before := make(map[string]installfs.Snapshot, len(paths))
	for _, path := range paths {
		snapshot, err := installfs.SnapshotFile(path)
		if err != nil || !snapshot.Exists {
			t.Fatalf("snapshot %s: %v", path, err)
		}
		before[path] = snapshot
	}
	result := w.run(t, "--apply", "--yes", "--expect-plan", plan.PlanID)
	if result.err == nil {
		t.Fatal("preparation blocker did not reject apply")
	}
	for path, snapshot := range before {
		current, err := installfs.SnapshotFile(path)
		if err != nil || !snapshot.Equal(current) {
			t.Fatalf("preparation changed %s: %v", path, err)
		}
		backup := installfs.BackupPath(path, plan.PlanID)
		if backup != blocker {
			if _, err := os.Lstat(backup); !os.IsNotExist(err) {
				t.Fatalf("preparation backup remained %s: %v", backup, err)
			}
		}
	}
	if info, err := os.Lstat(blocker); err != nil || !info.IsDir() {
		t.Fatalf("external blocker changed: %v", err)
	}
	assertWorkflowBytes(t, filepath.Join(w.root, "outside-sentinel"), []byte("untouched"))
}

func applyPreparationPlan(t *testing.T, w installWorkflow, plan install.Plan) {
	t.Helper()
	result := w.run(t, "--apply", "--yes", "--expect-plan", plan.PlanID)
	var report install.ApplyReport
	if result.err != nil || json.Unmarshal([]byte(result.stdout), &report) != nil || report.Status != "committed" {
		t.Fatalf("expected one committed apply report: %v\n%s\n%s", result.err, result.stdout, result.stderr)
	}
}
