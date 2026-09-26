package install

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

// subsetTestRequest plans codex with a profile that also carries requirements codex cannot honor.
func subsetTestRequest(t *testing.T) Request {
	t.Helper()
	request, root := codexTestRequest(t)
	request.Registry = DefaultRegistry()
	for path, content := range map[string]string{"instructions/a.md": "a\n", "instructions/b.md": "b\n", "skills/tdd/SKILL.md": "skill\n"} {
		if err := os.MkdirAll(filepath.Dir(filepath.Join(root, path)), 0o755); err != nil {
			t.Fatal(err)
		}
		writeInstallTestFile(t, filepath.Join(root, path), content)
	}
	request.ResourceRoot = root
	writeInstallTestFile(t, filepath.Join(request.ProfilesRoot, "route-only", "profile.yaml"), `apiVersion: profilemango.dev/v1alpha1
kind: PolicyProfile
metadata:
  name: route-only
spec:
  routeRef: primary
  permissions:
    mode: read-only
  tools:
    allow: [read]
  instructions:
    append: [instructions/a.md, instructions/b.md]
  skills: [skills/tdd/SKILL.md]
`)
	return request
}

func TestSubsetInstallSkipsUnsupportedRequirements(t *testing.T) {
	plan, err := BuildPlan(subsetTestRequest(t))
	if err != nil {
		t.Fatal(err)
	}
	target := plan.Targets[0]
	if plan.Status != StatusReady || target.Status != StatusReady || plan.Strict {
		t.Fatalf("subset plan = %s target=%s (%s)", plan.Status, target.Status, target.Reason)
	}
	want := []SkippedRequirement{{Requirement: "permissions"}, {Requirement: "tools"}, {Requirement: "instructions", Count: 2}, {Requirement: "skills", Count: 1}}
	if !reflect.DeepEqual(target.SkippedRequirements, want) {
		t.Fatalf("skipped = %#v, want %#v", target.SkippedRequirements, want)
	}
	data, err := plan.JSON()
	if err != nil || !strings.Contains(string(data), `"skippedRequirements"`) || !strings.Contains(string(data), `"requirement": "instructions"`) {
		t.Fatalf("plan JSON lacks skipped requirements: %s, err=%v", data, err)
	}
	if !hasFileAction(target, ActionCreate) || len(target.Fields) == 0 {
		t.Fatalf("supported subset was not planned: %#v", target)
	}
}

func TestStrictInstallBlocksUnsupportedRequirements(t *testing.T) {
	request := subsetTestRequest(t)
	request.Strict = true
	plan, err := BuildPlan(request)
	if err != nil {
		t.Fatal(err)
	}
	target := plan.Targets[0]
	if plan.Status != StatusBlocked || target.Status != StatusBlocked || !plan.Strict || !strings.Contains(target.Reason, "permission requirements") {
		t.Fatalf("strict plan = %s target=%#v", plan.Status, target)
	}
	if len(target.SkippedRequirements) != 0 {
		t.Fatalf("strict plan must not skip: %#v", target.SkippedRequirements)
	}
}

func TestPlanIDCoversStrictAndSkips(t *testing.T) {
	request := subsetTestRequest(t)
	subset, err := BuildPlan(request)
	if err != nil {
		t.Fatal(err)
	}
	request.Strict = true
	strict, err := BuildPlan(request)
	if err != nil {
		t.Fatal(err)
	}
	if subset.PlanID == strict.PlanID {
		t.Fatal("strict flag does not change the plan ID")
	}
	withoutSkips := subset
	withoutSkips.Targets = append([]TargetPlan(nil), subset.Targets...)
	withoutSkips.Targets[0].SkippedRequirements = nil
	id, err := planID(withoutSkips)
	if err != nil || id == subset.PlanID {
		t.Fatalf("skipped requirements do not change the plan ID: %v", err)
	}
}

func TestOpenCodeKeepsSupportedRequirements(t *testing.T) {
	named := AgentDestination{Mode: "primary", Name: "coder"}
	if _, _, skipped := supportedSubset(openCodeAdapter{}, AgentDestination{}, false, loadedInput{}); len(skipped) != 0 {
		t.Fatalf("empty profile skipped = %#v", skipped)
	}
	if supportsRequirement(openCodeAdapter{}, AgentDestination{}, false, RequirementSkills) || supportsRequirement(codexAdapter{}, AgentDestination{}, false, RequirementSkills) {
		t.Fatal("opencode must skip skills without --default and codex must always skip them")
	}
	if !supportsRequirement(openCodeAdapter{}, AgentDestination{}, true, RequirementSkills) {
		t.Fatal("opencode must keep skills alongside --default")
	}
	if !supportsRequirement(openCodeAdapter{}, named, false, RequirementInstructions) || supportsRequirement(openCodeAdapter{}, named, false, RequirementSkills) || supportsRequirement(openCodeAdapter{}, named, true, RequirementSkills) {
		t.Fatal("a named opencode agent must keep instructions and always skip skills")
	}
}

