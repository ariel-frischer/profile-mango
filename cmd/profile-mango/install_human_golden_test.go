package main

import (
	"bytes"
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"gitlab.com/ariel-frischer/profile-mango/pkg/install"
	"gitlab.com/ariel-frischer/profile-mango/pkg/profilemango"
)

func TestCompactInstallGolden(t *testing.T) {
	plan := install.Plan{PlanID: "example-id", Status: install.StatusReady, Targets: []install.TargetPlan{
		{Target: install.Target{Name: "zeta", Version: "1"}, Status: install.StatusBlocked, Reason: "not qualified"},
		{Target: install.Target{Name: "codex", Version: "0.154.0"}, Status: install.StatusReady, Reason: "long adapter evidence caveat", Config: &install.ConfigDestination{Path: "/sandbox/config.toml"}, Install: &install.InstallMode{Mode: install.InstallModeNamedProfile, ProfileName: "demo", UseCommand: "codex --profile demo"}, Fields: []install.FieldChange{{Path: "demo.config.model", After: "gpt-test"}, {Path: "demo.config.model_reasoning_effort", After: "high"}, {Path: "config.secret", Before: "secret", After: "changed", Sensitive: true}}, Files: []install.FilePlan{{Path: "/sandbox/demo.toml", Action: install.ActionCreate}}, SkippedRequirements: []install.SkippedRequirement{{Requirement: "permissions"}, {Requirement: "tools"}, {Requirement: "instructions", Count: 2}, {Requirement: "skills", Count: 1}}, Diagnostics: profilemango.Diagnostics{{Severity: profilemango.SeverityWarning, Code: "codex.install.auth_unmanaged", Message: "Authentication remains target-owned; check it locally."}}},
	}}
	var output bytes.Buffer
	if err := writeInstallPlan(commandOutput(&output), plan, false, false); err != nil {
		t.Fatal(err)
	}
	const golden = "plan example-id (ready)\n  codex@0.154.0: ready | destination: /sandbox/demo.config.toml | use it: codex --profile demo\n    route: model gpt-test, effort high | changes: secret \"<redacted>\" -> \"<redacted>\", model \"\" -> \"gpt-test\", effort \"\" -> \"high\"\n    files: /sandbox/demo.toml create\n    not installed for this agent: permissions, tools, instructions (2 files), skills (1)\n  zeta@1: blocked (not qualified)\nSummary: 1 ready, 0 unchanged, 1 blocked, 0 conflict, 0 skipped; files: 1 create, 0 update, 0 unchanged. Unrelated target settings are preserved.\n"
	if output.String() != golden {
		t.Fatalf("compact golden mismatch:\n%s", output.String())
	}
	for _, hidden := range []string{"demo.config.model", "config.secret", "long adapter evidence caveat", "Authentication remains target-owned"} {
		if strings.Contains(output.String(), hidden) {
			t.Fatalf("compact plan leaked detail %q: %s", hidden, output.String())
		}
	}
	var verbose bytes.Buffer
	if err := writeInstallPlan(commandOutput(&verbose), plan, false, true); err != nil {
		t.Fatal(err)
	}
	for _, detail := range []string{"demo.config.model", "long adapter evidence caveat", "Authentication remains target-owned"} {
		if !strings.Contains(verbose.String(), detail) {
			t.Fatalf("verbose plan omitted %q: %s", detail, verbose.String())
		}
	}
	if strings.Contains(output.String(), "\"secret\"") || strings.Contains(output.String(), "\"changed\"") {
		t.Fatal("sensitive value leaked")
	}
	var encoded bytes.Buffer
	if err := writeInstallPlan(commandOutput(&encoded), plan, true, false); err != nil {
		t.Fatal(err)
	}
	expected, err := plan.JSON()
	if err != nil {
		t.Fatal(err)
	}
	if encoded.String() != string(expected) || plan.PlanID != "example-id" {
		t.Fatal("JSON output or plan ID changed")
	}
}

