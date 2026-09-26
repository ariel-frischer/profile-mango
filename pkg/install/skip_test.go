package install

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// homeAdapter gives a test adapter a documented default config path beneath a synthetic home.
type homeAdapter struct {
	testAdapter
	folder string
}

func (adapter homeAdapter) DefaultConfigPath(env PathEnv) (string, error) {
	home, err := env.home()
	return filepath.Join(home, adapter.folder, "config"), err
}

// skipTestRequest registers "present" (folder exists) and "absent" (no folder) with default paths only.
func skipTestRequest(t *testing.T) (Request, string) {
	t.Helper()
	registry := NewRegistry(
		homeAdapter{testAdapter: testAdapterFor("present", "1", "present\n", false), folder: ".present"},
		homeAdapter{testAdapter: testAdapterFor("absent", "1", "absent\n", false), folder: ".absent"},
	)
	request, root := testRequest(t, registry)
	home := filepath.Join(root, "home")
	if err := os.MkdirAll(filepath.Join(home, ".present"), 0o755); err != nil {
		t.Fatal(err)
	}
	request.Env = syntheticPathEnv(t, home, nil)
	request.Targets = []TargetRequest{{Target: Target{Name: "present", Version: "1"}}, {Target: Target{Name: "absent", Version: "1"}}}
	request.SkipNotInstalled = true
	request.DetectVersion = fakeDetector("", false)
	return request, home
}

func targetByName(t *testing.T, plan Plan, name string) TargetPlan {
	t.Helper()
	for _, target := range plan.Targets {
		if target.Target.Name == name {
			return target
		}
	}
	t.Fatalf("plan has no %s target: %#v", name, plan.Targets)
	return TargetPlan{}
}

func TestSkipNotInstalledPlansInstalledAgentsOnly(t *testing.T) {
	request, home := skipTestRequest(t)
	plan, err := BuildPlan(request)
	if err != nil {
		t.Fatal(err)
	}
	present, absent := targetByName(t, plan, "present"), targetByName(t, plan, "absent")
	if plan.Status != StatusReady || present.Status != StatusReady || absent.Status != StatusSkipped {
		t.Fatalf("statuses plan=%s present=%s absent=%s (%s)", plan.Status, present.Status, absent.Status, absent.Reason)
	}
	if absent.Reason != "config folder ~/.absent not found" || absent.Config == nil || absent.Config.Path != filepath.Join(home, ".absent", "config") {
		t.Fatalf("skipped target = %#v", absent)
	}
	data, err := plan.JSON()
	if err != nil || !strings.Contains(string(data), `"status": "skipped"`) || strings.Contains(string(data), home) {
		t.Fatalf("plan JSON = %s, err=%v", data, err)
	}
	again, err := BuildPlan(request)
	if err != nil || again.PlanID != plan.PlanID {
		t.Fatalf("plan ID not deterministic: %s != %s, err=%v", again.PlanID, plan.PlanID, err)
	}
	if _, err := ApplyPlan(plan, ApplyOptions{ExpectedPlanID: plan.PlanID}); err != nil {
		t.Fatalf("apply with a skipped target: %v", err)
	}
	if _, err := os.Stat(filepath.Join(home, ".absent")); !os.IsNotExist(err) {
		t.Fatalf("skipped target folder was created: %v", err)
	}
}

func TestSkipNotInstalledSkipsMissingFolderEvenWhenCommandIsOnPath(t *testing.T) {
	request, _ := skipTestRequest(t)
	request.DetectVersion = fakeDetector("absent 1.0.0", true)
	plan, err := BuildPlan(request)
	if err != nil {
		t.Fatal(err)
	}
	absent := targetByName(t, plan, "absent")
	if plan.Status != StatusReady || absent.Status != StatusSkipped || absent.Reason != "config folder ~/.absent not found" {
		t.Fatalf("--all must skip an agent without a config folder: plan=%s target=%#v", plan.Status, absent)
	}
}

func TestSkipNotInstalledBlocksWhenEveryTargetIsSkipped(t *testing.T) {
	request, home := skipTestRequest(t)
	if err := os.Remove(filepath.Join(home, ".present")); err != nil {
		t.Fatal(err)
	}
	plan, err := BuildPlan(request)
	if err != nil {
		t.Fatal(err)
	}
	if plan.Status != StatusBlocked || !hasDiagnostic(plan.Diagnostics, "install.no_agents_found") {
		t.Fatalf("all-skipped plan = %s %#v", plan.Status, plan.Diagnostics)
	}
	for _, diagnostic := range plan.Diagnostics {
		if diagnostic.Code == "install.no_agents_found" && !strings.Contains(diagnostic.Message, "mango doctor") {
			t.Fatalf("no-agents diagnostic lacks the doctor hint: %s", diagnostic.Message)
		}
	}
}

func TestExplicitTargetMissingFolderNamesFix(t *testing.T) {
	request, home := skipTestRequest(t)
	request.SkipNotInstalled = false
	request.Targets = request.Targets[1:]
	plan, err := BuildPlan(request)
	if err != nil {
		t.Fatal(err)
	}
	want := "absent config folder " + filepath.Join(home, ".absent") + " not found — is absent installed? Pass --config absent=<path> to choose a file."
	if plan.Status != StatusBlocked || plan.Targets[0].Reason != want || !hasDiagnostic(plan.Targets[0].Diagnostics, "install.config_folder_missing") {
		t.Fatalf("explicit missing folder = %#v", plan.Targets[0])
	}
}
