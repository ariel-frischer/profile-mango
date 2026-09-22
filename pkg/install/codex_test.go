package install

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gitlab.com/ariel-frischer/profile-mango/internal/installfs"
)

func TestCodexInstallPreservesStateBacksUpAndReapplies(t *testing.T) {
	request, _ := codexTestRequest(t)
	config := request.Targets[0].ConfigPath
	before := "# keep\nunknown = true\n[features]\napps = false\n"
	writeInstallTestFile(t, config, before)
	request.Override = true
	if err := os.Chmod(config, 0o600); err != nil {
		t.Fatal(err)
	}
	plan, err := BuildPlan(request)
	if err != nil || plan.Status != StatusReady {
		t.Fatalf("plan = %#v, err=%v", plan, err)
	}
	if !plan.Backup || !hasFileAction(plan.Targets[0], ActionOverride) {
		t.Fatalf("plan backup/action = %v/%#v", plan.Backup, plan.Targets[0].Files)
	}
	if len(plan.Targets[0].Fields) != 3 {
		t.Fatalf("plan fields = %#v", plan.Targets[0].Fields)
	}
	if _, err := ApplyPlan(plan, ApplyOptions{ExpectedPlanID: plan.PlanID}); err != nil {
		t.Fatal(err)
	}
	content, err := os.ReadFile(config)
	if err != nil {
		t.Fatal(err)
	}
	want := "# keep\nunknown = true\nmodel_provider = \"openai\"\nmodel = \"gpt-5.6\"\nmodel_reasoning_effort = \"high\"\n[features]\napps = false\n"
	if string(content) != want {
		t.Fatalf("installed config = %q, want %q", content, want)
	}
	info, err := os.Stat(config)
	if err != nil || info.Mode().Perm() != 0o600 {
		t.Fatalf("config mode = %v, err=%v", info.Mode().Perm(), err)
	}
	backup, err := os.ReadFile(installfs.BackupPath(config, plan.PlanID))
	if err != nil || string(backup) != before {
		t.Fatalf("backup = %q, err=%v", backup, err)
	}
	reapply, err := BuildPlan(request)
	if err != nil || reapply.Status != StatusNoop || !hasFileAction(reapply.Targets[0], ActionNoop) {
		t.Fatalf("reapply = %#v, err=%v", reapply, err)
	}
}

func TestCodexInstallRejectsUnownedConflictAndAllowsOverride(t *testing.T) {
	request, _ := codexTestRequest(t)
	config := request.Targets[0].ConfigPath
	writeInstallTestFile(t, config, "unknown = true\n")
	conflict, err := BuildPlan(request)
	if err != nil || conflict.Targets[0].Status != StatusConflict {
		t.Fatalf("conflict = %#v, err=%v", conflict, err)
	}
	request.Override = true
	override, err := BuildPlan(request)
	if err != nil || override.Status != StatusReady || !hasFileAction(override.Targets[0], ActionOverride) {
		t.Fatalf("override = %#v, err=%v", override, err)
	}
}

func TestCodexInstallRejectsStalePlanWithoutWriting(t *testing.T) {
	request, _ := codexTestRequest(t)
	request.Override = true
	config := request.Targets[0].ConfigPath
	writeInstallTestFile(t, config, "unknown = true\n")
	plan, err := BuildPlan(request)
	if err != nil || plan.Status != StatusReady {
		t.Fatalf("plan = %#v, err=%v", plan, err)
	}
	writeInstallTestFile(t, config, "third-party = true\n")
	if _, err := ApplyPlan(plan, ApplyOptions{ExpectedPlanID: plan.PlanID}); !errors.Is(err, installfs.ErrStale) {
		t.Fatalf("stale apply error = %v", err)
	}
	assertInstallTestFile(t, config, "third-party = true\n")
}

func TestCodexInstallBlocksPermissionRequirements(t *testing.T) {
	request, _ := codexTestRequest(t)
	profile := filepath.Join(request.ProfilesRoot, "route-only", "profile.yaml")
	writeInstallTestFile(t, profile, `apiVersion: profilemango.dev/v1alpha1
kind: PolicyProfile
metadata:
  name: route-only
spec:
  routeRef: primary
  permissions:
    mode: read-only
`)
	plan, err := BuildPlan(request)
	if err != nil || plan.Status != StatusBlocked || !strings.Contains(plan.Targets[0].Reason, "permission requirements") {
		t.Fatalf("plan = %#v, err=%v", plan, err)
	}
}

func TestCodexInstallRestoresSyntheticConfigThroughTransactionEngine(t *testing.T) {
	request, _ := codexTestRequest(t)
	request.Override = true
	config := request.Targets[0].ConfigPath
	before := "model = \"old/model\"\n"
	writeInstallTestFile(t, config, before)
	original, err := installfs.SnapshotFile(config)
	if err != nil {
		t.Fatal(err)
	}
	plan, err := BuildPlan(request)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := ApplyPlan(plan, ApplyOptions{ExpectedPlanID: plan.PlanID}); err != nil {
		t.Fatal(err)
	}
	current, err := installfs.SnapshotFile(config)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := installfs.Apply([]installfs.Change{{Path: config, Before: current, Content: []byte(before)}}, installfs.ApplyOptions{PlanID: "codex-restore", Backup: false}); err != nil {
		t.Fatal(err)
	}
	restored, err := installfs.SnapshotFile(config)
	if err != nil || restored.SHA256 != original.SHA256 || restored.Mode != original.Mode || string(restored.Content) != before {
		t.Fatalf("restored = %#v, err=%v", restored, err)
	}
}

func codexTestRequest(t *testing.T) (Request, string) {
	t.Helper()
	request, root := testRequest(t, NewRegistry(codexAdapter{}))
	writeInstallTestFile(t, request.BindingsPath, `routes:
  primary:
    provider: openai
    transport: native
    authentication: oauth
    model: gpt-5.6
    effort: high
`)
	config := filepath.Join(root, "target", "config.toml")
	if err := os.MkdirAll(filepath.Dir(config), 0o755); err != nil {
		t.Fatal(err)
	}
	request.Targets = []TargetRequest{{Target: Target{Name: "codex", Version: "0.154.0"}, ConfigPath: config}}
	return request, root
}

func hasFileAction(target TargetPlan, action string) bool {
	for _, file := range target.Files {
		if file.Action == action {
			return true
		}
	}
	return false
}