func TestCompactUndoGoldenAndVerbose(t *testing.T) {
	plan := install.RestorePlan{PlanID: "undo-id", OriginalPlanID: "install-id", Target: "codex@0.154.0", Status: "ready", Files: []install.RestoreFile{{Path: "/sandbox/config.toml", Action: "update", BeforeSHA256: "old", AfterSHA256: "new", Diff: "--- before\n+++ after\n@@ -1 +1 @@\n-old\n+new\n"}}}
	var output bytes.Buffer
	if err := writeRestorePlan(commandOutput(&output), plan, false, false); err != nil {
		t.Fatal(err)
	}
	const golden = "undo plan undo-id (ready), original install install-id\n  target: codex@0.154.0\n  update /sandbox/config.toml\n    --- before\n    +++ after\n    @@ -1 +1 @@\n    -old\n    +new\n"
	if output.String() != golden {
		t.Fatalf("undo golden mismatch:\n%s", output.String())
	}
	output.Reset()
	if err := writeRestorePlan(commandOutput(&output), plan, false, true); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(output.String(), "old -> new") {
		t.Fatalf("verbose hashes missing: %s", output.String())
	}
}

func TestCompactWarningsAndUndoHint(t *testing.T) {
	plan := install.Plan{PlanID: "id", Status: install.StatusReady, Targets: []install.TargetPlan{{
		Target: install.Target{Name: "openclaw", Version: "2026.9.5"}, Status: install.StatusReady,
		VersionCheck: &install.VersionCheck{Status: install.VersionOutOfRange, Detected: "2026.10", Range: ">=2026.9 <2026.10"},
		Diagnostics:  profilemango.Diagnostics{{Severity: profilemango.SeverityWarning, Code: "openclaw.install.profile_state_separate", Message: "state is separate"}, {Severity: profilemango.SeverityWarning, Code: "install.version_out_of_range", Message: "version mismatch"}},
	}}}
	var output bytes.Buffer
	if err := writeInstallPlan(commandOutput(&output), plan, false, false); err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"outside tested range", "check agent compatibility", "sign in there if needed", "authentication was not inspected"} {
		if !strings.Contains(output.String(), want) {
			t.Fatalf("missing %q: %s", want, output.String())
		}
	}
	if strings.Contains(output.String(), "version mismatch") || strings.Contains(output.String(), "\x1b[") {
		t.Fatalf("duplicate warning or ANSI in non-TTY output: %s", output.String())
	}
	output.Reset()
	undo := install.RestorePlan{Target: "codex@0.154.0", PlanID: "undo-id"}
	options := restoreOptions{config: "/sandbox/a b.toml", originalPlan: "original"}
	if err := writeUndoHint(&output, undo, options); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(output.String(), "--config '/sandbox/a b.toml' --original-plan original --apply --yes --expect-plan undo-id") {
		t.Fatalf("undo hint = %q", output.String())
	}
}

func TestCompactRouteUsesInstalledFieldsOnly(t *testing.T) {
	tests := map[string]struct{ path, want string }{
		"pi":       {"config.defaultModel", "route: model gpt-test, effort not installed"},
		"opencode": {"agent.model", "route: model gpt-test, effort not installed"},
		"openclaw": {"demo.config.agents.defaults.model.primary", "route: model gpt-test, effort not installed"},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			var output bytes.Buffer
			target := install.TargetPlan{Install: &install.InstallMode{ProfileName: "demo"}, Fields: []install.FieldChange{{Path: test.path, After: "gpt-test"}}}
			if err := writeCompactRoute(&output, target); err != nil {
				t.Fatal(err)
			}
			if !strings.Contains(output.String(), test.want) {
				t.Fatalf("route = %q", output.String())
			}
		})
	}
}

func TestCompactSkippedListsRoleRequirements(t *testing.T) {
	var output bytes.Buffer
	skipped := []install.SkippedRequirement{
		{Requirement: install.RequirementRoleDefinitions, Count: 2},
		{Requirement: install.RequirementRoles, Count: 3},
		{Requirement: install.RequirementSubagentMaxEffort, Value: "high"},
	}
	if err := writeSkippedRequirements(&output, skipped); err != nil {
		t.Fatal(err)
	}
	if want := "    not installed for this agent: role-definitions (2), roles (3), subagentMaxEffort high\n"; output.String() != want {
		t.Fatalf("skipped = %q, want %q", output.String(), want)
	}
}

