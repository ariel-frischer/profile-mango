package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gitlab.com/ariel-frischer/profile-mango/pkg/adapters/codex"
)

func TestRenderNoPreviewDoesNotWrite(t *testing.T) {
	profiles, resources, bindings := writeRenderFixture(t, false)
	out := filepath.Join(t.TempDir(), "candidate")
	stdout, stderr, err := executeRenderForTest(t, []string{
		"route-only", "--profiles", profiles, "--resource-root", resources, "--bindings", bindings,
		"--target", codex.TargetName, "--target-version", codex.TargetVersion, "--out", out, "--json",
	})
	if err == nil {
		t.Fatal("blocked render unexpectedly succeeded")
	}
	if _, statErr := os.Stat(out); !os.IsNotExist(statErr) {
		t.Fatalf("no-preview render created output: %v", statErr)
	}
	var report codex.Result
	if err := json.Unmarshal([]byte(stdout), &report); err != nil {
		t.Fatalf("stdout is not render JSON: %v\n%s", err, stdout)
	}
	if report.Preview || report.Applicable || !strings.Contains(stderr, "codex.route.authentication_unverified") {
		t.Fatalf("unexpected blocked report/output: %#v\n%s", report, stderr)
	}
	if strings.Contains(stdout, profiles) || strings.Contains(stdout, resources) || strings.Contains(stdout, bindings) {
		t.Fatal("render JSON leaked an input path")
	}
}

