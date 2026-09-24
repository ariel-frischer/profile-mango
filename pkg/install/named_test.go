package install

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gitlab.com/ariel-frischer/profile-mango/internal/installfs"
)

const codexNamedBase = "# keep\nmodel = \"root-model\"\n[features]\napps = false\n[profiles.dev]\nmodel = \"legacy\"\n"

// codexNamedRequest plans the named profile into a synthetic Codex home next to config.toml.
func codexNamedRequest(t *testing.T, names ...string) (Request, string) {
	t.Helper()
	request, _ := codexTestRequest(t)
	request.Default = false
	for _, name := range names {
		if err := os.MkdirAll(filepath.Join(request.ProfilesRoot, name), 0o755); err != nil {
			t.Fatal(err)
		}
		writeInstallTestFile(t, filepath.Join(request.ProfilesRoot, name, "profile.yaml"), "apiVersion: profilemango.dev/v1alpha1\nkind: PolicyProfile\nmetadata:\n  name: "+name+"\nspec:\n  routeRef: primary\n")
	}
	config := request.Targets[0].ConfigPath
	writeInstallTestFile(t, config, codexNamedBase)
	return request, config
}

func applyNamed(t *testing.T, request Request, name string) Plan {
	t.Helper()
	request.ProfileName = name
	plan, err := BuildPlan(request)
	if err != nil || plan.Status != StatusReady {
		t.Fatalf("plan %s status=%s err=%v diagnostics=%v", name, plan.Status, err, plan.Diagnostics)
	}
	if _, err := ApplyPlan(plan, ApplyOptions{ExpectedPlanID: plan.PlanID}); err != nil {
		t.Fatal(err)
	}
	return plan
}

func TestCodexNamedProfilesInstallSideBySideAndKeepConfig(t *testing.T) {
	request, config := codexNamedRequest(t, "coding", "review")
	plan := applyNamed(t, request, "coding")
	want := InstallMode{Mode: InstallModeNamedProfile, ProfileName: "coding", UseCommand: "codex --profile coding"}
	if got := plan.Targets[0].Install; got == nil || *got != want {
		t.Fatalf("install mode = %#v, want %#v", got, want)
	}
	applyNamed(t, request, "review")
	assertInstallTestFile(t, config, codexNamedBase)
	dir := filepath.Dir(config)
	for _, name := range []string{"coding", "review"} {
		assertInstallTestFile(t, filepath.Join(dir, name+".config.toml"), "model_provider = \"openai\"\nmodel = \"gpt-5.6\"\nmodel_reasoning_effort = \"high\"\n")
	}
	for _, name := range []string{"review", "coding"} {
		request.ProfileName = name
		again, err := BuildPlan(request)
		if err != nil || again.Status != StatusNoop {
			t.Fatalf("reinstall %s status=%s err=%v", name, again.Status, err)
		}
	}
}

func TestCodexNamedPlanJSONReportsModeAndUseCommand(t *testing.T) {
	request, _ := codexNamedRequest(t, "coding")
	request.ProfileName = "coding"
	plan, err := BuildPlan(request)
	if err != nil {
		t.Fatal(err)
	}
	data, err := plan.JSON()
	if err != nil {
		t.Fatal(err)
	}
	var decoded struct {
		Targets []struct {
			Install map[string]any `json:"install"`
		} `json:"targets"`
	}
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatal(err)
	}
	install := decoded.Targets[0].Install
	if install["mode"] != "named-profile" || install["profileName"] != "coding" || install["useCommand"] != "codex --profile coding" {
		t.Fatalf("install JSON = %#v", install)
	}
}

func TestCodexDefaultFlagAlsoPatchesConfig(t *testing.T) {
	request, config := codexNamedRequest(t, "coding")
	request.Default = true
	plan := applyNamed(t, request, "coding")
	if !plan.Targets[0].Install.SetsDefault {
		t.Fatalf("install mode = %#v", plan.Targets[0].Install)
	}
	data, err := os.ReadFile(config)
	if err != nil || !strings.Contains(string(data), "model = \"gpt-5.6\"\nmodel_provider = \"openai\"\nmodel_reasoning_effort = \"high\"\n[features]") {
		t.Fatalf("config = %q err=%v", data, err)
	}
	if !strings.Contains(string(data), "[profiles.dev]\nmodel = \"legacy\"\n") {
		t.Fatalf("unrelated legacy profile lost: %q", data)
	}
	assertInstallTestFile(t, filepath.Join(filepath.Dir(config), "coding.config.toml"), "model_provider = \"openai\"\nmodel = \"gpt-5.6\"\nmodel_reasoning_effort = \"high\"\n")
}

