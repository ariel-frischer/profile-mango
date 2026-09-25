package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gitlab.com/ariel-frischer/profile-mango/internal/profilehome"
	"gitlab.com/ariel-frischer/profile-mango/pkg/adapters/claudecode"
	"gitlab.com/ariel-frischer/profile-mango/pkg/adapters/codex"
	"gitlab.com/ariel-frischer/profile-mango/pkg/adapters/hermes"
	"gitlab.com/ariel-frischer/profile-mango/pkg/adapters/jcodefork"
	"gitlab.com/ariel-frischer/profile-mango/pkg/adapters/ohmypi"
	"gitlab.com/ariel-frischer/profile-mango/pkg/adapters/openclaw"
	"gitlab.com/ariel-frischer/profile-mango/pkg/adapters/pi"
	"gitlab.com/ariel-frischer/profile-mango/pkg/render"
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

func TestRenderClaudeCodePreviewUsesTargetDispatch(t *testing.T) {
	profiles, resources, bindings := writeRenderFixture(t, false)
	out := filepath.Join(t.TempDir(), "candidate")
	stdout, stderr, err := executeRenderForTest(t, []string{
		"route-only", "--profiles", profiles, "--resource-root", resources, "--bindings", bindings,
		"--target", claudecode.TargetName, "--target-version", claudecode.TargetVersion, "--out", out, "--preview", "--json",
	})
	if err == nil {
		t.Fatal("blocked Claude Code preview unexpectedly succeeded")
	}
	var report render.Result
	if err := json.Unmarshal([]byte(stdout), &report); err != nil {
		t.Fatalf("stdout is not render JSON: %v\n%s", err, stdout)
	}
	if report.Target != claudecode.TargetName || report.AdapterVersion != claudecode.AdapterVersion || !report.Preview || report.Applicable {
		t.Fatalf("unexpected Claude Code report: %#v", report)
	}
	if !strings.Contains(stderr, "claudecode.config.acceptance_unverified") {
		t.Fatalf("Claude Code support blocker missing: %s", stderr)
	}
	assertRenderFile(t, out, "render.json")
	assertRenderFile(t, out, "preview/route-only.settings.json.preview")
	assertRenderFile(t, out, "resources/instructions/system.md")
	config, readErr := os.ReadFile(filepath.Join(out, "preview", "route-only.settings.json.preview"))
	if readErr != nil {
		t.Fatal(readErr)
	}
	if strings.Contains(string(config), "oauth") || strings.Contains(string(config), "secret") || strings.Contains(string(config), "openai") {
		t.Fatalf("candidate config leaked authentication data: %s", config)
	}
}

func TestRenderArielJcodePreviewUsesExperimentalDispatch(t *testing.T) {
	profiles, resources, bindings := writeRenderFixture(t, false)
	out := filepath.Join(t.TempDir(), "candidate")
	stdout, stderr, err := executeRenderForTest(t, []string{
		"route-only", "--profiles", profiles, "--resource-root", resources, "--bindings", bindings,
		"--target", jcodefork.TargetName, "--target-version", jcodefork.TargetVersion, "--out", out, "--preview", "--json",
	})
	if err == nil {
		t.Fatal("blocked Ariel custom Jcode preview unexpectedly succeeded")
	}
	var report render.Result
	if err := json.Unmarshal([]byte(stdout), &report); err != nil {
		t.Fatalf("stdout is not render JSON: %v\n%s", err, stdout)
	}
	if report.Target != jcodefork.TargetName || report.AdapterVersion != jcodefork.AdapterVersion || !report.Preview || report.Applicable {
		t.Fatalf("unexpected Ariel custom Jcode report: %#v", report)
	}
	if !strings.Contains(stderr, "jcodefork.experimental_only") || !strings.Contains(stderr, "jcodefork.route.authentication_unverified") {
		t.Fatalf("experimental blockers missing: %s", stderr)
	}
	assertRenderFile(t, out, "render.json")
	assertRenderFile(t, out, "preview/route-only.config.toml.preview")
	assertRenderFile(t, out, "resources/instructions/system.md")
	config, readErr := os.ReadFile(filepath.Join(out, "preview", "route-only.config.toml.preview"))
	if readErr != nil {
		t.Fatal(readErr)
	}
	for _, forbidden := range []string{"oauth", "credential", "secret", "authentication", "provider_profile", "agents_md_path"} {
		if strings.Contains(strings.ToLower(string(config)), forbidden) {
			t.Fatalf("candidate config leaked forbidden field %q: %s", forbidden, config)
		}
	}
}

