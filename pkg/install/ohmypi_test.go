package install

import (
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"

	"github.com/ariel-frischer/profile-mango/internal/installfs"
	"github.com/ariel-frischer/profile-mango/pkg/profilemango"
)

func TestOhMyPiInstallPreservesStateAndReapplies(t *testing.T) {
	request, _ := ohMyPiTestRequest(t)
	config := request.Targets[0].ConfigPath
	before := "# keep\nmodelRoles:\n  reviewer: other/model\n  default: old/model # owned\ndefaultThinkingLevel: low # owned\nunknown:\n  apiKey: SYNTHETIC\n"
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
	if got := fieldPaths(plan.Targets[0].Fields); got != "config.modelRoles.default,profile.modelRoles.default" {
		t.Fatalf("fields = %s", got)
	}
	if len(plan.Targets[0].Files) != 3 {
		t.Fatalf("files = %#v", plan.Targets[0].Files)
	}

	if _, err := ApplyPlan(plan, ApplyOptions{ExpectedPlanID: plan.PlanID}); err != nil {
		t.Fatal(err)
	}
	want := "# keep\nmodelRoles:\n  reviewer: other/model\n  default: \"openai/gpt-5.6:high\" # owned\ndefaultThinkingLevel: low # owned\nunknown:\n  apiKey: SYNTHETIC\n"
	assertInstallTestFile(t, config, want)
	assertInstallTestFile(t, installfs.BackupPath(config, plan.PlanID), before)
	info, err := os.Stat(config)
	if err != nil || info.Mode().Perm() != 0o600 {
		t.Fatalf("config mode = %v, err = %v", info.Mode().Perm(), err)
	}

	reapply, err := BuildPlan(request)
	if err != nil {
		t.Fatal(err)
	}
	if reapply.Status != StatusNoop || reapply.Targets[0].Files[0].Action != ActionNoop {
		t.Fatalf("reapply = %#v", reapply)
	}
}

const ohMyPiRolesBindings = `routes:
  primary:
    provider: anthropic
    model: claude-opus-5-5
    effort: medium
    subagentMaxEffort: high
    roles:
      worker:
        provider: anthropic
        model: claude-opus-5-5
        effort: medium
      planner:
        provider: anthropic
        model: claude-opus-5-5
        effort: high
      research:
        provider: opencode-go
        model: gpt-6-luna
        effort: high
      tiny:
        provider: opencode-go
        model: glm-5.3-flash
        effort: low
`

func TestOhMyPiInstallRolesPreservesUnrelatedKeysAndUndoes(t *testing.T) {
	test := undoTargetCase{name: "oh-my-pi", bindings: ohMyPiRolesBindings}
	original := "# keep\nmodelRoles:\n  reviewer: other/model\n  smol: old/fast:low # mine\ndefaultThinkingLevel: xhigh\nunknown:\n  apiKey: SYNTHETIC\n"
	config, _ := installAtDefault(t, test, &original)
	want := "# keep\nmodelRoles:\n  reviewer: other/model\n  smol: \"opencode-go/gpt-6-luna:high\" # mine\n" +
		"  default: \"anthropic/claude-opus-5-5:medium\"\n  commit: \"opencode-go/glm-5.3-flash:low\"\n  plan: \"anthropic/claude-opus-5-5:high\"\n  slow: \"anthropic/claude-opus-5-5:high\"\n" +
		"  task: \"anthropic/claude-opus-5-5:medium\"\n  tiny: \"opencode-go/glm-5.3-flash:low\"\n" +
		"defaultThinkingLevel: xhigh\nunknown:\n  apiKey: SYNTHETIC\ntask:\n  maxEffort: \"high\"\n"
	assertInstallTestFile(t, config, want)
	manifest, err := os.ReadFile(config + ".profile-mango.manifest.json")
	if err != nil {
		t.Fatal(err)
	}
	for _, field := range []string{"modelRoles.default", "modelRoles.commit", "modelRoles.plan", "modelRoles.slow", "modelRoles.smol", "modelRoles.task", "modelRoles.tiny", "task.maxEffort"} {
		if !strings.Contains(string(manifest), `"config.`+field+`"`) {
			t.Fatalf("manifest does not own %s:\n%s", field, manifest)
		}
	}
	if strings.Contains(string(manifest), "defaultThinkingLevel") {
		t.Fatalf("manifest claims defaultThinkingLevel:\n%s", manifest)
	}
	applyUndo(t, UndoRequest{Target: test.target(), ConfigPath: config})
	assertInstallTestFile(t, config, original)
}

