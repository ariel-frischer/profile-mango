package install

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gitlab.com/ariel-frischer/profile-mango/internal/installfs"
)

type testAdapter struct {
	metadata AdapterMetadata
	content  string
	allow    bool
}

func (adapter testAdapter) Metadata() AdapterMetadata { return adapter.metadata }

func (adapter testAdapter) Plan(input AdapterInput) (Patch, error) {
	return Patch{
		Files:           []FilePatch{{Content: []byte(adapter.content), Fields: []string{"config.route"}}},
		Fields:          []FieldChange{{Path: "config.route", Before: "old", After: adapter.content}},
		OverrideAllowed: adapter.allow,
	}, nil
}

func TestDefaultRegistryBlocksProductionWithoutReadingTargetPath(t *testing.T) {
	request, root := testRequest(t, nil)
	request.Registry = DefaultRegistry()
	request.Targets[0].Target = Target{Name: "codex", Version: "0.154.0"}
	request.Targets[0].ConfigPath = filepath.Join(root, "does-not-exist")
	plan, err := BuildPlan(request)
	if err != nil {
		t.Fatal(err)
	}
	if plan.Status != StatusBlocked || plan.Targets[0].Status != StatusBlocked {
		t.Fatalf("plan = %#v", plan)
	}
	if _, err := os.Stat(request.Targets[0].ConfigPath); !os.IsNotExist(err) {
		t.Fatalf("blocked plan touched target path: %v", err)
	}
}

func TestPlanIDsAreStableAcrossTargetInputOrder(t *testing.T) {
	registry := NewRegistry(
		testAdapterFor("alpha", "1", "alpha", false),
		testAdapterFor("beta", "1", "beta", false),
	)
	first, root := testRequest(t, registry)
	first.Targets = []TargetRequest{
		{Target: Target{Name: "beta", Version: "1"}, ConfigPath: filepath.Join(root, "beta")},
		{Target: Target{Name: "alpha", Version: "1"}, ConfigPath: filepath.Join(root, "alpha")},
	}
	second := first
	second.Targets = append([]TargetRequest(nil), first.Targets[1], first.Targets[0])
	firstPlan, err := BuildPlan(first)
	if err != nil {
		t.Fatal(err)
	}
	secondPlan, err := BuildPlan(second)
	if err != nil {
		t.Fatal(err)
	}
	if firstPlan.PlanID != secondPlan.PlanID {
		t.Fatalf("plan IDs differ: %s != %s", firstPlan.PlanID, secondPlan.PlanID)
	}
	if firstPlan.Targets[0].Target.Name != "alpha" {
		t.Fatalf("targets were not sorted: %#v", firstPlan.Targets)
	}
}

func TestApplyReapplyAndStaleConsent(t *testing.T) {
	registry := NewRegistry(testAdapterFor("fake", "1", "new", true))
	request, root := testRequest(t, registry)
	request.Override = true
	config := filepath.Join(root, "config")
	writeInstallTestFile(t, config, "old")
	request.Targets[0] = TargetRequest{Target: Target{Name: "fake", Version: "1"}, ConfigPath: config}
	writeOwnedManifest(t, config, Target{Name: "fake", Version: "1"}, installfs.Hash([]byte("old")))
	plan, err := BuildPlan(request)
	if err != nil {
		t.Fatal(err)
	}
	if plan.Status != StatusReady {
		t.Fatalf("plan status = %s, diagnostics = %#v", plan.Status, plan.Diagnostics)
	}
	if _, err := ApplyPlan(plan, ApplyOptions{ExpectedPlanID: plan.PlanID}); err != nil {
		t.Fatalf("apply: %v, checks=%#v, changes=%#v", err, plan.Targets[0].checks, plan.Targets[0].changes)
	}
	assertInstallTestFile(t, config, "new")
	if _, err := os.Stat(installfs.BackupPath(config, plan.PlanID)); err != nil {
		t.Fatalf("backup missing: %v", err)
	}
	reapply, err := BuildPlan(request)
	if err != nil {
		t.Fatal(err)
	}
	if reapply.Targets[0].Files[0].Action != ActionNoop {
		t.Fatalf("reapply action = %s", reapply.Targets[0].Files[0].Action)
	}
	stale, err := BuildPlan(request)
	if err != nil {
		t.Fatal(err)
	}
	writeInstallTestFile(t, config, "third-party")
	if _, err := ApplyPlan(stale, ApplyOptions{ExpectedPlanID: stale.PlanID}); err == nil {
		t.Fatal("stale plan applied")
	}
	assertInstallTestFile(t, config, "third-party")
}