func TestCompactClaudeExplicitDestinationAndUse(t *testing.T) {
	target := install.TargetPlan{Target: install.Target{Name: "claude-code", Version: "2.1.278"}, Status: install.StatusReady,
		Config:  &install.ConfigDestination{Path: "/sandbox/agent home/settings.json"},
		Install: &install.InstallMode{Mode: install.InstallModeNamedProfile, ProfileName: "demo", UseCommand: "claude --settings ~/.claude/profiles/demo.json"}}
	var output bytes.Buffer
	if err := writeCompactTarget(&output, target, newOutputStyles(false)); err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"destination: /sandbox/agent home/profiles/demo.json", "use it: claude --settings '/sandbox/agent home/profiles/demo.json'"} {
		if !strings.Contains(output.String(), want) {
			t.Fatalf("missing %q in %s", want, output.String())
		}
	}
}

func TestCompactConflictGolden(t *testing.T) {
	plan := install.Plan{PlanID: "conflict-id", Status: install.StatusBlocked, Targets: []install.TargetPlan{{
		Target: install.Target{Name: "opencode", Version: "1.18.31"}, Status: install.StatusConflict,
		Reason:      "owned profile was edited; review drift before using --override",
		Diagnostics: profilemango.Diagnostics{{Severity: profilemango.SeverityWarning, Code: "opencode.install.evidence", Message: "long adapter evidence caveat"}},
	}}}
	var output bytes.Buffer
	if err := writeInstallPlan(commandOutput(&output), plan, false, false); err != nil {
		t.Fatal(err)
	}
	const golden = "plan conflict-id (blocked)\n  opencode@1.18.31: conflict (owned profile was edited; review drift before using --override)\nSummary: 0 ready, 0 unchanged, 0 blocked, 1 conflict, 0 skipped; files: 0 create, 0 update, 0 unchanged. Unrelated target settings are preserved.\n"
	if output.String() != golden {
		t.Fatalf("conflict golden mismatch: %s", output.String())
	}
	if strings.Contains(output.String(), "long adapter evidence caveat") {
		t.Fatalf("generic warning leaked: %s", output.String())
	}
	t.Setenv("NO_COLOR", "")
	t.Setenv("TERM", "xterm")
	if !strings.Contains(styledPlanStatus(install.StatusConflict, newOutputStyles(true)), "\x1b[") {
		t.Fatal("conflict status is not failure-colored")
	}
}

func TestTypicalReadyTargetFitsThreeLines(t *testing.T) {
	plan := install.Plan{PlanID: "ready-id", Status: install.StatusReady, Targets: []install.TargetPlan{{
		Target: install.Target{Name: "codex", Version: "0.154.0"}, Status: install.StatusReady,
		Config:  &install.ConfigDestination{Path: "/sandbox/config.toml"},
		Install: &install.InstallMode{Mode: install.InstallModeNamedProfile, ProfileName: "demo", UseCommand: "codex --profile demo"},
		Fields:  []install.FieldChange{{Path: "demo.config.model", After: "gpt-test"}, {Path: "demo.config.model_reasoning_effort", After: "high"}},
		Files:   []install.FilePlan{{Path: "demo.config.toml", Action: install.ActionCreate}, {Path: "config.toml.profile-mango.manifest.json", Action: install.ActionCreate}},
	}}}
	var output bytes.Buffer
	if err := writeInstallPlan(commandOutput(&output), plan, false, false); err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(strings.TrimSpace(output.String()), "\n")
	if len(lines) != 5 {
		t.Fatalf("want plan header, three target lines and summary, got %d:\n%s", len(lines), output.String())
	}
	for _, want := range []string{"ready | destination: /sandbox/demo.config.toml | use it: codex --profile demo", "route: model gpt-test, effort high | changes:", "files: config.toml.profile-mango.manifest.json create, demo.config.toml create"} {
		if !strings.Contains(output.String(), want) {
			t.Fatalf("missing %q:\n%s", want, output.String())
		}
	}
}

func commandOutput(output *bytes.Buffer) *cobra.Command {
	cmd := &cobra.Command{}
	cmd.SetOut(output)
	return cmd
}