func TestCodexNamedUndoRevertsOnlyTheLastInstall(t *testing.T) {
	request, config := codexNamedRequest(t, "coding", "review")
	applyNamed(t, request, "coding")
	manifest := config + manifestSuffix
	afterCoding, err := os.ReadFile(manifest)
	if err != nil {
		t.Fatal(err)
	}
	review := applyNamed(t, request, "review")
	dir := filepath.Dir(config)
	undone := applyUndo(t, UndoRequest{Target: request.Targets[0].Target, ConfigPath: config, Registry: request.Registry})
	if undone.OriginalPlanID != review.PlanID {
		t.Fatalf("undo selected %s, want review install %s", undone.OriginalPlanID, review.PlanID)
	}
	if _, err := os.Lstat(filepath.Join(dir, "review.config.toml")); !os.IsNotExist(err) {
		t.Fatalf("review profile survived undo: %v", err)
	}
	assertInstallTestFile(t, filepath.Join(dir, "coding.config.toml"), "model_provider = \"openai\"\nmodel = \"gpt-5.6\"\nmodel_reasoning_effort = \"high\"\n")
	assertInstallTestFile(t, manifest, string(afterCoding))
	assertInstallTestFile(t, config, codexNamedBase)
	applyUndo(t, UndoRequest{Target: request.Targets[0].Target, ConfigPath: config, Registry: request.Registry})
	for _, path := range []string{filepath.Join(dir, "coding.config.toml"), manifest} {
		if _, err := os.Lstat(path); !os.IsNotExist(err) {
			t.Fatalf("%s survived second undo: %v", filepath.Base(path), err)
		}
	}
	assertInstallTestFile(t, config, codexNamedBase)
}

func TestCodexNamedPlanRejectsStaleConfig(t *testing.T) {
	request, config := codexNamedRequest(t, "coding")
	request.ProfileName = "coding"
	plan, err := BuildPlan(request)
	if err != nil || plan.Status != StatusReady {
		t.Fatalf("plan status=%s err=%v", plan.Status, err)
	}
	writeInstallTestFile(t, config, "[profiles.coding]\nmodel = \"late\"\n")
	if _, err := ApplyPlan(plan, ApplyOptions{ExpectedPlanID: plan.PlanID}); err == nil {
		t.Fatal("apply accepted a config changed after planning")
	}
	if _, err := os.Lstat(filepath.Join(filepath.Dir(config), "coding.config.toml")); !os.IsNotExist(err) {
		t.Fatalf("stale apply wrote the profile: %v", err)
	}
}

// Profile names are already lowercase letters, digits and '-', a subset of what Codex accepts.
func TestCodexNamedProfileFileRejectsUnusableNames(t *testing.T) {
	if _, err := (codexAdapter{}).NamedProfileFile("has.dot"); err == nil || !strings.Contains(err.Error(), "letters, digits") {
		t.Fatalf("NamedProfileFile error = %v", err)
	}
}

func TestAgentsWithoutProfilesReportDefaultConfigMode(t *testing.T) {
	request, _ := testRequest(t, NewRegistry(testAdapterFor("fake", "1", "{}\n", true)))
	request.Default = true
	plan, err := BuildPlan(request)
	if err != nil {
		t.Fatal(err)
	}
	if got := plan.Targets[0].Install; got == nil || *got != (InstallMode{Mode: InstallModeDefaultConfig}) {
		t.Fatalf("install mode = %#v", got)
	}
}

type pathNamedAdapter struct {
	codexAdapter
	file string
}

func (adapter pathNamedAdapter) NamedProfileFile(string) (string, error) { return adapter.file, nil }

func TestSnapshotNamedFilePaths(t *testing.T) {
	root := t.TempDir()
	config := filepath.Join(root, "settings.json")
	outside := filepath.Join(t.TempDir(), "profile", "config.yaml")
	cases := map[string]struct {
		file string
		want string
	}{
		"nested relative": {file: "profiles/coding.json", want: filepath.Join(root, "profiles", "coding.json")},
		"absolute":        {file: outside, want: outside},
		"escape":          {file: "../coding.json"},
		"main config":     {file: "settings.json"},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			mode := &InstallMode{Mode: InstallModeNamedProfile, ProfileName: "coding"}
			snapshot, err := snapshotNamedFile(pathNamedAdapter{file: tc.file}, mode, installfsSnapshot(config))
			if tc.want == "" {
				if err == nil {
					t.Fatalf("path %q was accepted", tc.file)
				}
				return
			}
			if err != nil || snapshot.Path != tc.want || snapshot.Exists {
				t.Fatalf("snapshot = %+v, %v", snapshot, err)
			}
		})
	}
}

func installfsSnapshot(path string) installfs.Snapshot { return installfs.Snapshot{Path: path} }