func TestOhMyPiRolePlanReportsEveryRoleField(t *testing.T) {
	request, _ := ohMyPiTestRequest(t)
	request.Default = false
	writeInstallTestFile(t, request.BindingsPath, ohMyPiRolesBindings)
	request.Strict = true
	plan, err := BuildPlan(request)
	if err != nil {
		t.Fatal(err)
	}
	target := plan.Targets[0]
	if plan.Status != StatusReady || len(target.SkippedRequirements) != 0 {
		t.Fatalf("strict roles plan = %s skipped=%v reason=%s", plan.Status, target.SkippedRequirements, target.Reason)
	}
	after := make(map[string]string, len(target.Fields))
	for _, field := range target.Fields {
		after[field.Path] = field.After
	}
	want := map[string]string{
		"profile.modelRoles.default": "anthropic/claude-opus-5-5:medium",
		"profile.modelRoles.commit":  "opencode-go/glm-5.3-flash:low",
		"profile.modelRoles.plan":    "anthropic/claude-opus-5-5:high",
		"profile.modelRoles.slow":    "anthropic/claude-opus-5-5:high",
		"profile.modelRoles.smol":    "opencode-go/gpt-6-luna:high",
		"profile.modelRoles.task":    "anthropic/claude-opus-5-5:medium",
		"profile.modelRoles.tiny":    "opencode-go/glm-5.3-flash:low",
		"profile.task.maxEffort":     "high",
	}
	if !reflect.DeepEqual(after, want) {
		t.Fatalf("fields = %v, want %v", after, want)
	}
}

func TestOhMyPiOldSlotRoleNamesFailWithPortableHint(t *testing.T) {
	request, _ := ohMyPiTestRequest(t)
	writeInstallTestFile(t, request.BindingsPath, "routes:\n  primary:\n    provider: openai\n    model: gpt-5.6\n    effort: high\n    roles:\n      smol:\n        provider: openai\n        model: gpt-5.6-mini\n")
	_, err := BuildPlan(request)
	if err == nil || !strings.Contains(err.Error(), `routes.primary.roles.smol`) || !strings.Contains(err.Error(), `use the portable role "research"`) {
		t.Fatalf("err = %v", err)
	}
}

// TestOhMyPiRoleFilesNameModelSlots checks that each declared role becomes a subagent
// file whose model is the @ alias of the modelRoles slot the same install writes.
func TestOhMyPiRoleFilesNameModelSlots(t *testing.T) {
	request, root := ohMyPiTestRequest(t)
	request.Strict = true
	writeInstallTestFile(t, filepath.Join(root, "profiles", "route-only", "profile.yaml"), rolesProfile+"  tiny: {description: Writes commit messages}\n")
	writeInstallTestFile(t, request.BindingsPath, ohMyPiRolesBindings)
	applySwitchTestPlan(t, request)
	agents := filepath.Join(filepath.Dir(request.Targets[0].ConfigPath), "agents")
	want := map[string]string{
		"worker.md":   "---\nname: \"worker\"\ndescription: \"Implements\"\nmodel: \"@task\"\n---\n\nImplements\n",
		"research.md": "---\nname: \"research\"\ndescription: \"Scouts read-only\"\nmodel: \"@smol\"\n---\n\nScouts read-only\n",
		"tiny.md":     "---\nname: \"tiny\"\ndescription: \"Writes commit messages\"\nmodel: \"@commit\"\n---\n\nWrites commit messages\n",
	}
	for name, content := range want {
		assertInstallTestFile(t, filepath.Join(agents, name), content)
	}
}