func TestConflictRequiresNarrowAdapterOverride(t *testing.T) {
	registry := NewRegistry(testAdapterFor("fake", "1", "new", false))
	request, root := testRequest(t, registry)
	config := filepath.Join(root, "config")
	writeInstallTestFile(t, config, "old")
	request.Targets[0] = TargetRequest{Target: Target{Name: "fake", Version: "1"}, ConfigPath: config}
	plan, err := BuildPlan(request)
	if err != nil {
		t.Fatal(err)
	}
	writeInstallTestFile(t, config, "edited")
	conflict, err := BuildPlan(request)
	if err != nil {
		t.Fatal(err)
	}
	if conflict.Status != StatusBlocked || conflict.Targets[0].Status != StatusConflict {
		t.Fatalf("conflict plan = %#v", conflict)
	}
	if _, err := ApplyPlan(plan, ApplyOptions{ExpectedPlanID: plan.PlanID}); err == nil {
		t.Fatal("conflicting stale plan applied")
	}
	overrideRequest := request
	overrideRequest.Override = true
	overrideRequest.Registry = NewRegistry(testAdapterFor("fake", "1", "new", true))
	override, err := BuildPlan(overrideRequest)
	if err != nil {
		t.Fatal(err)
	}
	if override.Targets[0].Files[0].Action != ActionOverride {
		t.Fatalf("override action = %s", override.Targets[0].Files[0].Action)
	}
}

func TestPlanJSONOmitsSyntheticAbsolutePaths(t *testing.T) {
	request, root := testRequest(t, NewRegistry(testAdapterFor("fake", "1", "new", false)))
	request.Targets[0] = TargetRequest{Target: Target{Name: "fake", Version: "1"}, ConfigPath: filepath.Join(root, "config")}
	plan, err := BuildPlan(request)
	if err != nil {
		t.Fatal(err)
	}
	data, err := plan.JSON()
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(data), root) {
		t.Fatalf("plan leaked synthetic absolute path: %s", data)
	}
}

func TestOpenCodeInstallPreservesUnrelatedStateAndReapplies(t *testing.T) {
	request, root := openCodeTestRequest(t)
	config := request.Targets[0].ConfigPath
	before := "{\n  // keep target-owned state\n  \"model\" : \"sentinel/old\",\n  \"provider\": {\"sentinel\": {\"options\": {\"apiKey\": \"SYNTHETIC\"}}},\n  \"unknown\": true,\n}\n"
	writeInstallTestFile(t, config, before)
	if err := os.Chmod(config, 0o600); err != nil {
		t.Fatal(err)
	}

	plan, err := BuildPlan(request)
	if err != nil {
		t.Fatal(err)
	}
	if plan.Status != StatusReady || plan.Targets[0].Files[0].Action != ActionOverride {
		t.Fatalf("plan = %#v", plan)
	}
	if plan.Targets[0].DestinationSHA256 == "" || len(plan.Targets[0].Fields) != 1 || plan.Targets[0].Fields[0].Path != "config.model" {
		t.Fatalf("target plan = %#v", plan.Targets[0])
	}
	data, err := plan.JSON()
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(data), root) || !strings.Contains(string(data), `"destinationSHA256"`) {
		t.Fatalf("public plan destination boundary = %s", data)
	}

	if _, err := ApplyPlan(plan, ApplyOptions{ExpectedPlanID: plan.PlanID}); err != nil {
		t.Fatal(err)
	}
	want := strings.Replace(before, `"sentinel/old"`, `"openai/gpt-5.6"`, 1)
	assertInstallTestFile(t, config, want)
	assertInstallTestFile(t, installfs.BackupPath(config, plan.PlanID), before)
	info, err := os.Stat(config)
	if err != nil || info.Mode().Perm() != 0o600 {
		t.Fatalf("config mode = %v, err = %v", info.Mode().Perm(), err)
	}
	journalData, err := os.ReadFile(installfs.JournalPath(config, plan.PlanID))
	if err != nil || !strings.Contains(string(journalData), `"status": "committed"`) {
		t.Fatalf("journal = %s, err = %v", journalData, err)
	}
	manifestData, err := os.ReadFile(config + ".profile-mango.manifest.json")
	if err != nil {
		t.Fatal(err)
	}
	manifest, err := decodeManifest(manifestData, Target{Name: "opencode", Version: "1.18.31"})
	if err != nil || len(manifest.Files) != 1 || strings.Join(manifest.Files[0].Fields, ",") != "config.model" {
		t.Fatalf("manifest = %#v, err = %v", manifest, err)
	}

	reapply, err := BuildPlan(request)
	if err != nil {
		t.Fatal(err)
	}
	if reapply.Status != StatusNoop || reapply.Targets[0].Files[0].Action != ActionNoop {
		t.Fatalf("reapply = %#v", reapply)
	}
}

