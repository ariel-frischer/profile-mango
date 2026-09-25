package install

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"gitlab.com/ariel-frischer/profile-mango/internal/installfs"
	"gitlab.com/ariel-frischer/profile-mango/pkg/adapters/claudecode"
	"gitlab.com/ariel-frischer/profile-mango/pkg/profilemango"
)

func TestClaudeCodeInstallMetadata(t *testing.T) {
	metadata := (claudeCodeAdapter{}).Metadata()
	if !metadata.Installable || metadata.Target != claudecode.TargetName || metadata.Version != claudecode.TargetVersion {
		t.Fatalf("metadata = %#v", metadata)
	}
	if !strings.Contains(metadata.Reason, "model") || !strings.Contains(metadata.Reason, "blocked") {
		t.Fatalf("metadata reason does not describe the narrow boundary: %q", metadata.Reason)
	}
}

func TestClaudeCodeInstallPlanPreservesTargetState(t *testing.T) {
	source := []byte("{\r\n  \"unknown\": true,\r\n  \"apiKey\": \"SYNTHETIC-CREDENTIAL\"\r\n}\r\n")
	patch, err := (claudeCodeAdapter{}).Plan(AdapterInput{
		Target:  Target{Name: claudecode.TargetName, Version: claudecode.TargetVersion},
		Profile: profilemango.ResolvedProfile{Metadata: profilemango.Metadata{Name: "route-only"}},
		Route:   claudeInstallRoute(),
		Config:  Snapshot{Content: source},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(patch.Files) != 1 || string(patch.Files[0].Content) != "{\r\n  \"model\": \"claude-sonnet-4-5\",\r\n  \"effortLevel\": \"high\",\r\n  \"unknown\": true,\r\n  \"apiKey\": \"SYNTHETIC-CREDENTIAL\"\r\n}\r\n" {
		t.Fatalf("patch files = %#v", patch.Files)
	}
	if len(patch.Fields) != 2 || patch.Fields[0].Path != "config.model" || patch.Fields[0].After != "claude-sonnet-4-5" || patch.Fields[1].Path != "config.effortLevel" || patch.Fields[1].After != "high" {
		t.Fatalf("patch fields = %#v", patch.Fields)
	}
	if !patch.OverrideAllowed || len(patch.Diagnostics) != 1 || patch.Diagnostics[0].Severity != profilemango.SeverityWarning {
		t.Fatalf("patch safety metadata = %#v", patch)
	}
}

func TestClaudeCodeInstallBlocksUnsupportedProfileEffects(t *testing.T) {
	mode := "read-only"
	tests := map[string]struct {
		profile profilemango.ResolvedProfile
		want    string
	}{
		"permissions": {
			profile: profilemango.ResolvedProfile{Permissions: &profilemango.PermissionPolicy{Mode: &mode}},
			want:    "permission requirements",
		},
		"tools": {
			profile: profilemango.ResolvedProfile{Tools: &profilemango.ResolvedRules{Managed: true}},
			want:    "tool requirements",
		},
		"instructions": {
			profile: profilemango.ResolvedProfile{Instructions: []string{"CLAUDE.md"}},
			want:    "instruction and skill delivery",
		},
		"skills": {
			profile: profilemango.ResolvedProfile{Skills: []string{"skills/research/SKILL.md"}},
			want:    "instruction and skill delivery",
		},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			input := AdapterInput{
				Target:  Target{Name: claudecode.TargetName, Version: claudecode.TargetVersion},
				Profile: test.profile,
				Route:   claudeInstallRoute(),
			}
			_, err := (claudeCodeAdapter{}).Plan(input)
			if err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("error = %v, want %q", err, test.want)
			}
		})
	}
}

