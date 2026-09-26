package install

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/ariel-frischer/profile-mango/internal/installfs"
)

func TestLaterTargetBackupFailurePreservesEveryConfig(t *testing.T) {
	registry := NewRegistry(testAdapterFor("a", "1", "new-a", true), testAdapterFor("b", "1", "new-b", true))
	request, root := testRequest(t, registry)
	first, second := filepath.Join(root, "a.conf"), filepath.Join(root, "b.conf")
	writeInstallTestFile(t, first, "old-a")
	writeInstallTestFile(t, second, "old-b")
	request.Override = true
	request.Targets = []TargetRequest{
		{Target: Target{Name: "a", Version: "1"}, ConfigPath: first},
		{Target: Target{Name: "b", Version: "1"}, ConfigPath: second},
	}
	plan, err := BuildPlan(request)
	if err != nil || plan.Status != StatusReady {
		t.Fatalf("plan status=%s err=%v", plan.Status, err)
	}
	backup := installfs.BackupPath(second, plan.PlanID)
	writeInstallTestFile(t, backup, "independent-backup")
	report, err := ApplyPlan(plan, ApplyOptions{ExpectedPlanID: plan.PlanID})
	if err == nil || !strings.Contains(err.Error(), "backup already exists") {
		t.Fatalf("apply report=%#v err=%v", report, err)
	}
	assertInstallTestFile(t, first, "old-a")
	assertInstallTestFile(t, second, "old-b")
	assertInstallTestFile(t, backup, "independent-backup")
	if len(report.Targets) != 2 || report.Status != "failed" {
		t.Fatalf("incomplete failure report: %#v", report)
	}
	for _, target := range report.Targets {
		if target.Status != "failed" || target.Error == "" {
			t.Fatalf("misleading target outcome: %#v", target)
		}
	}
}