func TestOhMyPiInstallBindsDestinationAndRejectsStalePlan(t *testing.T) {
	request, root := ohMyPiTestRequest(t)
	config := request.Targets[0].ConfigPath
	writeInstallTestFile(t, config, "modelRoles:\n  default: old/model\ndefaultThinkingLevel: low\n")
	first, err := BuildPlan(request)
	if err != nil {
		t.Fatal(err)
	}
	other := filepath.Join(root, "other", "config.yml")
	if err := os.MkdirAll(filepath.Dir(other), 0o755); err != nil {
		t.Fatal(err)
	}
	writeInstallTestFile(t, other, "modelRoles:\n  default: old/model\ndefaultThinkingLevel: low\n")
	secondRequest := request
	secondRequest.Targets = []TargetRequest{{Target: request.Targets[0].Target, ConfigPath: other}}
	second, err := BuildPlan(secondRequest)
	if err != nil {
		t.Fatal(err)
	}
	if first.PlanID == second.PlanID || first.Targets[0].DestinationSHA256 == second.Targets[0].DestinationSHA256 {
		t.Fatal("plan was not bound to the explicit destination")
	}

	writeInstallTestFile(t, config, "# third-party edit\nmodelRoles:\n  default: edited/model\n")
	if _, err := ApplyPlan(first, ApplyOptions{ExpectedPlanID: first.PlanID}); !errors.Is(err, installfs.ErrStale) {
		t.Fatalf("stale apply error = %v", err)
	}
	assertInstallTestFile(t, config, "# third-party edit\nmodelRoles:\n  default: edited/model\n")
}

func TestOhMyPiAdapterBlocksUnverifiedRequirements(t *testing.T) {
	mode := "read-only"
	tests := map[string]struct {
		profile profilemango.ResolvedProfile
		route   profilemango.RouteBinding
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
		"resources": {
			profile: profilemango.ResolvedProfile{Instructions: []string{"instructions/system.md"}},
			want:    "delivery remains install-blocking",
		},
		"proxy route": {
			route: profilemango.RouteBinding{Provider: "openai", Model: "gpt-5.6", Transport: "proxy", Authentication: "oauth", Effort: "high"},
			want:  "native transport",
		},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			route := test.route
			if route.Provider == "" {
				route = ohMyPiRoute()
			}
			_, err := (ohMyPiAdapter{}).Plan(AdapterInput{Target: Target{Name: "oh-my-pi", Version: "18.6.0"}, Profile: test.profile, Route: route})
			if err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("error = %v, want substring %q", err, test.want)
			}
		})
	}
}

func TestOhMyPiMetadataReportsNarrowNativeApplicability(t *testing.T) {
	metadata := (ohMyPiAdapter{}).Metadata()
	if !metadata.Installable || metadata.Status != StatusReady {
		t.Fatalf("metadata = %#v, want ready and installable", metadata)
	}
	for _, want := range []string{"Settings.loadReadOnly", "modelRoles selectors", ":effort suffix", "remain unmanaged"} {
		if !strings.Contains(metadata.Reason, want) {
			t.Fatalf("reason = %q, want substring %q", metadata.Reason, want)
		}
	}
}

func ohMyPiTestRequest(t *testing.T) (Request, string) {
	t.Helper()
	request, root := testRequest(t, NewRegistry(ohMyPiAdapter{}))
	writeInstallTestFile(t, request.BindingsPath, ohMyPiTestBindings)
	// Most tests here cover the config.yml patch, which only --default writes.
	request.Override, request.Default = true, true
	request.Targets = []TargetRequest{{Target: Target{Name: "oh-my-pi", Version: "18.6.0"}, ConfigPath: filepath.Join(root, "target", "config.yml")}}
	if err := os.MkdirAll(filepath.Dir(request.Targets[0].ConfigPath), 0o755); err != nil {
		t.Fatal(err)
	}
	return request, root
}

func ohMyPiRoute() profilemango.RouteBinding {
	return profilemango.RouteBinding{Provider: "openai", Transport: "native", Authentication: "oauth", Model: "gpt-5.6", Effort: "high"}
}

func fieldPaths(fields []FieldChange) string {
	paths := make([]string, 0, len(fields))
	for _, field := range fields {
		paths = append(paths, field.Path)
	}
	sort.Strings(paths)
	return strings.Join(paths, ",")
}