func TestClaudeCodeInstallPlanApplyReapplyAndStale(t *testing.T) {
	request, root := claudeCodeTestRequest(t)
	request.Override = true
	config := request.Targets[0].ConfigPath
	before := "{\n  \"model\": \"old/model\",\n  \"unknown\": true\n}\n"
	writeInstallTestFile(t, config, before)
	plan, err := BuildPlan(request)
	if err != nil {
		t.Fatal(err)
	}
	if plan.Status != StatusReady || plan.Targets[0].Status != StatusReady {
		t.Fatalf("plan = %#v", plan)
	}
	want := []FieldChange{
		{Path: "config.effortLevel", After: "high"},
		{Path: "config.model", Before: "old/model", After: "claude-sonnet-4-5"},
		{Path: "profile.effortLevel", After: "high"},
		{Path: "profile.model", After: "claude-sonnet-4-5"},
	}
	if !plan.Backup || !reflect.DeepEqual(plan.Targets[0].Fields, want) {
		t.Fatalf("plan diff = %#v", plan.Targets[0].Fields)
	}
	settingsFile := claudeCodeFile(plan.Targets[0], "settings.json")
	if len(settingsFile.Fields) != 2 || settingsFile.Fields[0].Path != "config.model" || settingsFile.Fields[1].Path != "config.effortLevel" {
		t.Fatalf("settings.json file diff = %#v", settingsFile)
	}
	profileFile := claudeCodeFile(plan.Targets[0], "route-only.json")
	if len(profileFile.Fields) != 2 || profileFile.Fields[0].Path != "profile.model" || profileFile.Action != ActionCreate {
		t.Fatalf("profile file diff = %#v", profileFile)
	}
	if _, err := ApplyPlan(plan, ApplyOptions{ExpectedPlanID: plan.PlanID}); err != nil {
		t.Fatal(err)
	}
	assertInstallTestFile(t, config, "{\n  \"effortLevel\": \"high\",\n  \"model\": \"claude-sonnet-4-5\",\n  \"unknown\": true\n}\n")
	assertInstallTestFile(t, filepath.Join(filepath.Dir(config), "profiles", "route-only.json"), "{\n  \"model\": \"claude-sonnet-4-5\",\n  \"effortLevel\": \"high\"\n}\n")
	if _, err := os.Stat(installfs.BackupPath(config, plan.PlanID)); err != nil {
		t.Fatalf("backup missing: %v", err)
	}
	assertInstallTestFile(t, installfs.BackupPath(config, plan.PlanID), before)
	reapply, err := BuildPlan(request)
	if err != nil {
		t.Fatal(err)
	}
	if claudeCodeFile(reapply.Targets[0], "settings.json").Action != ActionNoop || claudeCodeFile(reapply.Targets[0], "route-only.json").Action != ActionNoop {
		t.Fatalf("reapply files = %#v", reapply.Targets[0].Files)
	}
	stale, err := BuildPlan(request)
	if err != nil {
		t.Fatal(err)
	}
	writeInstallTestFile(t, config, "{\n  \"model\": \"third-party/edit\"\n}\n")
	if _, err := ApplyPlan(stale, ApplyOptions{ExpectedPlanID: stale.PlanID}); !errors.Is(err, installfs.ErrStale) {
		t.Fatalf("stale apply error = %v", err)
	}
	if _, err := os.Stat(filepath.Join(root, "target")); err != nil {
		t.Fatalf("target directory disappeared: %v", err)
	}
}

func TestClaudeCodeInstallRejectsMalformedAndAdoptsUnowned(t *testing.T) {
	request, _ := claudeCodeTestRequest(t)
	config := request.Targets[0].ConfigPath
	writeInstallTestFile(t, config, "")
	empty, err := BuildPlan(request)
	if err != nil {
		t.Fatal(err)
	}
	if empty.Status != StatusBlocked || !strings.Contains(empty.Targets[0].Reason, "empty") {
		t.Fatalf("empty plan = %#v", empty)
	}

	writeInstallTestFile(t, config, `{ "model": `)
	malformed, err := BuildPlan(request)
	if err != nil {
		t.Fatal(err)
	}
	if malformed.Status != StatusBlocked || !strings.Contains(malformed.Targets[0].Reason, "valid JSON") {
		t.Fatalf("malformed plan = %#v", malformed)
	}
	writeInstallTestFile(t, config, `{ "model": "edited-by-user" }`)
	conflict, err := BuildPlan(request)
	if err != nil {
		t.Fatal(err)
	}
	if conflict.Status != StatusReady || claudeCodeFile(conflict.Targets[0], "settings.json").Action != ActionAdopt {
		t.Fatalf("unowned config was not adopted: %#v", conflict)
	}
}

