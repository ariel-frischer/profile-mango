package install

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gitlab.com/ariel-frischer/profile-mango/internal/installfs"
	"gitlab.com/ariel-frischer/profile-mango/pkg/profilemango"
)

func TestPiInstallPreservesStateAndUsesOwnership(t *testing.T) {
	request, root := piTestRequest(t)
	config := filepath.Join(root, "target", "settings.json")
	if err := os.MkdirAll(filepath.Dir(config), 0o755); err != nil {
		t.Fatal(err)
	}
	before := "{\r\n  \"defaultProvider\" : \"old/provider\",\r\n  \"defaultModel\":\"old-model\",\r\n  \"defaultThinkingLevel\": \"low\",\r\n  \"unknown\": {\"keep\": true},\r\n  \"provider\": {\"sentinel\": {\"options\": {\"apiKey\": \"SYNTHETIC\"}}}\r\n}\r\n"
	writeInstallTestFile(t, config, before)
	if err := os.Chmod(config, 0o600); err != nil {
		t.Fatal(err)
	}
	request.Targets[0].ConfigPath = config
	request.Override = true
	first, err := BuildPlan(request)
	if err != nil {
		t.Fatal(err)
	}
	second, err := BuildPlan(request)
	if err != nil {
		t.Fatal(err)
	}
	if first.PlanID != second.PlanID || first.Status != StatusReady {
		t.Fatalf("plans are not deterministic and ready: first=%#v second=%#v", first, second)
	}
	if first.Targets[0].Files[0].Action != ActionOverride || len(first.Targets[0].Fields) != 3 {
		t.Fatalf("Pi plan = %#v", first.Targets[0])
	}
	public, err := first.JSON()
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(public), root) || !strings.Contains(string(public), "destinationSHA256") {
		t.Fatalf("public Pi plan leaked path or destination digest: %s", public)
	}
	if _, err := ApplyPlan(first, ApplyOptions{ExpectedPlanID: first.PlanID}); err != nil {
		t.Fatal(err)
	}
	want := strings.Replace(before, `"old/provider"`, `"openai"`, 1)
	want = strings.Replace(want, `"old-model"`, `"gpt-5.6"`, 1)
	want = strings.Replace(want, `"low"`, `"high"`, 1)
	assertInstallTestFile(t, config, want)
	assertInstallTestFile(t, installfs.BackupPath(config, first.PlanID), before)
	info, err := os.Stat(config)
	if err != nil || info.Mode().Perm() != 0o600 {
		t.Fatalf("Pi config mode = %v, err = %v", info.Mode().Perm(), err)
	}
	reapply, err := BuildPlan(request)
	if err != nil {
		t.Fatal(err)
	}
	if reapply.Status != StatusNoop || reapply.Targets[0].Files[0].Action != ActionNoop {
		t.Fatalf("Pi reapply = %#v", reapply)
	}
}

func TestPiInstallConflictAndStaleApply(t *testing.T) {
	request, root := piTestRequest(t)
	config := filepath.Join(root, "target", "settings.json")
	if err := os.MkdirAll(filepath.Dir(config), 0o755); err != nil {
		t.Fatal(err)
	}
	writeInstallTestFile(t, config, `{ "defaultProvider": "third-party", "defaultModel": "third-party", "defaultThinkingLevel": "low" }`)
	request.Targets[0].ConfigPath = config
	adopt, err := BuildPlan(request)
	if err != nil {
		t.Fatal(err)
	}
	if adopt.Status != StatusReady || adopt.Targets[0].Files[0].Action != ActionAdopt {
		t.Fatalf("unowned Pi config was not adopted: %#v", adopt)
	}
	request.Override = true
	initial, err := BuildPlan(request)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := ApplyPlan(initial, ApplyOptions{ExpectedPlanID: initial.PlanID}); err != nil {
		t.Fatal(err)
	}
	writeInstallTestFile(t, request.BindingsPath, piBindings("gpt-5.7"))
	request.Override = false
	update, err := BuildPlan(request)
	if err != nil {
		t.Fatal(err)
	}
	if update.Status != StatusReady || update.Targets[0].Files[0].Action != ActionUpdate {
		t.Fatalf("owned Pi update was not ready: %#v", update)
	}
	writeInstallTestFile(t, config, `{ "defaultProvider": "third-party", "defaultModel": "third-party", "defaultThinkingLevel": "low" }`)
	if _, err := ApplyPlan(update, ApplyOptions{ExpectedPlanID: update.PlanID}); !errors.Is(err, installfs.ErrStale) {
		t.Fatalf("stale Pi apply error = %v", err)
	}
	assertInstallTestFile(t, config, `{ "defaultProvider": "third-party", "defaultModel": "third-party", "defaultThinkingLevel": "low" }`)
}