func TestRenderOhMyPiPreviewUsesTargetDispatch(t *testing.T) {
	profiles, resources, bindings := writeRenderFixture(t, false)
	out := filepath.Join(t.TempDir(), "candidate")
	stdout, stderr, err := executeRenderForTest(t, []string{
		"route-only", "--profiles", profiles, "--resource-root", resources, "--bindings", bindings,
		"--target", ohmypi.TargetName, "--target-version", ohmypi.TargetVersion, "--out", out, "--preview", "--json",
	})
	if err == nil {
		t.Fatal("blocked Oh My Pi preview unexpectedly succeeded")
	}
	var report render.Result
	if err := json.Unmarshal([]byte(stdout), &report); err != nil {
		t.Fatalf("stdout is not render JSON: %v\n%s", err, stdout)
	}
	if report.Target != ohmypi.TargetName || report.AdapterVersion != ohmypi.AdapterVersion || !report.Preview || report.Applicable {
		t.Fatalf("unexpected Oh My Pi report: %#v", report)
	}
	if !strings.Contains(stderr, "ohmypi.config.inspector_unsafe") {
		t.Fatalf("support blocker missing: %s", stderr)
	}
	assertRenderFile(t, out, "render.json")
	assertRenderFile(t, out, "preview/route-only.config.yml.preview")
	assertRenderFile(t, out, "resources/instructions/system.md")
	config, readErr := os.ReadFile(filepath.Join(out, "preview", "route-only.config.yml.preview"))
	if readErr != nil {
		t.Fatal(readErr)
	}
	if strings.Contains(string(config), "oauth") || strings.Contains(string(config), "secret") {
		t.Fatalf("candidate config leaked authentication data: %s", config)
	}
}

func TestRenderOpenClawPreviewUsesTargetDispatch(t *testing.T) {
	profiles, resources, bindings := writeRenderFixture(t, false)
	out := filepath.Join(t.TempDir(), "candidate")
	stdout, stderr, err := executeRenderForTest(t, []string{
		"route-only", "--profiles", profiles, "--resource-root", resources, "--bindings", bindings,
		"--target", openclaw.TargetName, "--target-version", openclaw.TargetVersion, "--out", out, "--preview", "--json",
	})
	if err == nil {
		t.Fatal("blocked OpenClaw preview unexpectedly succeeded")
	}
	var report render.Result
	if err := json.Unmarshal([]byte(stdout), &report); err != nil {
		t.Fatalf("stdout is not render JSON: %v\n%s", err, stdout)
	}
	if report.Target != openclaw.TargetName || report.AdapterVersion != openclaw.AdapterVersion || !report.Preview || report.Applicable {
		t.Fatalf("unexpected OpenClaw report: %#v", report)
	}
	if !strings.Contains(stderr, "openclaw.config.inspector_unsafe") {
		t.Fatalf("OpenClaw support blocker missing: %s", stderr)
	}
	assertRenderFile(t, out, "render.json")
	assertRenderFile(t, out, "preview/route-only.config.json5.preview")
	assertRenderFile(t, out, "resources/instructions/system.md")
	config, readErr := os.ReadFile(filepath.Join(out, "preview", "route-only.config.json5.preview"))
	if readErr != nil {
		t.Fatal(readErr)
	}
	if strings.Contains(string(config), "oauth") || strings.Contains(string(config), "credential") || strings.Contains(string(config), "secret") {
		t.Fatalf("candidate config leaked authentication data: %s", config)
	}
}

func TestRenderHermesPreviewUsesTargetDispatch(t *testing.T) {
	profiles, resources, bindings := writeRenderFixture(t, false)
	out := filepath.Join(t.TempDir(), "candidate")
	stdout, stderr, err := executeRenderForTest(t, []string{
		"route-only", "--profiles", profiles, "--resource-root", resources, "--bindings", bindings,
		"--target", hermes.TargetName, "--target-version", hermes.TargetVersion, "--out", out, "--preview", "--json",
	})
	if err == nil {
		t.Fatal("blocked Hermes preview unexpectedly succeeded")
	}
	var report render.Result
	if err := json.Unmarshal([]byte(stdout), &report); err != nil {
		t.Fatalf("stdout is not render JSON: %v\n%s", err, stdout)
	}
	if report.Target != hermes.TargetName || report.AdapterVersion != hermes.AdapterVersion || !report.Preview || report.Applicable {
		t.Fatalf("unexpected Hermes report: %#v", report)
	}
	if !strings.Contains(stderr, "hermes.config.inspector_unsafe") {
		t.Fatalf("Hermes support blocker missing: %s", stderr)
	}
	assertRenderFile(t, out, "render.json")
	assertRenderFile(t, out, "preview/route-only.config.yaml.preview")
	assertRenderFile(t, out, "resources/instructions/system.md")
	config, readErr := os.ReadFile(filepath.Join(out, "preview", "route-only.config.yaml.preview"))
	if readErr != nil {
		t.Fatal(readErr)
	}
	if strings.Contains(string(config), "oauth") || strings.Contains(string(config), "credential") || strings.Contains(string(config), "secret") {
		t.Fatalf("candidate config leaked authentication data: %s", config)
	}
}