func claudeCodeTestRequest(t *testing.T) (Request, string) {
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
	bindingsDir := filepath.Join(root, "bindings")
	if err := os.MkdirAll(bindingsDir, 0o755); err != nil {
		t.Fatal(err)
	}
	bindings := `routes:
  primary:
    provider: anthropic
    transport: native
    authentication: oauth
    model: claude-sonnet-4-5
    effort: high
`
	bindingsPath := filepath.Join(bindingsDir, "local.yaml")
	writeInstallTestFile(t, bindingsPath, bindings)
	config := filepath.Join(root, "target", "settings.json")
	if err := os.MkdirAll(filepath.Dir(config), 0o755); err != nil {
		t.Fatal(err)
	}
	return Request{
		ProfileName:  "route-only",
		ProfilesRoot: filepath.Join(root, "profiles"),
		ResourceRoot: root,
		BindingsPath: bindingsPath,
		Backup:       true,
		Override:     false,
		Registry:     NewRegistry(claudeCodeAdapter{}),
		Targets:      []TargetRequest{{Target: Target{Name: claudecode.TargetName, Version: claudecode.TargetVersion}, ConfigPath: config}},
		// Most tests in this file cover the settings.json patch, which only --default writes.
		Default: true,
	}, root
}

func claudeCodeFile(target TargetPlan, name string) FilePlan {
	for _, file := range target.Files {
		if file.Path == name {
			return file
		}
	}
	return FilePlan{}
}

func claudeInstallRoute() profilemango.RouteBinding {
	return profilemango.RouteBinding{
		Provider:       "anthropic",
		Transport:      "native",
		Authentication: "oauth",
		Model:          "claude-sonnet-4-5",
		Effort:         "high",
	}
}

const claudeCodeNamedBase = "{\n  \"unknown\": true\n}\n"

// claudeCodeNamedRequest plans named profiles into a synthetic settings.json directory,
// with request.Default left false so tests cover the emulated-profile-file default.
func claudeCodeNamedRequest(t *testing.T, names ...string) (Request, string) {
	t.Helper()
	request, _ := claudeCodeTestRequest(t)
	request.Default = false
	for _, name := range names {
		if err := os.MkdirAll(filepath.Join(request.ProfilesRoot, name), 0o755); err != nil {
			t.Fatal(err)
		}
		writeInstallTestFile(t, filepath.Join(request.ProfilesRoot, name, "profile.yaml"), "apiVersion: profilemango.dev/v1alpha1\nkind: PolicyProfile\nmetadata:\n  name: "+name+"\nspec:\n  routeRef: primary\n")
	}
	config := request.Targets[0].ConfigPath
	writeInstallTestFile(t, config, claudeCodeNamedBase)
	return request, config
}

func TestClaudeCodeNamedProfilesInstallSideBySideAndKeepSettings(t *testing.T) {
	request, config := claudeCodeNamedRequest(t, "coding", "review")
	plan := applyNamed(t, request, "coding")
	want := InstallMode{Mode: InstallModeNamedProfile, ProfileName: "coding", UseCommand: "claude --settings ~/.claude/profiles/coding.json"}
	if got := plan.Targets[0].Install; got == nil || *got != want {
		t.Fatalf("install mode = %#v, want %#v", got, want)
	}
	applyNamed(t, request, "review")
	assertInstallTestFile(t, config, claudeCodeNamedBase)
	dir := filepath.Dir(config)
	for _, name := range []string{"coding", "review"} {
		assertInstallTestFile(t, filepath.Join(dir, "profiles", name+".json"), "{\n  \"model\": \"claude-sonnet-4-5\",\n  \"effortLevel\": \"high\"\n}\n")
	}
	for _, name := range []string{"review", "coding"} {
		request.ProfileName = name
		again, err := BuildPlan(request)
		if err != nil || again.Status != StatusNoop {
			t.Fatalf("reinstall %s status=%s err=%v", name, again.Status, err)
		}
	}
}

