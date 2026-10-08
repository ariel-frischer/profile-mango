package install

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

const (
	roleTestProfile  = "route: primary\nroles:\n  worker:\n    description: Implements changes\n    instructions: roles/worker.md\n  research: {description: Explores read-only}\n"
	roleTestWorker   = "Edit only what the task names.\n"
	roleTestOriginal = "hand-written research agent\n"
	roleTestKeep     = "unrelated agent\n"
)

// roleTargetCase is one qualified target: its bindings, role file extension, and the
// exact files a default install writes for roleTestProfile.
type roleTargetCase struct {
	undoTargetCase
	ext            string
	worker, search string
}

func roleTargetCases() map[string]roleTargetCase {
	openAI := "routes:\n  primary:\n    provider: openai\n    transport: native\n    authentication: oauth\n    model: gpt-5.6\n    effort: high\n    roles:\n      worker: {provider: openai, model: gpt-5.6-codex, effort: xhigh}\n      research: {provider: openai, model: gpt-5.6-mini}\n"
	anthropic := "routes:\n  primary:\n    provider: anthropic\n    transport: native\n    authentication: oauth\n    model: claude-sonnet-4-5\n    effort: high\n    roles:\n      worker: {provider: anthropic, model: claude-opus-5-5, effort: xhigh}\n      research: {provider: anthropic, model: claude-haiku-5}\n"
	cases := undoTargetCases()
	return map[string]roleTargetCase{
		"codex": {cases["codex"].withBindings(openAI), ".toml",
			"name = \"worker\"\ndescription = \"Implements changes\"\ndeveloper_instructions = \"Edit only what the task names.\"\nmodel_provider = \"openai\"\nmodel = \"gpt-5.6-codex\"\nmodel_reasoning_effort = \"xhigh\"\n",
			"name = \"research\"\ndescription = \"Explores read-only\"\ndeveloper_instructions = \"Explores read-only\"\nmodel_provider = \"openai\"\nmodel = \"gpt-5.6-mini\"\n"},
		"opencode": {cases["opencode"].withBindings(openAI), ".md",
			"---\ndescription: \"Implements changes\"\nmode: \"subagent\"\nmodel: \"openai/gpt-5.6-codex\"\nvariant: \"xhigh\"\n---\n\nEdit only what the task names.\n",
			"---\ndescription: \"Explores read-only\"\nmode: \"subagent\"\nmodel: \"openai/gpt-5.6-mini\"\n---\n\nExplores read-only\n"},
		"oh-my-pi": {cases["oh-my-pi"].withBindings(openAI), ".md",
			"---\nname: \"worker\"\ndescription: \"Implements changes\"\nmodel: \"@task\"\n---\n\nEdit only what the task names.\n",
			"---\nname: \"research\"\ndescription: \"Explores read-only\"\nmodel: \"@smol\"\n---\n\nExplores read-only\n"},
		"claude-code": {cases["claude-code"].withBindings(anthropic), ".md",
			"---\nname: \"worker\"\ndescription: \"Implements changes\"\nmodel: \"claude-opus-5-5\"\neffort: \"xhigh\"\n---\n\nEdit only what the task names.\n",
			"---\nname: \"research\"\ndescription: \"Explores read-only\"\nmodel: \"claude-haiku-5\"\n---\n\nExplores read-only\n"},
	}
}

func (test undoTargetCase) withBindings(bindings string) undoTargetCase {
	test.bindings = bindings
	return test
}

// roleInstallRequest is a default install of roleTestProfile under a synthetic home,
// with a hand-written research agent and an unrelated agent already present.
func roleInstallRequest(t *testing.T, test roleTargetCase) (Request, string, string) {
	t.Helper()
	request, root := testRequest(t, DefaultRegistry())
	writeInstallTestFile(t, request.BindingsPath, test.bindings)
	writeInstallTestFile(t, filepath.Join(root, "profiles", "route-only", "profile.yaml"), roleTestProfile)
	for dir, content := range map[string]string{"roles/worker.md": roleTestWorker, "profiles/plain/profile.yaml": "route: primary\n"} {
		if err := os.MkdirAll(filepath.Dir(filepath.Join(root, dir)), 0o755); err != nil {
			t.Fatal(err)
		}
		writeInstallTestFile(t, filepath.Join(root, dir), content)
	}
	request.Default, request.Override = true, true
	request.Env = syntheticPathEnv(t, filepath.Join(root, "home"), nil)
	request.Targets = []TargetRequest{{Target: test.target()}}
	config, err := DefaultRegistry().DefaultConfigPath(test.target(), request.Env)
	if err != nil {
		t.Fatal(err)
	}
	agents := filepath.Join(filepath.Dir(config), "agents")
	if err := os.MkdirAll(agents, 0o700); err != nil {
		t.Fatal(err)
	}
	writeInstallTestFile(t, config, test.original)
	writeInstallTestFile(t, filepath.Join(agents, "research"+test.ext), roleTestOriginal)
	writeInstallTestFile(t, filepath.Join(agents, "keep"+test.ext), roleTestKeep)
	return request, config, agents
}