func TestRenderPreviewWritesOnlyInertArtifacts(t *testing.T) {
	profiles, resources, bindings := writeRenderFixture(t, true)
	out := filepath.Join(t.TempDir(), "candidate")
	stdout, stderr, err := executeRenderForTest(t, []string{
		"read-only", "--profiles", profiles, "--resource-root", resources, "--bindings", bindings,
		"--target", codex.TargetName, "--target-version", codex.TargetVersion, "--out", out, "--preview", "--json",
	})
	if err == nil {
		t.Fatal("blocked preview unexpectedly succeeded")
	}
	var report codex.Result
	if err := json.Unmarshal([]byte(stdout), &report); err != nil {
		t.Fatalf("stdout is not render JSON: %v\n%s", err, stdout)
	}
	if !report.Preview || report.Applicable {
		t.Fatalf("unexpected preview report: %#v", report)
	}
	if !strings.Contains(stderr, "codex.security.permissions_unverified") || !strings.Contains(stderr, "codex.security.tools_unverified") {
		t.Fatalf("security blockers missing: %s", stderr)
	}
	for _, relative := range []string{
		"render.json",
		"preview/read-only.config.toml.preview",
		"resources/instructions/research.md",
		"resources/skills/research/SKILL.md",
	} {
		assertRenderFile(t, out, relative)
	}
	for _, forbidden := range []string{"config.toml", "AGENTS.md", "plan.json", "manifest.json", "skills"} {
		if _, statErr := os.Stat(filepath.Join(out, forbidden)); !os.IsNotExist(statErr) {
			t.Fatalf("forbidden active artifact exists: %s", forbidden)
		}
	}
	reportData, err := os.ReadFile(filepath.Join(out, "render.json"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(reportData), codex.EvidenceSHA256) {
		t.Fatal("render.json omitted exact evidence hash")
	}
}

func TestRenderRejectsInvalidProfileWithoutPreviewOutput(t *testing.T) {
	profiles := t.TempDir()
	writeFile(t, filepath.Join(profiles, "unsupported", "profile.yaml"), "apiVersion: profilemango.dev/v1alpha1\nkind: PolicyProfile\nmetadata:\n  name: unsupported\nspec:\n  routeRef: codex-oauth\nroles: [reviewer]\n")
	resources := t.TempDir()
	bindings := writeBindingsFixture(t)
	out := filepath.Join(t.TempDir(), "candidate")
	_, _, err := executeRenderForTest(t, []string{
		"unsupported", "--profiles", profiles, "--resource-root", resources, "--bindings", bindings,
		"--target", codex.TargetName, "--target-version", codex.TargetVersion, "--out", out, "--preview",
	})
	if err == nil {
		t.Fatal("invalid profile unexpectedly succeeded")
	}
	if _, statErr := os.Stat(out); !os.IsNotExist(statErr) {
		t.Fatalf("invalid input created output: %v", statErr)
	}
}

func TestRenderRequiresExplicitFlags(t *testing.T) {
	cmd := newRenderCmd()
	cmd.SetArgs([]string{"route-only"})
	if err := cmd.Execute(); err == nil || !strings.Contains(err.Error(), "--profiles is required") {
		t.Fatalf("missing explicit flag error = %v", err)
	}
}

func TestRenderPreservesExistingOutput(t *testing.T) {
	profiles, resources, bindings := writeRenderFixture(t, false)
	out := filepath.Join(t.TempDir(), "candidate")
	_, _, firstErr := executeRenderForTest(t, []string{
		"route-only", "--profiles", profiles, "--resource-root", resources, "--bindings", bindings,
		"--target", codex.TargetName, "--target-version", codex.TargetVersion, "--out", out, "--preview",
	})
	if firstErr == nil {
		t.Fatal("blocked preview unexpectedly succeeded")
	}
	marker := filepath.Join(out, "marker")
	writeFile(t, marker, "keep")
	_, _, secondErr := executeRenderForTest(t, []string{
		"route-only", "--profiles", profiles, "--resource-root", resources, "--bindings", bindings,
		"--target", codex.TargetName, "--target-version", codex.TargetVersion, "--out", out, "--preview",
	})
	if secondErr == nil {
		t.Fatal("existing output unexpectedly overwritten")
	}
	data, err := os.ReadFile(marker)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "keep" {
		t.Fatalf("existing marker changed: %q", data)
	}
}

func executeRenderForTest(t *testing.T, args []string) (string, string, error) {
	t.Helper()
	cmd := newRenderCmd()
	var stdout, stderr bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	cmd.SetArgs(args)
	err := cmd.Execute()
	return stdout.String(), stderr.String(), err
}

func writeRenderFixture(t *testing.T, constrained bool) (string, string, string) {
	t.Helper()
	profiles := t.TempDir()
	resources := t.TempDir()
	name := "route-only"
	profile := "apiVersion: profilemango.dev/v1alpha1\nkind: PolicyProfile\nmetadata:\n  name: route-only\nspec:\n  routeRef: codex-oauth\n  instructions:\n    append: [instructions/system.md]\n"
	resourcePath := filepath.Join(resources, "instructions", "system.md")
	if constrained {
		name = "read-only"
		profile = "apiVersion: profilemango.dev/v1alpha1\nkind: PolicyProfile\nmetadata:\n  name: read-only\nspec:\n  routeRef: codex-oauth\n  permissions:\n    mode: read-only\n    network: allow\n    shell: deny\n  tools:\n    allow: [read]\n    deny: [write]\n  instructions:\n    append: [instructions/research.md]\n  skills: [skills/research/SKILL.md]\n"
		resourcePath = filepath.Join(resources, "instructions", "research.md")
		writeFile(t, filepath.Join(resources, "skills", "research", "SKILL.md"), "skill\n")
	}
	writeFile(t, filepath.Join(profiles, name, "profile.yaml"), profile)
	writeFile(t, resourcePath, "research\n")
	return profiles, resources, writeBindingsFixture(t)
}

func writeBindingsFixture(t *testing.T) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "bindings.yaml")
	writeFile(t, path, "routes:\n  codex-oauth:\n    provider: openai\n    transport: native\n    authentication: oauth\n    model: gpt-5.6\n    effort: high\n")
	return path
}

func assertRenderFile(t *testing.T, root, relative string) {
	t.Helper()
	if _, err := os.Stat(filepath.Join(root, filepath.FromSlash(relative))); err != nil {
		t.Fatalf("missing expected file %s: %v", relative, err)
	}
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}