func TestRenderPiPreviewUsesTargetDispatch(t *testing.T) {
	profiles, resources, bindings := writeRenderFixture(t, false)
	out := filepath.Join(t.TempDir(), "candidate")
	stdout, stderr, err := executeRenderForTest(t, []string{
		"route-only", "--profiles", profiles, "--resource-root", resources, "--bindings", bindings,
		"--target", pi.TargetName, "--target-version", pi.TargetVersion, "--out", out, "--preview", "--json",
	})
	if err == nil {
		t.Fatal("blocked Pi preview unexpectedly succeeded")
	}
	var report render.Result
	if err := json.Unmarshal([]byte(stdout), &report); err != nil {
		t.Fatalf("stdout is not render JSON: %v\n%s", err, stdout)
	}
	if report.Target != pi.TargetName || report.AdapterVersion != pi.AdapterVersion || !report.Preview || report.Applicable {
		t.Fatalf("unexpected Pi report: %#v", report)
	}
	if !strings.Contains(stderr, "pi.config.acceptance_unverified") {
		t.Fatalf("Pi support blocker missing: %s", stderr)
	}
	assertRenderFile(t, out, "render.json")
	assertRenderFile(t, out, "preview/route-only.settings.json.preview")
	assertRenderFile(t, out, "resources/instructions/system.md")
	config, readErr := os.ReadFile(filepath.Join(out, "preview", "route-only.settings.json.preview"))
	if readErr != nil {
		t.Fatal(readErr)
	}
	if strings.Contains(string(config), "oauth") || strings.Contains(string(config), "secret") || strings.Contains(string(config), "authentication") {
		t.Fatalf("candidate config leaked authentication data: %s", config)
	}
}

func TestRenderUnknownTargetFailsClosedWithoutReadingInputs(t *testing.T) {
	out := filepath.Join(t.TempDir(), "candidate")
	stdout, stderr, err := executeRenderForTest(t, []string{
		"missing", "--profiles", filepath.Join(t.TempDir(), "missing-profiles"),
		"--resource-root", filepath.Join(t.TempDir(), "missing-resources"),
		"--bindings", filepath.Join(t.TempDir(), "missing-bindings.yaml"),
		"--target", "unknown", "--target-version", "1.0.0", "--out", out, "--preview", "--json",
	})
	if err == nil {
		t.Fatal("unknown target unexpectedly succeeded")
	}
	var report render.Result
	if err := json.Unmarshal([]byte(stdout), &report); err != nil {
		t.Fatalf("stdout is not render JSON: %v\n%s", err, stdout)
	}
	if len(report.Artifacts) != 0 || strings.Contains(stdout, "codex") || strings.Contains(stdout, "ohmypi") {
		t.Fatalf("unknown target received target-specific output: %#v\n%s", report, stdout)
	}
	if !strings.Contains(stderr, "render.target.unsupported") {
		t.Fatalf("unknown target diagnostic missing: %s", stderr)
	}
	if _, statErr := os.Stat(out); !os.IsNotExist(statErr) {
		t.Fatalf("unknown target created output: %v", statErr)
	}
}

func TestRenderHelpListsKnownTargets(t *testing.T) {
	cmd := newRenderCmd()
	flag := cmd.Flag("target")
	if flag == nil || !strings.Contains(flag.Usage, "claude-code, codex, pi, oh-my-pi, openclaw, hermes, or opencode") {
		t.Fatalf("target help does not list explicit adapters: %#v", flag)
	}
}

// TestRenderHelpHidesArielJcode ensures the experimental jcode-fork target is
// absent from the public --target help text and command Long description
// while remaining an accepted --target value (see
// TestRenderArielJcodePreviewUsesExperimentalDispatch).
func TestRenderHelpHidesArielJcode(t *testing.T) {
	cmd := newRenderCmd()
	flag := cmd.Flag("target")
	if flag == nil || strings.Contains(flag.Usage, "jcode-fork") {
		t.Fatalf("target help still mentions jcode-fork: %#v", flag)
	}
	if strings.Contains(cmd.Long, "jcode-fork") || strings.Contains(cmd.Short, "jcode-fork") {
		t.Fatalf("render help text still mentions jcode-fork: short=%q long=%q", cmd.Short, cmd.Long)
	}
}