func TestPiInstallBlocksMalformedSettingsAndUnverifiedRequirements(t *testing.T) {
	request, root := piTestRequest(t)
	config := filepath.Join(root, "target", "settings.json")
	if err := os.MkdirAll(filepath.Dir(config), 0o755); err != nil {
		t.Fatal(err)
	}
	writeInstallTestFile(t, config, `{ malformed`)
	request.Targets[0].ConfigPath = config
	plan, err := BuildPlan(request)
	if err != nil {
		t.Fatal(err)
	}
	if plan.Status != StatusBlocked || !strings.Contains(plan.Targets[0].Reason, "adapter planning failed") {
		t.Fatalf("malformed Pi settings were not blocked: %#v", plan)
	}
	input := AdapterInput{Target: Target{Name: "pi", Version: "0.86.1"}, Profile: requestProfile(), Route: piTestRoute()}
	input.Profile.Instructions = []string{"instructions/system.md"}
	if _, err := (piAdapter{}).Plan(input); err == nil || !strings.Contains(err.Error(), "delivery") {
		t.Fatalf("Pi resource delivery was not blocked: %v", err)
	}
}

func TestPiInstallPreservesStateBacksUpAndReapplies(t *testing.T) {
	request, root := piInstallTestRequest(t)
	config := request.Targets[0].ConfigPath
	before := "{\n  \"defaultProvider\" : \"old/provider\",\n  \"defaultModel\":\"old-model\",\n  \"defaultThinkingLevel\": \"low\",\n  \"unknown\": {\"keep\": true},\n  \"provider\": {\"sentinel\": {\"options\": {\"apiKey\": \"SYNTHETIC\"}}}\n}\n"
	writeInstallTestFile(t, config, before)
	if err := os.Chmod(config, 0o600); err != nil {
		t.Fatal(err)
	}
	writeOwnedManifest(t, config, request.Targets[0].Target, installfs.Hash([]byte(before)))

	plan, err := BuildPlan(request)
	if err != nil {
		t.Fatal(err)
	}
	if plan.Status != StatusReady || piFilePlan(plan.Targets[0], "settings.json").Action != ActionUpdate {
		t.Fatalf("plan = %#v", plan)
	}
	if len(plan.Targets[0].Fields) != 3 || !strings.Contains(diagnosticText(plan.Targets[0]), "three top-level route defaults") {
		t.Fatalf("Pi plan fields/diagnostics = %#v", plan.Targets[0])
	}
	data, err := plan.JSON()
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(data), root) || !strings.Contains(string(data), "defaultThinkingLevel") {
		t.Fatalf("Pi plan leaked path or fields: %s", data)
	}

	if _, err := ApplyPlan(plan, ApplyOptions{ExpectedPlanID: plan.PlanID}); err != nil {
		t.Fatal(err)
	}
	want := strings.Replace(before, `"old/provider"`, `"openai"`, 1)
	want = strings.Replace(want, `"old-model"`, `"gpt-5.6"`, 1)
	want = strings.Replace(want, `"low"`, `"high"`, 1)
	assertInstallTestFile(t, config, want)
	assertInstallTestFile(t, installfs.BackupPath(config, plan.PlanID), before)
	info, err := os.Stat(config)
	if err != nil || info.Mode().Perm() != 0o600 {
		t.Fatalf("Pi config mode = %v, err = %v", info.Mode().Perm(), err)
	}

	reapply, err := BuildPlan(request)
	if err != nil {
		t.Fatal(err)
	}
	if reapply.Status != StatusNoop || piFilePlan(reapply.Targets[0], "settings.json").Action != ActionNoop {
		t.Fatalf("Pi reapply = %#v", reapply)
	}
}