func TestOpenCodePlanBindsDestinationAndRejectsStaleApply(t *testing.T) {
	request, root := openCodeTestRequest(t)
	first := request.Targets[0].ConfigPath
	writeInstallTestFile(t, first, `{ "model": "old/model" }`)
	firstPlan, err := BuildPlan(request)
	if err != nil {
		t.Fatal(err)
	}

	second := filepath.Join(root, "other", "opencode.jsonc")
	if err := os.MkdirAll(filepath.Dir(second), 0o755); err != nil {
		t.Fatal(err)
	}
	writeInstallTestFile(t, second, `{ "model": "old/model" }`)
	secondRequest := request
	secondRequest.Targets = []TargetRequest{{Target: request.Targets[0].Target, ConfigPath: second}}
	secondPlan, err := BuildPlan(secondRequest)
	if err != nil {
		t.Fatal(err)
	}
	if firstPlan.PlanID == secondPlan.PlanID || firstPlan.Targets[0].DestinationSHA256 == secondPlan.Targets[0].DestinationSHA256 {
		t.Fatal("plan consent was not bound to the explicit destination")
	}

	writeInstallTestFile(t, first, `{ "model": "third-party/edit" }`)
	if _, err := ApplyPlan(firstPlan, ApplyOptions{ExpectedPlanID: firstPlan.PlanID}); !errors.Is(err, installfs.ErrStale) {
		t.Fatalf("stale apply error = %v", err)
	}
	assertInstallTestFile(t, first, `{ "model": "third-party/edit" }`)
}

func TestOpenCodeInstallBlocksUnverifiedProfileRequirements(t *testing.T) {
	request, root := openCodeTestRequest(t)
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
		t.Fatalf("plan = %#v", plan)
	}
}

func openCodeTestRequest(t *testing.T) (Request, string) {
	t.Helper()
	request, root := testRequest(t, DefaultRegistry())
	bindings := `routes:
  primary:
    provider: openai
    transport: native
    authentication: oauth
    model: gpt-5.6
    effort: high
`
	writeInstallTestFile(t, request.BindingsPath, bindings)
	config := filepath.Join(root, "target", "opencode.jsonc")
	if err := os.MkdirAll(filepath.Dir(config), 0o755); err != nil {
		t.Fatal(err)
	}
	request.Override = true
	request.Targets = []TargetRequest{{Target: Target{Name: "opencode", Version: "1.18.31"}, ConfigPath: config}}
	return request, root
}

func testAdapterFor(name, version, content string, allow bool) testAdapter {
	return testAdapter{metadata: AdapterMetadata{
		Target: name, Version: version, AdapterVersion: "test/v1", Installable: true,
	}, content: content, allow: allow}
}

func testRequest(t *testing.T, registry *Registry) (Request, string) {
	t.Helper()
	root := t.TempDir()
	profiles := filepath.Join(root, "profiles", "route-only")
	if err := os.MkdirAll(profiles, 0o755); err != nil {
		t.Fatal(err)
	}
	writeInstallTestFile(t, filepath.Join(profiles, "profile.yaml"), `apiVersion: profilemango.dev/v1alpha1
kind: PolicyProfile
metadata:
  name: route-only
spec:
  routeRef: primary
`)
	if err := os.MkdirAll(filepath.Join(root, "bindings"), 0o755); err != nil {
		t.Fatal(err)
	}
	bindings := `routes:
  primary:
    provider: test
    transport: local
    authentication: none
    model: test-model
    effort: low
`
	bindingsPath := filepath.Join(root, "bindings", "local.yaml")
	writeInstallTestFile(t, bindingsPath, bindings)
	return Request{
		ProfileName:  "route-only",
		ProfilesRoot: filepath.Join(root, "profiles"),
		ResourceRoot: root,
		BindingsPath: bindingsPath,
		Registry:     registry,
		Backup:       true,
		Targets:      []TargetRequest{{Target: Target{Name: "fake", Version: "1"}, ConfigPath: filepath.Join(root, "config")}},
	}, root
}

func writeInstallTestFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func assertInstallTestFile(t *testing.T, path, want string) {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != want {
		t.Fatalf("%s = %q, want %q", path, data, want)
	}
}

func writeOwnedManifest(t *testing.T, config string, target Target, hash string) {
	t.Helper()
	manifest := Manifest{
		APIVersion: ManifestAPIVersion,
		Kind:       ManifestKind,
		Owner:      "profile-mango",
		Generation: 1,
		Profile:    "route-only",
		Target:     target,
		Files:      []ManifestFile{{Path: config, SHA256: hash}},
	}
	data, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	writeInstallTestFile(t, config+".profile-mango.manifest.json", string(data)+"\n")
}
