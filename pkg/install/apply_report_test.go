package install

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gitlab.com/ariel-frischer/profile-mango/internal/installfs"
)

func TestApplyPreflightReportPreservesNoopAndSortsTargets(t *testing.T) {
	registry := NewRegistry(
		testAdapterFor("alpha", "1", "alpha", true),
		testAdapterFor("zeta", "1", "zeta", true),
	)
	request, root := testRequest(t, registry)
	request.Override = true
	alpha := filepath.Join(root, "alpha")
	writeInstallTestFile(t, alpha, "alpha")
	request.Targets = []TargetRequest{{Target: Target{Name: "alpha", Version: "1"}, ConfigPath: alpha}}
	first, err := BuildPlan(request)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := ApplyPlan(first, ApplyOptions{ExpectedPlanID: first.PlanID}); err != nil {
		t.Fatal(err)
	}

	zeta := filepath.Join(root, "zeta")
	writeInstallTestFile(t, zeta, "old-zeta")
	request.Targets = []TargetRequest{
		{Target: Target{Name: "zeta", Version: "1"}, ConfigPath: zeta},
		{Target: Target{Name: "alpha", Version: "1"}, ConfigPath: alpha},
	}
	plan, err := BuildPlan(request)
	if err != nil {
		t.Fatal(err)
	}
	if plan.Status != StatusReady || len(plan.Targets) != 2 {
		t.Fatalf("plan = %#v", plan)
	}

	report, err := ApplyPlan(plan, ApplyOptions{ExpectedPlanID: "wrong-consent"})
	if err == nil || !strings.Contains(err.Error(), "does not match actual plan") {
		t.Fatalf("apply report=%#v err=%v", report, err)
	}
	if report.Status != StatusNotAttempted || len(report.Targets) != 2 {
		t.Fatalf("preflight report = %#v", report)
	}
	if report.Targets[0].Target != "alpha@1" || report.Targets[0].Status != StatusNoop || report.Targets[0].Error != "" {
		t.Fatalf("noop result = %#v", report.Targets[0])
	}
	if report.Targets[1].Target != "zeta@1" || report.Targets[1].Status != StatusNotAttempted || report.Targets[1].Error == "" {
		t.Fatalf("pending result = %#v", report.Targets[1])
	}
	assertInstallTestFile(t, zeta, "old-zeta")
	assertAbsentInstallArtifact(t, installfs.BackupPath(zeta, plan.PlanID))
	assertAbsentInstallArtifact(t, installfs.JournalPath(zeta, plan.PlanID))
	assertAbsentInstallArtifact(t, zeta+".profile-mango.lock")
}

func TestApplyPreflightReportOnStaleSource(t *testing.T) {
	registry := NewRegistry(testAdapterFor("fake", "1", "new", true))
	request, root := testRequest(t, registry)
	config := filepath.Join(root, "config")
	request.Override = true
	request.Targets[0] = TargetRequest{Target: Target{Name: "fake", Version: "1"}, ConfigPath: config}
	plan, err := BuildPlan(request)
	if err != nil {
		t.Fatal(err)
	}
	profile := filepath.Join(request.ProfilesRoot, "route-only", "profile.yaml")
	writeInstallTestFile(t, profile, "changed source")

	report, err := ApplyPlan(plan, ApplyOptions{ExpectedPlanID: plan.PlanID})
	if err == nil || !strings.Contains(err.Error(), "source") {
		t.Fatalf("apply report=%#v err=%v", report, err)
	}
	if report.Status != StatusNotAttempted || len(report.Targets) != 1 {
		t.Fatalf("preflight report = %#v", report)
	}
	if result := report.Targets[0]; result.Status != StatusNotAttempted || result.Error == "" {
		t.Fatalf("source result = %#v", result)
	}
	assertAbsentInstallArtifact(t, config)
	assertAbsentInstallArtifact(t, installfs.BackupPath(config, plan.PlanID))
	assertAbsentInstallArtifact(t, installfs.JournalPath(config, plan.PlanID))
	assertAbsentInstallArtifact(t, config+".profile-mango.lock")
}

func TestApplyPreflightReportOnStaleTarget(t *testing.T) {
	registry := NewRegistry(testAdapterFor("fake", "1", "new", true))
	request, root := testRequest(t, registry)
	config := filepath.Join(root, "config")
	request.Override = true
	writeInstallTestFile(t, config, "old")
	request.Targets[0] = TargetRequest{Target: Target{Name: "fake", Version: "1"}, ConfigPath: config}
	plan, err := BuildPlan(request)
	if err != nil {
		t.Fatal(err)
	}
	writeInstallTestFile(t, config, "stale target")

	report, err := ApplyPlan(plan, ApplyOptions{ExpectedPlanID: plan.PlanID})
	if err == nil || !strings.Contains(err.Error(), "preflight") {
		t.Fatalf("apply report=%#v err=%v", report, err)
	}
	if report.Status != StatusNotAttempted || len(report.Targets) != 1 {
		t.Fatalf("preflight report = %#v", report)
	}
	if result := report.Targets[0]; result.Status != StatusNotAttempted || result.Error == "" {
		t.Fatalf("target result = %#v", result)
	}
	assertInstallTestFile(t, config, "stale target")
	assertAbsentInstallArtifact(t, installfs.BackupPath(config, plan.PlanID))
	assertAbsentInstallArtifact(t, installfs.JournalPath(config, plan.PlanID))
	assertAbsentInstallArtifact(t, config+".profile-mango.lock")
}

func TestApplyPreflightReportOnMissingConsentAndBlockedPlan(t *testing.T) {
	registry := NewRegistry(testAdapterFor("fake", "1", "new", true))
	request, root := testRequest(t, registry)
	request.Override = true
	config := filepath.Join(root, "config")
	request.Targets[0] = TargetRequest{Target: Target{Name: "fake", Version: "1"}, ConfigPath: config}
	plan, err := BuildPlan(request)
	if err != nil {
		t.Fatal(err)
	}

	report, err := ApplyPlan(plan, ApplyOptions{})
	if err == nil || !strings.Contains(err.Error(), "requires an expected plan ID") {
		t.Fatalf("missing-consent report=%#v err=%v", report, err)
	}
	if report.Status != StatusNotAttempted || report.Targets[0].Status != StatusNotAttempted {
		t.Fatalf("missing-consent report = %#v", report)
	}

	blocked := plan
	blocked.Status = StatusBlocked
	blocked.Targets = append([]TargetPlan(nil), plan.Targets...)
	blocked.Targets[0].Status = StatusBlocked
	blocked.Targets[0].changes = nil
	blocked.Targets[0].checks = nil
	report, err = ApplyPlan(blocked, ApplyOptions{ExpectedPlanID: blocked.PlanID})
	if err == nil || !strings.Contains(err.Error(), "not applicable") {
		t.Fatalf("blocked report=%#v err=%v", report, err)
	}
	if report.Status != StatusNotAttempted || report.Targets[0].Status != StatusBlocked {
		t.Fatalf("blocked report = %#v", report)
	}
	assertAbsentInstallArtifact(t, config)
	assertAbsentInstallArtifact(t, installfs.BackupPath(config, plan.PlanID))
	assertAbsentInstallArtifact(t, installfs.JournalPath(config, plan.PlanID))
	assertAbsentInstallArtifact(t, config+".profile-mango.lock")
}

func assertAbsentInstallArtifact(t *testing.T, path string) {
	t.Helper()
	if _, err := os.Lstat(path); !os.IsNotExist(err) {
		t.Fatalf("unexpected install artifact %s: %v", path, err)
	}
}