// TestRoleFilesInstallAdoptAndUndo installs each qualified target's role files at its
// default path, adopting the hand-written research agent, then undoes byte-identically.
func TestRoleFilesInstallAdoptAndUndo(t *testing.T) {
	for name, test := range roleTargetCases() {
		t.Run(name, func(t *testing.T) {
			request, config, agents := roleInstallRequest(t, test)
			applySwitchTestPlan(t, request)
			assertInstallTestFile(t, filepath.Join(agents, "worker"+test.ext), test.worker)
			assertInstallTestFile(t, filepath.Join(agents, "research"+test.ext), test.search)
			assertInstallTestFile(t, filepath.Join(agents, "keep"+test.ext), roleTestKeep)
			entry, owned := manifestEntry(readSwitchManifest(t, config), filepath.Join(agents, "research"+test.ext))
			if !owned || !slices.Contains(entry.Fields, ownershipRoleDefinition) {
				t.Fatalf("research agent is not owned as a role definition: %#v", entry)
			}
			applyUndo(t, UndoRequest{Target: test.target(), ConfigPath: config})
			assertInstallTestFile(t, config, test.original)
			assertInstallTestFile(t, filepath.Join(agents, "research"+test.ext), roleTestOriginal)
			assertInstallTestFile(t, filepath.Join(agents, "keep"+test.ext), roleTestKeep)
			if _, err := os.Stat(filepath.Join(agents, "worker"+test.ext)); !os.IsNotExist(err) {
				t.Fatalf("undo left the created worker agent: %v", err)
			}
		})
	}
}

// TestRoleFilesUseReleasesRolesTheNewProfileLacks switches to a profile without roles:
// the created worker agent is deleted and the adopted research agent gets its bytes back.
func TestRoleFilesUseReleasesRolesTheNewProfileLacks(t *testing.T) {
	for name, test := range roleTargetCases() {
		t.Run(name, func(t *testing.T) {
			request, config, agents := roleInstallRequest(t, test)
			applySwitchTestPlan(t, request)
			request.ProfileName, request.Release = "plain", true
			applySwitchTestPlan(t, request)
			if _, err := os.Stat(filepath.Join(agents, "worker"+test.ext)); !os.IsNotExist(err) {
				t.Fatalf("use kept the released worker agent: %v", err)
			}
			assertInstallTestFile(t, filepath.Join(agents, "research"+test.ext), roleTestOriginal)
			assertInstallTestFile(t, filepath.Join(agents, "keep"+test.ext), roleTestKeep)
			if _, owned := manifestEntry(readSwitchManifest(t, config), filepath.Join(agents, "research"+test.ext)); owned {
				t.Fatal("released research agent is still owned")
			}
		})
	}
}

// TestRoleFileGates checks the requirements a role install reports instead of writing:
// an effort the target cannot write, a provider it cannot select, and an unqualified target.
func TestRoleFileGates(t *testing.T) {
	cases := roleTargetCases()
	tests := map[string]struct {
		test     roleTargetCase
		from, to string
		want     SkippedRequirement
		strict   bool
	}{
		"codex max effort is not applied": {test: cases["codex"], from: "effort: xhigh}", to: "effort: max}",
			want: SkippedRequirement{Requirement: RequirementEffort, Value: "max", Role: "worker", Reason: "Codex 0.157.1 model_reasoning_effort is installed only as none, minimal, low, medium, high, xhigh"}},
		"claude minimal effort is not applied": {test: cases["claude-code"], from: "effort: xhigh}", to: "effort: minimal}",
			want: SkippedRequirement{Requirement: RequirementEffort, Value: "minimal", Role: "worker", Reason: "Claude Code agent file effort accepts only low, medium, high, xhigh, max"}},
		"codex foreign provider keeps the file without a model": {test: cases["codex"], from: "worker: {provider: openai", to: "worker: {provider: anthropic",
			want: SkippedRequirement{Requirement: RequirementRoles, Role: "worker", Reason: "Codex role files select only the built-in openai provider, not anthropic"}},
		"pi is unqualified": {test: roleTargetCase{undoTargetCase: undoTargetCases()["pi"].withBindings(cases["codex"].bindings)},
			want: SkippedRequirement{Requirement: RequirementRoleDefinitions, Count: 2, Reason: roleDefinitionsUnsupportedReason}},
		"strict blocks a skipped effort": {test: cases["codex"], from: "effort: xhigh}", to: "effort: max}", strict: true},
	}
	for name, gate := range tests {
		t.Run(name, func(t *testing.T) {
			gate.test.bindings = strings.Replace(gate.test.bindings, gate.from, gate.to, 1)
			request, _, _ := roleInstallRequest(t, gate.test)
			request.Strict = gate.strict
			plan, err := BuildPlan(request)
			if err != nil {
				t.Fatal(err)
			}
			target := plan.Targets[0]
			if gate.strict {
				if target.Status != StatusBlocked || !planHasCode(plan, "install.strict_requirement_unsupported") {
					t.Fatalf("strict status = %s (%s)", target.Status, target.Reason)
				}
				return
			}
			if target.Status != StatusReady || !slices.Contains(target.SkippedRequirements, gate.want) {
				t.Fatalf("status = %s (%s), skipped = %#v", target.Status, target.Reason, target.SkippedRequirements)
			}
		})
	}
}

