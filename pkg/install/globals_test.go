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

// TestGlobalInstructionsComposeFragments installs a list-form global file: each
// fragment renders its {{route.…}} placeholders, every fragment but the last loses
// its trailing newlines and gains one blank line, and the last is kept verbatim.
func TestGlobalInstructionsComposeFragments(t *testing.T) {
	tests := map[string]struct {
		files     string
		fragments map[string]string
		want      string
	}{
		"one blank line between fragments": {files: "[a.md, b.md]",
			fragments: map[string]string{"a.md": "# core\n", "b.md": "# omp\n"},
			want:      "# core\n\n# omp\n"},
		"trailing newlines trimmed, last verbatim": {files: "[a.md, b.md, c.md]",
			fragments: map[string]string{"a.md": "# core\n\n\n", "b.md": "no newline", "c.md": "# last\n\n"},
			want:      "# core\n\nno newline\n\n# last\n\n"},
		"placeholders render per fragment": {files: "[a.md, b.md]",
			fragments: map[string]string{"a.md": "model {{route.primary.model}}\n", "b.md": "effort {{route.primary.effort}}"},
			want:      "model gpt-5.6\n\neffort high"},
		"one-item list equals scalar": {files: "[a.md]",
			fragments: map[string]string{"a.md": "# only\n\n"},
			want:      "# only\n\n"},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			request, root, agents := globalTestRequest(t, "    AGENTS.md: "+test.files+"\n")
			for file, content := range test.fragments {
				writeInstallTestFile(t, filepath.Join(root, file), content)
			}
			applySwitchTestPlan(t, request)
			assertInstallTestFile(t, agents, test.want)
		})
	}
}

// TestGlobalInstructionsMissingFragmentFails names the missing fragment's path.
func TestGlobalInstructionsMissingFragmentFails(t *testing.T) {
	request, root, _ := globalTestRequest(t, "    AGENTS.md: [a.md, shared/missing.md]\n")
	writeInstallTestFile(t, filepath.Join(root, "a.md"), "# core\n")
	_, err := BuildPlan(request)
	if err == nil || !strings.Contains(err.Error(), "globalInstructions.opencode.AGENTS.md") || !strings.Contains(err.Error(), "resource shared/missing.md does not exist") {
		t.Fatalf("missing fragment error = %v", err)
	}
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

// TestHomeInstructionGates plans globalInstructions.home: only Pi installs ~/AGENTS.md,
// another target skips it (or blocks with --strict), and unqualified names block.
func TestHomeInstructionGates(t *testing.T) {
	tests := map[string]homeGateCase{
		"pi default installs ~/AGENTS.md": {file: "AGENTS.md", writes: true},
		"unresolvable home skips":         {file: "AGENTS.md", noHome: true, skipped: true},
		"unqualified home file blocks":    {file: "RULES.md", code: "install.global_instruction_unqualified"},
		"non-owner target skips":          {opencode: true, file: "AGENTS.md", skipped: true},
		"non-owner target strict blocks":  {opencode: true, file: "AGENTS.md", strict: true, code: "install.strict_requirement_unsupported"},
		"unresolvable home strict blocks": {file: "AGENTS.md", noHome: true, strict: true, code: "install.strict_requirement_unsupported"},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			request, root := piInstallTestRequest(t)
			if test.opencode {
				request, root = openCodeTestRequest(t)
			}
			home := filepath.Join(root, "home")
			request.Default, request.Override, request.Strict = true, false, test.strict
			if !test.noHome {
				request.Env = syntheticPathEnv(t, home, nil)
			}
			writeInstallTestFile(t, filepath.Join(root, "home.md"), globalTestWork)
			writeInstallTestFile(t, filepath.Join(root, "profiles", "route-only", "profile.yaml"), "route: primary\nglobalInstructions:\n  home:\n    "+test.file+": home.md\n")
			test.check(t, request, filepath.Join(home, "AGENTS.md"))
		})
	}
}

type homeGateCase struct {
	opencode, noHome, strict, skipped, writes bool
	file, code                                string
}

// check plans request, asserts the skip and block outcome, applies a writing case,
// and asserts whether ~/AGENTS.md exists afterwards.
func (test homeGateCase) check(t *testing.T, request Request, agents string) {
	t.Helper()
	plan, err := BuildPlan(request)
	if err != nil {
		t.Fatal(err)
	}
	if test.skipped != hasSkipped(plan.Targets, RequirementGlobalInstructions) {
		t.Fatalf("skipped globalInstructions = %v, want %v", !test.skipped, test.skipped)
	}
	if test.code != "" && (plan.Status != StatusBlocked || !planHasCode(plan, test.code)) {
		t.Fatalf("status = %s, want blocked with %s", plan.Status, test.code)
	}
	if test.writes {
		applySwitchTestPlan(t, request)
	}
	if _, err := os.Stat(agents); test.writes != (err == nil) {
		t.Fatalf("~/AGENTS.md written = %v, want %v", err == nil, test.writes)
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