// TestRenderPreviewAliasMatchesRenderCommand confirms "preview" is a working
// alias for "render" through the real rootCmd dispatch path.
func TestRenderPreviewAliasMatchesRenderCommand(t *testing.T) {
	found, _, err := rootCmd.Find([]string{"preview"})
	if err != nil {
		t.Fatalf("find preview alias: %v", err)
	}
	if found.Name() != "render" {
		t.Fatalf("preview alias resolved to %s, want render", found.Name())
	}

	profiles, resources, bindings := writeRenderFixture(t, false)
	out := filepath.Join(t.TempDir(), "candidate")
	homePathOverride = ""
	var stdout, stderr bytes.Buffer
	rootCmd.SetOut(&stdout)
	rootCmd.SetErr(&stderr)
	resetHomeFlag(t)
	resetNonInteractiveFlag(t)
	resetSubcommandFlags(rootCmd)
	rootCmd.SetArgs([]string{"preview", "route-only",
		"--profiles", profiles, "--resource-root", resources, "--bindings", bindings,
		"--target", claudecode.TargetName, "--target-version", claudecode.TargetVersion,
		"--out", out, "--preview", "--json",
	})
	err = rootCmd.Execute()
	if err == nil {
		t.Fatal("blocked Claude Code preview via alias unexpectedly succeeded")
	}
	var report render.Result
	if jsonErr := json.Unmarshal(stdout.Bytes(), &report); jsonErr != nil {
		t.Fatalf("stdout is not render JSON: %v\n%s", jsonErr, stdout.String())
	}
	if report.Target != claudecode.TargetName || !report.Preview || report.Applicable {
		t.Fatalf("unexpected report via preview alias: %#v", report)
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

func TestRenderRequiresTargetFlagsWhenUsingHomeDefaults(t *testing.T) {
	t.Setenv(profilehome.EnvHome, t.TempDir())
	cmd := newRenderCmd()
	cmd.SetArgs([]string{"route-only"})
	if err := cmd.Execute(); err == nil || !strings.Contains(err.Error(), "--target is required") {
		t.Fatalf("missing explicit flag error = %v", err)
	}
}

func TestRenderRejectsPartialRepositoryOverrides(t *testing.T) {
	cmd := newRenderCmd()
	cmd.SetArgs([]string{"route-only", "--profiles", t.TempDir()})
	if err := cmd.Execute(); err == nil || !strings.Contains(err.Error(), "set --profiles, --resource-root, and --bindings together") {
		t.Fatalf("partial repository override error = %v", err)
	}
}

func TestRenderUsesHomeRepositoryDefaults(t *testing.T) {
	home := t.TempDir()
	t.Setenv(profilehome.EnvHome, home)
	writeFile(t, filepath.Join(home, "profiles", "route-only", "profile.yaml"), "apiVersion: profilemango.dev/v1alpha1\nkind: PolicyProfile\nmetadata:\n  name: route-only\nspec:\n  routeRef: codex-oauth\n  instructions:\n    append: [instructions/system.md]\n")
	writeFile(t, filepath.Join(home, "instructions", "system.md"), "system\n")
	writeFile(t, filepath.Join(home, "bindings", "local.yaml"), "routes:\n  codex-oauth:\n    provider: openai\n    transport: native\n    authentication: oauth\n    model: gpt-5.6\n    effort: high\n")
	out := filepath.Join(t.TempDir(), "candidate")

	_, _, err := executeRenderForTest(t, []string{
		"route-only", "--target", codex.TargetName, "--target-version", codex.TargetVersion,
		"--out", out, "--preview", "--json",
	})
	if err == nil {
		t.Fatal("blocked home preview unexpectedly succeeded")
	}
	assertRenderFile(t, out, "preview/route-only.config.toml.preview")
	assertRenderFile(t, out, "resources/instructions/system.md")
}

func TestRenderMissingHomeBindingsIsActionable(t *testing.T) {
	home := t.TempDir()
	t.Setenv(profilehome.EnvHome, home)
	writeFile(t, filepath.Join(home, "profiles", "default", "profile.yaml"), "apiVersion: profilemango.dev/v1alpha1\nkind: PolicyProfile\nmetadata:\n  name: default\nspec:\n  routeRef: local\n")
	_, _, err := executeRenderForTest(t, []string{
		"default", "--target", codex.TargetName, "--target-version", codex.TargetVersion,
		"--out", filepath.Join(t.TempDir(), "candidate"),
	})
	if err == nil || !strings.Contains(err.Error(), "copy") || !strings.Contains(err.Error(), "local.example.yaml") || !strings.Contains(err.Error(), "mango init") {
		t.Fatalf("missing home bindings error = %v", err)
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
	homePathOverride = ""
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
