package install

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/ariel-frischer/profile-mango/internal/installfs"
)

type mixedPreflightFixture struct {
	request     Request
	plan        Plan
	root, alpha string
	zeta        string
}

type mixedPreflightCase struct {
	allNoop, wrongConsent bool
	mutation              string
	noopError             bool
	cause                 string
}

func TestApplyPreflightReportMixedRejections(t *testing.T) {
	tests := map[string]mixedPreflightCase{
		"stale-ready-target":  {mutation: "stale-ready", cause: "preflight"},
		"stale-noop-target":   {mutation: "stale-noop", noopError: true, cause: "preflight"},
		"stale-source":        {mutation: "stale-source", cause: "source"},
		"wrong-consent-mixed": {wrongConsent: true, cause: "does not match"},
		"all-noop-consent":    {allNoop: true, wrongConsent: true, cause: "does not match"},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			fixture := newMixedPreflightFixture(t, test.allNoop)
			mutateMixedPreflightFixture(t, &fixture, test.mutation)
			before := snapshotInstallTree(t, fixture.root)
			expected := fixture.plan.PlanID
			if test.wrongConsent {
				expected = "wrong-consent"
			}
			report, err := ApplyPlan(fixture.plan, ApplyOptions{ExpectedPlanID: expected})
			if err == nil || !strings.Contains(err.Error(), test.cause) {
				t.Fatalf("report=%#v err=%v", report, err)
			}
			assertMixedPreflightReport(t, report, test.allNoop, test.noopError, test.cause)
			assertInstallTreeUnchanged(t, fixture.root, before)
		})
	}
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
	before := snapshotInstallTree(t, root)

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
	assertInstallTreeUnchanged(t, root, before)
}

func TestApplyPreflightReportOnEmptyPlan(t *testing.T) {
	report, err := ApplyPlan(Plan{}, ApplyOptions{})
	if err == nil || !strings.Contains(err.Error(), "requires an expected plan ID") {
		t.Fatalf("report=%#v err=%v", report, err)
	}
	if report.Status != StatusNotAttempted || len(report.Targets) != 1 {
		t.Fatalf("empty plan report = %#v", report)
	}
	if result := report.Targets[0]; result.Target != "install" || result.Status != StatusNotAttempted || result.Error == "" {
		t.Fatalf("empty plan result = %#v", result)
	}
}

func newMixedPreflightFixture(t *testing.T, allNoop bool) mixedPreflightFixture {
	registry := NewRegistry(
		testAdapterFor("alpha", "1", "alpha-new", true),
		testAdapterFor("zeta", "1", "zeta-new", true),
	)
	request, root := testRequest(t, registry)
	request.Override = true
	fixture := mixedPreflightFixture{request: request, root: root, alpha: filepath.Join(root, "alpha"), zeta: filepath.Join(root, "zeta")}
	writeInstallTestFile(t, fixture.alpha, "alpha-new")
	writeInstallTestFile(t, fixture.zeta, "zeta-old")
	writeOwnedManifest(t, fixture.zeta, Target{Name: "zeta", Version: "1"}, installfs.Hash([]byte("zeta-old")))
	request.Targets = []TargetRequest{{Target: Target{Name: "alpha", Version: "1"}, ConfigPath: fixture.alpha}}
	seed, err := BuildPlan(request)
	if err != nil {
		t.Fatal(err)
	}
	applyFixturePlan(t, seed)
	request.Targets = []TargetRequest{
		{Target: Target{Name: "zeta", Version: "1"}, ConfigPath: fixture.zeta},
		{Target: Target{Name: "alpha", Version: "1"}, ConfigPath: fixture.alpha},
	}
	fixture.plan = buildFixturePlan(t, request)
	if allNoop {
		applyFixturePlan(t, fixture.plan)
		fixture.plan = buildFixturePlan(t, request)
	}
	fixture.request = request
	return fixture
}

func buildFixturePlan(t *testing.T, request Request) Plan {
	t.Helper()
	plan, err := BuildPlan(request)
	if err != nil {
		t.Fatal(err)
	}
	return plan
}

func applyFixturePlan(t *testing.T, plan Plan) {
	t.Helper()
	if _, err := ApplyPlan(plan, ApplyOptions{ExpectedPlanID: plan.PlanID}); err != nil {
		t.Fatal(err)
	}
}

func mutateMixedPreflightFixture(t *testing.T, fixture *mixedPreflightFixture, mutation string) {
	t.Helper()
	switch mutation {
	case "stale-ready":
		writeInstallTestFile(t, fixture.zeta, "stale-ready")
	case "stale-noop":
		writeInstallTestFile(t, fixture.alpha, "stale-noop")
	case "stale-source":
		profile := filepath.Join(fixture.request.ProfilesRoot, "route-only", "profile.yaml")
		writeInstallTestFile(t, profile, "changed source")
	}
}

func assertMixedPreflightReport(t *testing.T, report ApplyReport, allNoop, noopError bool, cause string) {
	t.Helper()
	if report.Status != StatusNotAttempted || len(report.Targets) != 2 {
		t.Fatalf("preflight report = %#v", report)
	}
	alpha, zeta := report.Targets[0], report.Targets[1]
	if alpha.Target != "alpha@1" || zeta.Target != "zeta@1" {
		t.Fatalf("report ordering = %#v", report.Targets)
	}
	if alpha.Status != StatusNoop {
		t.Fatalf("alpha result = %#v", alpha)
	}
	if allNoop {
		if zeta.Status != StatusNoop || !strings.Contains(alpha.Error, cause) || zeta.Error != "" {
			t.Fatalf("all-noop results = %#v", report.Targets)
		}
		return
	}
	if noopError != strings.Contains(alpha.Error, cause) {
		t.Fatalf("alpha error = %#v", alpha)
	}
	if zeta.Status != StatusNotAttempted || !strings.Contains(zeta.Error, cause) {
		t.Fatalf("zeta result = %#v", zeta)
	}
}

func snapshotInstallTree(t *testing.T, root string) map[string]string {
	t.Helper()
	files := make(map[string]string)
	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		files[path] = string(data)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return files
}

func assertInstallTreeUnchanged(t *testing.T, root string, before map[string]string) {
	t.Helper()
	after := snapshotInstallTree(t, root)
	if !reflect.DeepEqual(before, after) {
		t.Fatalf("synthetic install tree changed:\nbefore=%#v\nafter=%#v", before, after)
	}
}
