package install

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

const (
	agentTestScout    = "---\nname: scout\ndescription: Fast read-only scout\ntools: read, grep\nthinking-level: low\n---\n\nScout the repo.\n"
	agentTestOriginal = "---\nname: scout\ndescription: hand-written\n---\n\nmine\n"
)

// agentFileRequest is a default install of a profile whose agentFiles give target one
// file named file, under a synthetic home, plus a "plain" profile without agentFiles.
// It returns the request, the target's config path, and its subagent directory.
func agentFileRequest(t *testing.T, target, file string) (Request, string, string) {
	t.Helper()
	test := undoTargetCases()[target]
	request, root := testRequest(t, DefaultRegistry())
	writeInstallTestFile(t, request.BindingsPath, test.bindings)
	profile := "route: primary\nagentFiles:\n  " + target + ":\n    " + file + ": agents/scout-src.md\n"
	writeInstallTestFile(t, filepath.Join(root, "profiles", "route-only", "profile.yaml"), profile)
	for path, content := range map[string]string{"agents/scout-src.md": agentTestScout, "profiles/plain/profile.yaml": "route: primary\n"} {
		if err := os.MkdirAll(filepath.Dir(filepath.Join(root, path)), 0o755); err != nil {
			t.Fatal(err)
		}
		writeInstallTestFile(t, filepath.Join(root, path), content)
	}
	request.Default, request.Override = true, false
	request.Env = syntheticPathEnv(t, filepath.Join(root, "home"), nil)
	request.Targets = []TargetRequest{{Target: test.target()}}
	config, err := DefaultRegistry().DefaultConfigPath(test.target(), request.Env)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Dir(config), 0o700); err != nil {
		t.Fatal(err)
	}
	writeInstallTestFile(t, config, test.original)
	return request, config, filepath.Join(filepath.Dir(config), "agents")
}

// TestAgentFilesInstallAndUseRelease installs an Oh My Pi agent file verbatim as an owned
// whole file, then switches to a profile without it: a created file is removed and an
// adopted hand-written file gets its original bytes back.
func TestAgentFilesInstallAndUseRelease(t *testing.T) {
	tests := map[string]struct {
		existing bool
		prior    string
	}{
		"created file is removed on release":        {prior: priorAbsent},
		"adopted file gets its original bytes back": {existing: true},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			request, config, agents := agentFileRequest(t, "oh-my-pi", "scout.md")
			scout := filepath.Join(agents, "scout.md")
			if test.existing {
				if err := os.MkdirAll(agents, 0o700); err != nil {
					t.Fatal(err)
				}
				writeInstallTestFile(t, scout, agentTestOriginal)
			}
			applySwitchTestPlan(t, request)
			assertInstallTestFile(t, scout, agentTestScout)
			entry, owned := manifestEntry(readSwitchManifest(t, config), scout)
			if !owned || !slices.Contains(entry.Fields, ownershipAgentFile) || entryPrior(entry) == "" {
				t.Fatalf("scout.md is not owned as an agent file with its prior state: %#v", entry)
			}
			if test.prior != "" && entryPrior(entry) != test.prior {
				t.Fatalf("prior = %s, want %s", entryPrior(entry), test.prior)
			}
			request.ProfileName, request.Release = "plain", true
			applySwitchTestPlan(t, request)
			if test.existing {
				assertInstallTestFile(t, scout, agentTestOriginal)
			} else if _, err := os.Stat(scout); !os.IsNotExist(err) {
				t.Fatalf("created scout.md survived release: %v", err)
			}
			if _, owned := manifestEntry(readSwitchManifest(t, config), scout); owned {
				t.Fatal("released scout.md is still owned")
			}
		})
	}
}

// TestAgentFilesStatusAndDrift reports an installed agent file as kind agent-file,
// flags a hand edit as edited drift, and refuses to overwrite it without --override.
func TestAgentFilesStatusAndDrift(t *testing.T) {
	request, config, agents := agentFileRequest(t, "oh-my-pi", "scout.md")
	applySwitchTestPlan(t, request)
	status := agentFileStatus(t, request)
	if status.Kind != ownershipAgentFile || status.State != FileInSync {
		t.Fatalf("installed agents/scout.md status = %#v", status)
	}
	writeInstallTestFile(t, filepath.Join(agents, "scout.md"), "edited by hand\n")
	if status := agentFileStatus(t, request); status.State != FileEdited {
		t.Fatalf("edited agents/scout.md state = %s", status.State)
	}
	plan, err := BuildPlan(request)
	if err != nil {
		t.Fatal(err)
	}
	if plan.Targets[0].Status != StatusConflict {
		t.Fatalf("edited agent file plan status = %s", plan.Targets[0].Status)
	}
	request.Override = true
	applySwitchTestPlan(t, request)
	assertInstallTestFile(t, filepath.Join(agents, "scout.md"), agentTestScout)
	if _, owned := manifestEntry(readSwitchManifest(t, config), filepath.Join(agents, "scout.md")); !owned {
		t.Fatal("overridden scout.md is no longer owned")
	}
}

