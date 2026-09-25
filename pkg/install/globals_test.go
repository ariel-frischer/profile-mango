package install

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

const (
	globalTestOriginal = "# hand-written global rules\n"
	globalTestWork     = "# work rules\n"
)

// globalTestRequest is an OpenCode default install whose route-only profile owns a
// global AGENTS.md and whose "plain" profile owns none.
func globalTestRequest(t *testing.T, files string) (Request, string, string) {
	t.Helper()
	request, root := openCodeTestRequest(t)
	request.Default, request.Override = true, false
	writeInstallTestFile(t, filepath.Join(root, "instructions.md"), globalTestWork)
	writeInstallTestFile(t, filepath.Join(root, "profiles", "route-only", "profile.yaml"), "route: primary\nglobalInstructions:\n  opencode:\n"+files)
	plain := filepath.Join(root, "profiles", "plain")
	if err := os.MkdirAll(plain, 0o755); err != nil {
		t.Fatal(err)
	}
	writeInstallTestFile(t, filepath.Join(plain, "profile.yaml"), "route: primary\n")
	return request, root, filepath.Join(filepath.Dir(request.Targets[0].ConfigPath), "AGENTS.md")
}

func TestGlobalInstructionsInstallAndUseRelease(t *testing.T) {
	tests := map[string]struct {
		existing string
		exists   bool
	}{
		"created file is deleted on release":        {},
		"adopted file gets its original bytes back": {existing: globalTestOriginal, exists: true},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			request, _, agents := globalTestRequest(t, "    AGENTS.md: instructions.md\n")
			if test.exists {
				writeInstallTestFile(t, agents, test.existing)
			}
			applySwitchTestPlan(t, request)
			assertInstallTestFile(t, agents, globalTestWork)
			entry, owned := manifestEntry(readSwitchManifest(t, request.Targets[0].ConfigPath), agents)
			if !owned || !slices.Contains(entry.Fields, ownershipGlobalInstruction) {
				t.Fatalf("AGENTS.md is not owned as a global instruction: %#v", entry)
			}
			request.ProfileName, request.Release = "plain", true
			applySwitchTestPlan(t, request)
			if !test.exists {
				if _, err := os.Stat(agents); !os.IsNotExist(err) {
					t.Fatalf("created AGENTS.md survived release: %v", err)
				}
			} else {
				assertInstallTestFile(t, agents, test.existing)
			}
			if _, owned := manifestEntry(readSwitchManifest(t, request.Targets[0].ConfigPath), agents); owned {
				t.Fatal("released AGENTS.md is still in the ownership manifest")
			}
		})
	}
}

func TestGlobalInstructionsInstallWithoutReleaseKeepsFile(t *testing.T) {
	request, _, agents := globalTestRequest(t, "    AGENTS.md: instructions.md\n")
	applySwitchTestPlan(t, request)
	request.ProfileName = "plain"
	applySwitchTestPlan(t, request)
	assertInstallTestFile(t, agents, globalTestWork)
}

func TestGlobalInstructionsDriftBlocksUntilOverride(t *testing.T) {
	request, root, agents := globalTestRequest(t, "    AGENTS.md: instructions.md\n")
	applySwitchTestPlan(t, request)
	writeInstallTestFile(t, agents, "# edited by hand\n")
	writeInstallTestFile(t, filepath.Join(root, "instructions.md"), globalTestWork+"more\n")
	plan, err := BuildPlan(request)
	if err != nil {
		t.Fatal(err)
	}
	if plan.Targets[0].Status != StatusConflict {
		t.Fatalf("drifted AGENTS.md status = %s", plan.Targets[0].Status)
	}
	request.Override = true
	applySwitchTestPlan(t, request)
	assertInstallTestFile(t, agents, globalTestWork+"more\n")
}

func TestGlobalInstructionsReleaseBlocksDriftedFile(t *testing.T) {
	request, _, agents := globalTestRequest(t, "    AGENTS.md: instructions.md\n")
	applySwitchTestPlan(t, request)
	writeInstallTestFile(t, agents, "# edited by hand\n")
	request.ProfileName, request.Release = "plain", true
	plan, err := BuildPlan(request)
	if err != nil {
		t.Fatal(err)
	}
	if plan.Status != StatusBlocked {
		t.Fatalf("release of drifted AGENTS.md status = %s", plan.Status)
	}
	assertInstallTestFile(t, agents, "# edited by hand\n")
}

func TestGlobalInstructionsTargetGates(t *testing.T) {
	tests := map[string]struct {
		files   string
		strict  bool
		named   bool
		code    string
		skipped bool
	}{
		"unqualified file blocks":        {files: "    RULES.md: instructions.md\n", code: "install.global_instruction_unqualified"},
		"named profile skips":            {files: "    AGENTS.md: instructions.md\n", named: true, skipped: true},
		"named profile strict blocks":    {files: "    AGENTS.md: instructions.md\n", named: true, strict: true, code: "install.strict_requirement_unsupported"},
		"missing resource fails to load": {files: "    AGENTS.md: missing.md\n", code: "resource.read"},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			request, _, agents := globalTestRequest(t, test.files)
			request.Default, request.Strict = !test.named, test.strict
			plan, err := BuildPlan(request)
			if err != nil {
				if test.code == "" || !strings.Contains(err.Error(), test.code) {
					t.Fatal(err)
				}
				return
			}
			if test.skipped != hasSkipped(plan.Targets, RequirementGlobalInstructions) {
				t.Fatalf("skipped globalInstructions = %v, want %v", !test.skipped, test.skipped)
			}
			if test.code != "" && (plan.Status != StatusBlocked || !planHasCode(plan, test.code)) {
				t.Fatalf("status = %s, want blocked with %s: %#v", plan.Status, test.code, plan.Diagnostics)
			}
			if _, err := os.Stat(agents); !os.IsNotExist(err) {
				t.Fatalf("planning wrote AGENTS.md: %v", err)
			}
		})
	}
}

func hasSkipped(targets []TargetPlan, requirement string) bool {
	for _, target := range targets {
		for _, skipped := range target.SkippedRequirements {
			if skipped.Requirement == requirement {
				return true
			}
		}
	}
	return false
}

func planHasCode(plan Plan, code string) bool {
	if hasSwitchDiagnostic(plan.Diagnostics, code) {
		return true
	}
	for _, target := range plan.Targets {
		if hasSwitchDiagnostic(target.Diagnostics, code) {
			return true
		}
	}
	return false
}