func TestClaudeCodeNamedPlanJSONReportsModeAndUseCommand(t *testing.T) {
	request, _ := claudeCodeNamedRequest(t, "coding")
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
	if install["mode"] != "named-profile" || install["profileName"] != "coding" || install["useCommand"] != "claude --settings ~/.claude/profiles/coding.json" {
		t.Fatalf("install JSON = %#v", install)
	}
}

func TestClaudeCodeDefaultFlagInsertsModelOnOwnLineAndUndoRestoresBytes(t *testing.T) {
	request, config := claudeCodeNamedRequest(t, "coding")
	request.Default = true
	const pretty = "{\n    \"permissions\": {\n        \"allow\": [\"Read\"]\n    },\n    \"unknown\": true\n}\n"
	writeInstallTestFile(t, config, pretty)
	plan := applyNamed(t, request, "coding")
	if !plan.Targets[0].Install.SetsDefault {
		t.Fatalf("install mode = %#v", plan.Targets[0].Install)
	}
	assertInstallTestFile(t, config, "{\n    \"model\": \"claude-sonnet-4-5\",\n    \"effortLevel\": \"high\",\n    \"permissions\": {\n        \"allow\": [\"Read\"]\n    },\n    \"unknown\": true\n}\n")
	assertInstallTestFile(t, filepath.Join(filepath.Dir(config), "profiles", "coding.json"), "{\n  \"model\": \"claude-sonnet-4-5\",\n  \"effortLevel\": \"high\"\n}\n")
	applyUndo(t, UndoRequest{Target: request.Targets[0].Target, ConfigPath: config, Registry: request.Registry})
	assertInstallTestFile(t, config, pretty)
}

func TestClaudeCodeNamedUndoRevertsOnlyTheLastInstall(t *testing.T) {
	request, config := claudeCodeNamedRequest(t, "coding", "review")
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
	if _, err := os.Lstat(filepath.Join(dir, "profiles", "review.json")); !os.IsNotExist(err) {
		t.Fatalf("review profile survived undo: %v", err)
	}
	assertInstallTestFile(t, filepath.Join(dir, "profiles", "coding.json"), "{\n  \"model\": \"claude-sonnet-4-5\",\n  \"effortLevel\": \"high\"\n}\n")
	assertInstallTestFile(t, manifest, string(afterCoding))
	assertInstallTestFile(t, config, claudeCodeNamedBase)
	applyUndo(t, UndoRequest{Target: request.Targets[0].Target, ConfigPath: config, Registry: request.Registry})
	for _, path := range []string{filepath.Join(dir, "profiles", "coding.json"), manifest} {
		if _, err := os.Lstat(path); !os.IsNotExist(err) {
			t.Fatalf("%s survived second undo: %v", filepath.Base(path), err)
		}
	}
	assertInstallTestFile(t, config, claudeCodeNamedBase)
}

func TestClaudeCodeNamedPlanRejectsStaleConfig(t *testing.T) {
	request, config := claudeCodeNamedRequest(t, "coding")
	request.ProfileName = "coding"
	plan, err := BuildPlan(request)
	if err != nil || plan.Status != StatusReady {
		t.Fatalf("plan status=%s err=%v", plan.Status, err)
	}
	writeInstallTestFile(t, config, "{\n  \"model\": \"late\"\n}\n")
	if _, err := ApplyPlan(plan, ApplyOptions{ExpectedPlanID: plan.PlanID}); err == nil {
		t.Fatal("apply accepted a config changed after planning")
	}
	if _, err := os.Lstat(filepath.Join(filepath.Dir(config), "profiles", "coding.json")); !os.IsNotExist(err) {
		t.Fatalf("stale apply wrote the profile: %v", err)
	}
}

func TestClaudeCodeNamedProfileFileRejectsUnusableNames(t *testing.T) {
	if _, err := (claudeCodeAdapter{}).NamedProfileFile("has.dot"); err == nil || !strings.Contains(err.Error(), "letters, digits") {
		t.Fatalf("NamedProfileFile error = %v", err)
	}
}