func TestPiInstallRejectsStalePlanWithoutWrites(t *testing.T) {
	request, _ := piInstallTestRequest(t)
	config := request.Targets[0].ConfigPath
	before := "{\"defaultProvider\":\"old\",\"defaultModel\":\"old\",\"defaultThinkingLevel\":\"low\"}\n"
	writeInstallTestFile(t, config, before)
	writeOwnedManifest(t, config, request.Targets[0].Target, installfs.Hash([]byte(before)))
	plan, err := BuildPlan(request)
	if err != nil {
		t.Fatal(err)
	}
	stale := "{\"defaultProvider\":\"third-party\",\"defaultModel\":\"old\",\"defaultThinkingLevel\":\"low\"}\n"
	writeInstallTestFile(t, config, stale)
	if _, err := ApplyPlan(plan, ApplyOptions{ExpectedPlanID: plan.PlanID}); !errors.Is(err, installfs.ErrStale) {
		t.Fatalf("Pi stale apply error = %v", err)
	}
	assertInstallTestFile(t, config, stale)
}

func TestPiInstallAdoptsUnownedConfigOrOverrides(t *testing.T) {
	request, _ := piInstallTestRequest(t)
	config := request.Targets[0].ConfigPath
	writeInstallTestFile(t, config, "{\"defaultProvider\":\"old\",\"defaultModel\":\"old\",\"defaultThinkingLevel\":\"low\"}\n")
	plan, err := BuildPlan(request)
	if err != nil {
		t.Fatal(err)
	}
	if plan.Status != StatusReady || piFilePlan(plan.Targets[0], "settings.json").Action != ActionAdopt {
		t.Fatalf("unowned Pi config was not adopted: %#v", plan)
	}
	request.Override = true
	override, err := BuildPlan(request)
	if err != nil {
		t.Fatal(err)
	}
	if override.Status != StatusReady || piFilePlan(override.Targets[0], "settings.json").Action != ActionOverride {
		t.Fatalf("Pi override was not narrow and ready: %#v", override)
	}
}

func TestPiStrictInstallBlocksUnverifiedRequirements(t *testing.T) {
	request, root := piInstallTestRequest(t)
	request.Strict = true
	profile := filepath.Join(root, "profiles", "route-only", "profile.yaml")
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
	if err != nil {
		t.Fatal(err)
	}
	if plan.Status != StatusBlocked || !strings.Contains(plan.Targets[0].Reason, "permission requirements") {
		t.Fatalf("Pi blocked plan = %#v", plan)
	}
}

func piInstallTestRequest(t *testing.T) (Request, string) {
	t.Helper()
	request, root := testRequest(t, NewRegistry(piAdapter{}))
	writeInstallTestFile(t, request.BindingsPath, `routes:
  primary:
    provider: openai
    transport: native
    authentication: oauth
    model: gpt-5.6
    effort: high
`)
	config := filepath.Join(root, "target", "pi", "settings.json")
	if err := os.MkdirAll(filepath.Dir(config), 0o755); err != nil {
		t.Fatal(err)
	}
	request.Targets = []TargetRequest{{Target: Target{Name: "pi", Version: "0.86.1"}, ConfigPath: config}}
	return request, root
}

func piFilePlan(target TargetPlan, name string) FilePlan {
	for _, file := range target.Files {
		if file.Path == name {
			return file
		}
	}
	return FilePlan{}
}

func diagnosticText(target TargetPlan) string {
	var text strings.Builder
	for _, diagnostic := range target.Diagnostics {
		text.WriteString(diagnostic.Message)
		text.WriteByte('\n')
	}
	return text.String()
}

func piTestRequest(t *testing.T) (Request, string) {
	t.Helper()
	request, root := testRequest(t, NewRegistry(piAdapter{}))
	writeInstallTestFile(t, request.BindingsPath, piBindings("gpt-5.6"))
	request.Targets[0].Target = Target{Name: "pi", Version: "0.86.1"}
	return request, root
}

func piBindings(model string) string {
	return "routes:\n  primary:\n    provider: openai\n    transport: native\n    authentication: oauth\n    model: " + model + "\n    effort: high\n"
}

func piTestRoute() profilemango.RouteBinding {
	return profilemango.RouteBinding{Provider: "openai", Transport: "native", Authentication: "oauth", Model: "gpt-5.6", Effort: "high"}
}

func requestProfile() profilemango.ResolvedProfile {
	return profilemango.ResolvedProfile{Metadata: profilemango.Metadata{Name: "route-only"}}
}
