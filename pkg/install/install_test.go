package install

import (
	"encoding/json"
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
