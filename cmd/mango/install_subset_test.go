package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// subsetSandbox extends the --all sandbox profile with requirements codex cannot install.
func subsetSandbox(t *testing.T) installOptions {
	t.Helper()
	options, _ := allSkipSandbox(t)
	root := options.resourceRoot
	for path, content := range map[string]string{"instructions/a.md": "a\n", "instructions/b.md": "b\n", "skills/tdd/SKILL.md": "skill\n"} {
		if err := os.MkdirAll(filepath.Dir(filepath.Join(root, path)), 0o755); err != nil {
			t.Fatal(err)
		}
		writeFile(t, filepath.Join(root, path), content)
	}
	writeFile(t, filepath.Join(options.profiles, "route-only", "profile.yaml"), "apiVersion: profilemango.dev/v1alpha1\nkind: PolicyProfile\nmetadata:\n  name: route-only\nspec:\n  routeRef: route\n  permissions:\n    mode: read-only\n  tools:\n    allow: [read]\n  instructions:\n    append: [instructions/a.md, instructions/b.md]\n  skills: [skills/tdd/SKILL.md]\n")
	return options
}

func TestInstallListsSkippedRequirementsAndPlansReady(t *testing.T) {
	options := subsetSandbox(t)
	output := runInstallForTest(t, options)
	want := "  codex@0.154.0: ready | destination:"
	skipped := "    not installed for this agent: permissions, tools, instructions (2 files), skills (1)\n"
	if !strings.Contains(output, want) || !strings.Contains(output, skipped) {
		t.Fatalf("subset plan must be ready and list skips:\n%s", output)
	}
	if !strings.Contains(output, "--all --apply --yes --expect-plan") || strings.Contains(output, "--strict") {
		t.Fatalf("apply hint must reproduce the non-strict plan:\n%s", output)
	}
	options.jsonOutput = true
	data := runInstallForTest(t, options)
	if !strings.Contains(data, `"skippedRequirements"`) || !strings.Contains(data, `"strict": false`) {
		t.Fatalf("JSON plan must list skipped requirements:\n%s", data)
	}
}

func TestInstallStrictBlocksUnsupportedRequirements(t *testing.T) {
	options := subsetSandbox(t)
	options.strict = true
	output, err := runInstallResult(t, options)
	if err == nil || !strings.Contains(err.Error(), "install plan is blocked") {
		t.Fatalf("strict install error = %v\n%s", err, output)
	}
	if !strings.Contains(output, "codex@0.154.0: blocked") || strings.Contains(output, "not installed for this agent") {
		t.Fatalf("strict plan must block without listing skips:\n%s", output)
	}
}

func TestInstallApplyHintKeepsStrict(t *testing.T) {
	args := installPlanArgs(installOptions{all: true, strict: true})
	if strings.Join(args, " ") != "--all --strict" {
		t.Fatalf("plan args = %q", args)
	}
}