// TestRoleFilesTargetRoleOverride installs one binding into Oh My Pi and Codex: the
// research role keeps the base ChatGPT OAuth provider openai-codex on Oh My Pi, and
// targets.codex.roles.research switches it to Codex's built-in openai provider.
func TestRoleFilesTargetRoleOverride(t *testing.T) {
	bindings := "routes:\n  primary:\n    provider: openai-codex\n    model: gpt-6-sol\n    effort: high\n" +
		"    roles:\n      research:\n        provider: openai-codex\n        model: gpt-6-luna\n" +
		"    targets:\n      codex:\n        provider: openai\n        roles:\n          research:\n            provider: openai\n"
	cases := roleTargetCases()
	tests := map[string]struct {
		test roleTargetCase
		file string
		want string
	}{
		"oh-my-pi modelRoles keep openai-codex": {test: cases["oh-my-pi"], file: "config", want: "smol: \"openai-codex/gpt-6-luna\"\n"},
		"codex role file gets openai":           {test: cases["codex"], file: "agents/research.toml", want: "model_provider = \"openai\"\nmodel = \"gpt-6-luna\"\n"},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			test.test.bindings = bindings
			request, config, _ := roleInstallRequest(t, test.test)
			writeInstallTestFile(t, filepath.Join(request.ProfilesRoot, request.ProfileName, "profile.yaml"), "route: primary\nroles:\n  research: {description: Explores read-only}\n")
			plan, err := BuildPlan(request)
			if err != nil {
				t.Fatal(err)
			}
			target := plan.Targets[0]
			if target.Status != StatusReady || len(target.SkippedRequirements) != 0 {
				t.Fatalf("status = %s (%s), skipped = %#v", target.Status, target.Reason, target.SkippedRequirements)
			}
			applySwitchTestPlan(t, request)
			path := config
			if test.file != "config" {
				path = filepath.Join(filepath.Dir(config), test.file)
			}
			data, err := os.ReadFile(path)
			if err != nil || !strings.Contains(string(data), test.want) {
				t.Fatalf("%s lacks %q (%v):\n%s", path, test.want, err, data)
			}
		})
	}
}

func TestCustomRoleFilesInstallAndRelease(t *testing.T) {
	rename := strings.NewReplacer("worker", "coder", "research", "reviewer", "@task", "@coder", "@smol", "@reviewer")
	for name, test := range roleTargetCases() {
		t.Run(name, func(t *testing.T) {
			test.bindings = rename.Replace(test.bindings)
			request, _, agents := roleInstallRequest(t, test)
			root := request.ResourceRoot
			writeInstallTestFile(t, filepath.Join(root, "profiles", "route-only", "profile.yaml"), rename.Replace(roleTestProfile))
			writeInstallTestFile(t, filepath.Join(root, "roles", "coder.md"), roleTestWorker)
			applySwitchTestPlan(t, request)
			assertInstallTestFile(t, filepath.Join(agents, "coder"+test.ext), rename.Replace(test.worker))
			assertInstallTestFile(t, filepath.Join(agents, "reviewer"+test.ext), rename.Replace(test.search))
			request.ProfileName, request.Release = "plain", true
			applySwitchTestPlan(t, request)
			for _, role := range []string{"coder", "reviewer"} {
				if _, err := os.Stat(filepath.Join(agents, role+test.ext)); !os.IsNotExist(err) {
					t.Fatalf("released custom role %s remains: %v", role, err)
				}
			}
			assertInstallTestFile(t, filepath.Join(agents, "keep"+test.ext), roleTestKeep)
		})
	}
}