func agentFileStatus(t *testing.T, request Request) FileStatus {
	t.Helper()
	report, err := InspectStatus(StatusRequest{ProfilesRoot: request.ProfilesRoot, ResourceRoot: request.ResourceRoot, BindingsPath: request.BindingsPath, Registry: request.Registry, Env: request.Env, Targets: request.Targets})
	if err != nil {
		t.Fatal(err)
	}
	for _, file := range report.Targets[0].Files {
		if file.Path == "agents/scout.md" {
			return file
		}
	}
	t.Fatalf("status lists no agents/scout.md: %#v", report.Targets[0].Files)
	return FileStatus{}
}

// TestAgentFileGates checks where an agent file installs, and what a plan reports
// instead of writing it: a target without subagent files, a named-only install, the
// wrong role-file extension, and a missing resource.
func TestAgentFileGates(t *testing.T) {
	tests := map[string]struct {
		target, file string
		named        bool
		strict       bool
		skipped      *SkippedRequirement
		code         string
		writes       string
	}{
		"codex installs a toml agent":   {target: "codex", file: "scout.toml", writes: "agents/scout.toml"},
		"claude code installs an agent": {target: "claude-code", file: "scout.md", writes: "agents/scout.md"},
		"pi without subagent files skips": {target: "pi", file: "scout.md",
			skipped: &SkippedRequirement{Requirement: RequirementAgentFiles, Count: 1, Reason: roleDefinitionsUnsupportedReason}},
		"pi strict blocks": {target: "pi", file: "scout.md", strict: true, code: "install.strict_requirement_unsupported"},
		"named-only install skips": {target: "oh-my-pi", file: "scout.md", named: true,
			skipped: &SkippedRequirement{Requirement: RequirementAgentFiles, Count: 1, Reason: roleFilesNamedOnlyReason}},
		"named-only strict blocks": {target: "oh-my-pi", file: "scout.md", named: true, strict: true, code: "install.strict_requirement_unsupported"},
		"codex markdown blocks":    {target: "codex", file: "scout.md", code: "install.agent_file_extension_invalid"},
		"omp toml blocks":          {target: "oh-my-pi", file: "scout.toml", code: "install.agent_file_extension_invalid"},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			request, config, _ := agentFileRequest(t, test.target, test.file)
			request.Default, request.Strict = !test.named, test.strict
			plan, err := BuildPlan(request)
			if err != nil {
				t.Fatal(err)
			}
			if got := hasSkipped(plan.Targets, RequirementAgentFiles); got != (test.skipped != nil) || (got && !slices.Contains(plan.Targets[0].SkippedRequirements, *test.skipped)) {
				t.Fatalf("skipped = %#v, want %#v", plan.Targets[0].SkippedRequirements, test.skipped)
			}
			if test.code != "" && (plan.Status != StatusBlocked || !planHasCode(plan, test.code)) {
				t.Fatalf("status = %s, want blocked with %s: %s", plan.Status, test.code, plan.Targets[0].Reason)
			}
			if test.writes == "" {
				return
			}
			applySwitchTestPlan(t, request)
			assertInstallTestFile(t, filepath.Join(filepath.Dir(config), test.writes), agentTestScout)
		})
	}
}

func TestAgentFileMissingResourceFailsToLoad(t *testing.T) {
	request, _, _ := agentFileRequest(t, "oh-my-pi", "scout.md")
	if err := os.Remove(filepath.Join(request.ResourceRoot, "agents", "scout-src.md")); err != nil {
		t.Fatal(err)
	}
	if _, err := BuildPlan(request); err == nil || !strings.Contains(err.Error(), "resource.read") {
		t.Fatalf("plan with a missing agent file resource: %v", err)
	}
}