// TestOpenCodeSubsetSkipsSkillsAndPlansNamedAgent covers an agent-empty install without
// --default: instructions are supported (rendered into the named agent), but the profile's
// skill is skipped, since OpenCode skills are global and only install alongside --default.
func TestOpenCodeSubsetSkipsSkillsAndPlansNamedAgent(t *testing.T) {
	request, root := openCodeTestRequest(t)
	writeInstallTestFile(t, filepath.Join(root, "profiles", "route-only", "profile.yaml"), `apiVersion: profilemango.dev/v1alpha1
kind: PolicyProfile
metadata:
  name: route-only
spec:
  routeRef: primary
  instructions:
    append: [instructions/system.md]
  skills:
    - skills/research/SKILL.md
`)
	if err := os.MkdirAll(filepath.Join(root, "instructions"), 0o755); err != nil {
		t.Fatal(err)
	}
	writeInstallTestFile(t, filepath.Join(root, "instructions", "system.md"), "synthetic instruction\n")
	if err := os.MkdirAll(filepath.Join(root, "skills", "research"), 0o755); err != nil {
		t.Fatal(err)
	}
	writeInstallTestFile(t, filepath.Join(root, "skills", "research", "SKILL.md"), openCodeTestSkill)
	plan, err := BuildPlan(request)
	if err != nil {
		t.Fatal(err)
	}
	want := []SkippedRequirement{{Requirement: RequirementSkills, Count: 1}}
	if plan.Status != StatusReady || !reflect.DeepEqual(plan.Targets[0].SkippedRequirements, want) {
		t.Fatalf("opencode subset plan = %s %#v", plan.Status, plan.Targets[0].SkippedRequirements)
	}
	if !hasFileAction(plan.Targets[0], ActionCreate) {
		t.Fatalf("named agent file was not planned: %#v", plan.Targets[0].Files)
	}
}

const rolesProfile = "name: route-only\nroute: primary\nroles:\n  worker: {description: Implements}\n  research: {description: Scouts read-only}\n"

const rolesSubagentBindings = "routes:\n  primary:\n    provider: openai\n    model: gpt-5.6\n    effort: high\n    subagentMaxEffort: medium\n    roles:\n      research:\n        provider: openai\n        model: gpt-5.6-mini\n      planner:\n        provider: openai\n        model: gpt-5.6\n        effort: xhigh\n"

// TestCodexRoleSubsetByInstallMode checks what a Codex install keeps of the profile's
// roles: a default install writes declared roles as subagent files and skips only the
// route role no definition carries; a named-only install skips every role requirement.
func TestCodexRoleSubsetByInstallMode(t *testing.T) {
	tests := map[string]struct {
		named, strict bool
		status        string
		want          []SkippedRequirement
		strictReason  string
	}{
		"default": {status: StatusReady, want: []SkippedRequirement{
			{Requirement: RequirementRoles, Count: 1, Reason: rolesUnsupportedReason},
			{Requirement: RequirementSubagentMaxEffort, Value: "medium", Reason: subagentMaxEffortUnsupportedReason},
		}},
		"default strict": {strict: true, status: StatusBlocked, strictReason: "1 route roles cannot be installed"},
		"named": {named: true, status: StatusReady, want: []SkippedRequirement{
			{Requirement: RequirementRoleDefinitions, Count: 2, Reason: roleFilesNamedOnlyReason},
			{Requirement: RequirementRoles, Count: 2, Reason: rolesUnsupportedReason},
			{Requirement: RequirementSubagentMaxEffort, Value: "medium", Reason: subagentMaxEffortUnsupportedReason},
		}},
		"named strict": {named: true, strict: true, status: StatusBlocked, strictReason: "2 role definitions cannot be installed: " + roleFilesNamedOnlyReason},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			request, root := codexTestRequest(t)
			request.Registry = DefaultRegistry()
			request.Default, request.Strict = !test.named, test.strict
			writeInstallTestFile(t, filepath.Join(root, "profiles", "route-only", "profile.yaml"), rolesProfile)
			writeInstallTestFile(t, request.BindingsPath, rolesSubagentBindings)
			plan, err := BuildPlan(request)
			if err != nil {
				t.Fatal(err)
			}
			target := plan.Targets[0]
			if target.Status != test.status || !strings.Contains(target.Reason, test.strictReason) {
				t.Fatalf("target = %s (%s)", target.Status, target.Reason)
			}
			if !test.strict && !reflect.DeepEqual(target.SkippedRequirements, test.want) {
				t.Fatalf("skipped = %#v", target.SkippedRequirements)
			}
			if wrote := hasPlannedFile(target, "agents/research.toml"); wrote != (test.status == StatusReady && !test.named) {
				t.Fatalf("research role file planned = %v: %#v", wrote, target.Files)
			}
		})
	}
}

func hasPlannedFile(target TargetPlan, path string) bool {
	for _, file := range target.Files {
		if file.Path == path {
			return true
		}
	}
	return false
}

func TestStrictBlocksRouteRolesWithoutRoleDefinitions(t *testing.T) {
	request, _ := codexTestRequest(t)
	request.Registry = DefaultRegistry()
	request.Strict = true
	writeInstallTestFile(t, request.BindingsPath, rolesSubagentBindings)
	plan, err := BuildPlan(request)
	if err != nil {
		t.Fatal(err)
	}
	if target := plan.Targets[0]; target.Status != StatusBlocked || !strings.Contains(target.Reason, "2 route roles cannot be installed") {
		t.Fatalf("strict target = %s (%s)", target.Status, target.Reason)
	}
}
