package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/ariel-frischer/profile-mango/pkg/install"
	"github.com/spf13/cobra"
)

const useTestHandRules = "# my own omp rules\n"

// useTestHome is a sandbox HOME with Codex and Oh My Pi folders, a hand-written omp
// RULES.md, and two profiles: work owns global files for both agents, personal only
// omp's AGENTS.md.
type useTestHome struct {
	root, home, codexDir, ompDir string
	options                      installOptions
}

func newUseTestHome(t *testing.T) useTestHome {
	t.Helper()
	root := t.TempDir()
	env := useTestHome{root: root, home: filepath.Join(root, "home")}
	t.Setenv("HOME", env.home)
	t.Setenv("CODEX_HOME", "")
	env.codexDir, env.ompDir = filepath.Join(env.home, ".codex"), filepath.Join(env.home, ".omp", "agent")
	writeFile(t, filepath.Join(env.ompDir, "RULES.md"), useTestHandRules)
	profiles := filepath.Join(root, "profiles")
	writeFile(t, filepath.Join(profiles, "work", "profile.yaml"), "route: route\nglobalInstructions:\n  codex:\n    AGENTS.md: global/work-codex.md\n  oh-my-pi:\n    AGENTS.md: global/work-omp.md\n    RULES.md: global/work-rules.md\n")
	writeFile(t, filepath.Join(profiles, "personal", "profile.yaml"), "route: route\nglobalInstructions:\n  oh-my-pi:\n    AGENTS.md: global/personal-omp.md\n")
	for name, content := range map[string]string{"work-codex.md": "# work codex\n", "work-omp.md": "# work omp\n", "work-rules.md": "# work rules\n", "personal-omp.md": "# personal omp\n"} {
		writeFile(t, filepath.Join(root, "global", name), content)
	}
	bindings := filepath.Join(root, "bindings.yaml")
	writeFile(t, bindings, "routes:\n  route:\n    provider: openai\n    transport: native\n    authentication: oauth\n    model: gpt-5.6\n    effort: high\n")
	if err := os.MkdirAll(env.codexDir, 0o700); err != nil {
		t.Fatal(err)
	}
	env.options = installOptions{profiles: profiles, resourceRoot: root, bindings: bindings}
	return env
}

// planThenApply runs the plan, checks its apply hint names the subcommand, then applies it.
func (env useTestHome) planThenApply(t *testing.T, profile string, options installOptions) string {
	t.Helper()
	output := runInstallProfileForTest(t, profile, options)
	planID := regexp.MustCompile(`plan ([0-9a-f]{64}) \(ready\)`).FindStringSubmatch(output)
	if planID == nil || !strings.Contains(output, "mango "+options.command()+" "+profile+" ") {
		t.Fatalf("%s plan is not ready or lacks its apply hint:\n%s", options.command(), output)
	}
	options.apply, options.yes, options.expectPlan = true, true, planID[1]
	return runInstallProfileForTest(t, profile, options)
}

func runInstallProfileForTest(t *testing.T, profile string, options installOptions) string {
	t.Helper()
	var output bytes.Buffer
	cmd := &cobra.Command{}
	cmd.SetOut(&output)
	cmd.SetErr(&output)
	if err := runInstall(cmd, profile, options); err != nil {
		t.Fatalf("%s %s: %v\n%s", options.command(), profile, err, output.String())
	}
	return output.String()
}

func statusForTest(t *testing.T, options installOptions) install.StatusReport {
	t.Helper()
	var output bytes.Buffer
	cmd := &cobra.Command{}
	cmd.SetOut(&output)
	options.jsonOutput = true
	if err := runStatus(cmd, options); err != nil {
		t.Fatalf("status: %v\n%s", err, output.String())
	}
	if strings.Contains(output.String(), os.Getenv("HOME")) {
		t.Fatalf("status JSON leaks an absolute path:\n%s", output.String())
	}
	var report install.StatusReport
	if err := json.Unmarshal(output.Bytes(), &report); err != nil {
		t.Fatal(err)
	}
	return report
}

func targetStatus(t *testing.T, report install.StatusReport, name string) install.TargetStatus {
	t.Helper()
	for _, target := range report.Targets {
		if target.Target.Name == name {
			return target
		}
	}
	t.Fatalf("status has no %s target", name)
	return install.TargetStatus{}
}

func fileState(target install.TargetStatus, path string) string {
	for _, file := range target.Files {
		if file.Path == path {
			return file.State
		}
	}
	return ""
}

func TestUseSwitchesManagedAgentsAndReleasesGlobalFiles(t *testing.T) {
	env := newUseTestHome(t)
	installWork := env.options
	installWork.targets, installWork.makeDefault = []string{"codex,oh-my-pi"}, true
	env.planThenApply(t, "work", installWork)
	assertFileContent(t, filepath.Join(env.codexDir, "AGENTS.md"), "# work codex\n")
	assertFileContent(t, filepath.Join(env.ompDir, "RULES.md"), "# work rules\n")

	report := statusForTest(t, env.options)
	for _, name := range []string{"codex", "oh-my-pi"} {
		target := targetStatus(t, report, name)
		if target.State != install.StatusManaged || target.Profile != "work" || target.Source != install.SourceCurrent || fileState(target, "AGENTS.md") != install.FileInSync {
			t.Fatalf("%s status after install = %#v", name, target)
		}
	}
	if state := targetStatus(t, report, "claude-code").State; state != install.StatusUnmanaged {
		t.Fatalf("claude-code state = %s", state)
	}

	writeFile(t, filepath.Join(env.root, "global", "work-omp.md"), "# work omp v2\n")
	if source := targetStatus(t, statusForTest(t, env.options), "oh-my-pi").Source; source != install.SourceChanged {
		t.Fatalf("omp source after resource edit = %s", source)
	}
	writeFile(t, filepath.Join(env.root, "global", "work-omp.md"), "# work omp\n")

	env.planThenApply(t, "personal", env.useOptions())
	if _, err := os.Stat(filepath.Join(env.codexDir, "AGENTS.md")); !os.IsNotExist(err) {
		t.Fatalf("created codex AGENTS.md survived the switch: %v", err)
	}
	assertFileContent(t, filepath.Join(env.ompDir, "RULES.md"), useTestHandRules)
	assertFileContent(t, filepath.Join(env.ompDir, "AGENTS.md"), "# personal omp\n")
	report = statusForTest(t, env.options)
	omp := targetStatus(t, report, "oh-my-pi")
	if omp.Profile != "personal" || fileState(omp, "RULES.md") != "" || targetStatus(t, report, "codex").Profile != "personal" {
		t.Fatalf("status after use = %#v", report.Targets)
	}
}

// A list-form global file installs through use as the composed fragments, reads as
// in sync, and reads as sources changed after an edit to any one fragment.
func TestUseComposedGlobalFileTracksEveryFragment(t *testing.T) {
	tests := map[string]struct{ edited string }{
		"first fragment edited": {edited: "shared-core.md"},
		"last fragment edited":  {edited: "omp-extra.md"},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			env := newUseTestHome(t)
			writeFile(t, filepath.Join(env.options.profiles, "composed", "profile.yaml"), "route: route\nglobalInstructions:\n  oh-my-pi:\n    AGENTS.md: [global/shared-core.md, global/omp-extra.md]\n")
			writeFile(t, filepath.Join(env.root, "global", "shared-core.md"), "# core\n")
			writeFile(t, filepath.Join(env.root, "global", "omp-extra.md"), "# omp {{route.route.model}}\n")
			options := env.useOptions()
			options.targets = []string{"oh-my-pi"}
			env.planThenApply(t, "composed", options)
			assertFileContent(t, filepath.Join(env.ompDir, "AGENTS.md"), "# core\n\n# omp gpt-5.6\n")
			omp := targetStatus(t, statusForTest(t, env.options), "oh-my-pi")
			if omp.Profile != "composed" || omp.Source != install.SourceCurrent || fileState(omp, "AGENTS.md") != install.FileInSync {
				t.Fatalf("status after use = %#v", omp)
			}
			writeFile(t, filepath.Join(env.root, "global", test.edited), "# edited\n")
			if source := targetStatus(t, statusForTest(t, env.options), "oh-my-pi").Source; source != install.SourceChanged {
				t.Fatalf("source after editing %s = %s", test.edited, source)
			}
		})
	}
}

// Re-planning an in-sync install after only a global instruction edit lists no route
// changes: the named overlay and config.yml keep their values, so nothing reads as
// being written from empty.
func TestReplanInSyncShowsOnlyInstructionUpdate(t *testing.T) {
	env := newUseTestHome(t)
	options := env.options
	options.targets, options.makeDefault = []string{"oh-my-pi"}, true
	env.planThenApply(t, "work", options)
	writeFile(t, filepath.Join(env.root, "global", "work-omp.md"), "# work omp v2\n")

	output := runInstallProfileForTest(t, "work", options)
	output = regexp.MustCompile(`[0-9a-f]{64}`).ReplaceAllString(strings.ReplaceAll(output, env.home, "~"), "<id>")
	const golden = "plan <id> (ready)\n" +
		"  oh-my-pi@18.6.0: ready | destination: ~/.omp/agent/profiles/work.yml | also the default: ~/.omp/agent/config.yml | use it: omp --config ~/.omp/agent/profiles/work.yml\n" +
		"    route: model openai/gpt-5.6, effort high\n" +
		"    files: AGENTS.md update, RULES.md noop, config.yml noop, config.yml.profile-mango.manifest.json update, work.yml noop\n" +
		"Summary: 1 ready, 0 unchanged, 0 blocked, 0 conflict, 0 skipped; files: 0 create, 2 update, 3 unchanged. Unrelated target settings are preserved.\n"
	if !strings.HasPrefix(output, golden) {
		t.Fatalf("re-plan golden mismatch:\n%s", output)
	}
}

// Without a terminal, --apply succeeds only when the plan writes nothing.
func TestUseApplyWithoutTerminalNeedsConsentOnlyForChanges(t *testing.T) {
	env := newUseTestHome(t)
	options := env.options
	options.targets, options.makeDefault, options.apply = []string{"oh-my-pi"}, true, true
	tests := map[string]struct {
		setup   func()
		wantErr string
	}{
		"ready plan": {wantErr: "interactive apply requires a terminal"},
		"unchanged plan": {setup: func() {
			planOnly := options
			planOnly.apply = false
			env.planThenApply(t, "work", planOnly)
		}},
	}
	for _, name := range []string{"ready plan", "unchanged plan"} {
		test := tests[name]
		if test.setup != nil {
			test.setup()
		}
		cmd := &cobra.Command{}
		var output bytes.Buffer
		cmd.SetOut(&output)
		cmd.SetErr(&output)
		cmd.SetIn(strings.NewReader("y\n"))
		err := runInstall(cmd, "work", options)
		if test.wantErr == "" && err != nil || test.wantErr != "" && (err == nil || !strings.Contains(err.Error(), test.wantErr)) {
			t.Fatalf("%s: err = %v, want %q\n%s", name, err, test.wantErr, output.String())
		}
		if strings.Contains(output.String(), "[y/N]") {
			t.Fatalf("%s prompted without a terminal:\n%s", name, output.String())
		}
	}
	assertFileContent(t, filepath.Join(env.ompDir, "RULES.md"), "# work rules\n")
}

func TestStatusReportsEditedAndMissingOwnedFiles(t *testing.T) {
	env := newUseTestHome(t)
	installWork := env.options
	installWork.targets, installWork.makeDefault = []string{"oh-my-pi"}, true
	env.planThenApply(t, "work", installWork)
	writeFile(t, filepath.Join(env.ompDir, "AGENTS.md"), "# edited by hand\n")
	if err := os.Remove(filepath.Join(env.ompDir, "RULES.md")); err != nil {
		t.Fatal(err)
	}
	omp := targetStatus(t, statusForTest(t, env.options), "oh-my-pi")
	if fileState(omp, "AGENTS.md") != install.FileEdited || fileState(omp, "RULES.md") != install.FileMissing {
		t.Fatalf("drifted status = %#v", omp.Files)
	}
	var output bytes.Buffer
	cmd := &cobra.Command{}
	cmd.SetOut(&output)
	cmd.SetErr(&output)
	if err := runInstall(cmd, "personal", env.useOptions()); err == nil {
		t.Fatalf("use over an edited global file planned without --override:\n%s", output.String())
	}
}

func TestUseWithoutManagedAgentsNeedsTargets(t *testing.T) {
	env := newUseTestHome(t)
	var output bytes.Buffer
	cmd := &cobra.Command{}
	cmd.SetOut(&output)
	err := runInstall(cmd, "work", env.useOptions())
	if err == nil || !strings.Contains(err.Error(), "no agent is managed") {
		t.Fatalf("use with nothing managed = %v", err)
	}
}

func TestUndoAfterUseRestoresEachGeneration(t *testing.T) {
	env := newUseTestHome(t)
	installWork := env.options
	installWork.targets, installWork.makeDefault = []string{"oh-my-pi"}, true
	env.planThenApply(t, "work", installWork)
	env.planThenApply(t, "personal", env.useOptions())
	for _, want := range []string{"# work rules\n", useTestHandRules} {
		preview, err := runUndoForTest(t, restoreOptions{target: "oh-my-pi"})
		planID := regexp.MustCompile(`undo plan ([0-9a-f]{64}) \(ready\)`).FindStringSubmatch(preview)
		if err != nil || planID == nil {
			t.Fatalf("undo preview: %v\n%s", err, preview)
		}
		if out, err := runUndoForTest(t, restoreOptions{target: "oh-my-pi", apply: true, yes: true, expectPlan: planID[1]}); err != nil {
			t.Fatalf("undo apply: %v\n%s", err, out)
		}
		assertFileContent(t, filepath.Join(env.ompDir, "RULES.md"), want)
	}
	if _, err := os.Stat(filepath.Join(env.ompDir, "AGENTS.md")); !os.IsNotExist(err) {
		t.Fatalf("AGENTS.md created by the first install survived both undos: %v", err)
	}
}

func TestUseSwitchesOhMyPiRolesAndReleasesDroppedOnes(t *testing.T) {
	env := newUseTestHome(t)
	config := filepath.Join(env.ompDir, "config.yml")
	writeFile(t, config, "modelRoles:\n  smol: user/fast:low\ntask:\n  maxEffort: max\n")
	writeFile(t, filepath.Join(env.options.profiles, "roles-a", "profile.yaml"), "route: a\n")
	writeFile(t, filepath.Join(env.options.profiles, "roles-b", "profile.yaml"), "route: b\n")
	route := "    provider: openai\n    transport: native\n    authentication: oauth\n    model: gpt-5.6\n    effort: high\n"
	writeFile(t, env.options.bindings, "routes:\n  a:\n"+route+"    subagentMaxEffort: high\n    roles:\n"+
		"      planner: {provider: anthropic, model: opus, effort: high}\n      research: {provider: openai, model: mini, effort: low}\n      tiny: {provider: openai, model: nano}\n"+
		"  b:\n"+route+"    roles:\n      planner: {provider: openai, model: gpt-5.6, effort: xhigh}\n")
	installA := env.options
	installA.targets, installA.makeDefault = []string{"oh-my-pi"}, true
	env.planThenApply(t, "roles-a", installA)
	assertFileContent(t, config, "modelRoles:\n  smol: \"openai/mini:low\"\n  default: \"openai/gpt-5.6:high\"\n  commit: \"openai/nano\"\n  plan: \"anthropic/opus:high\"\n  slow: \"anthropic/opus:high\"\n  tiny: \"openai/nano\"\ntask:\n  maxEffort: \"high\"\n"+useRolesPresetGolden)

	env.planThenApply(t, "roles-b", env.useOptions())
	assertFileContent(t, config, "modelRoles:\n  smol: \"user/fast:low\"\n  default: \"openai/gpt-5.6:high\"\n  plan: \"openai/gpt-5.6:xhigh\"\n  slow: \"openai/gpt-5.6:xhigh\"\ntask:\n  maxEffort: \"max\"\n"+useRolesPresetGolden)
	omp := targetStatus(t, statusForTest(t, env.options), "oh-my-pi")
	if omp.Profile != "roles-b" || omp.Source != install.SourceCurrent || fileState(omp, "config.yml") != install.FileInSync {
		t.Fatalf("status after use = %#v", omp)
	}
}

// A default install whose main config already holds the profile's values leaves that
// file unowned; status still re-checks it as a default install, so the owned global
// instruction files read as current rather than drifted.
func TestStatusKeepsDefaultWhenConfigAlreadyMatched(t *testing.T) {
	first := newUseTestHome(t)
	installWork := first.options
	installWork.targets, installWork.makeDefault = []string{"oh-my-pi"}, true
	first.planThenApply(t, "work", installWork)
	matched, err := os.ReadFile(filepath.Join(first.ompDir, "config.yml"))
	if err != nil {
		t.Fatal(err)
	}

	env := newUseTestHome(t)
	// Only copy the matching role config, not the unowned Mango-named preset:
	// preset adoption deliberately requires an explicit override.
	writeFile(t, filepath.Join(env.ompDir, "config.yml"), strings.Split(string(matched), "modelPresets:")[0])
	useWork := env.useOptions()
	useWork.targets = []string{"oh-my-pi"}
	env.planThenApply(t, "work", useWork)
	omp := targetStatus(t, statusForTest(t, env.options), "oh-my-pi")
	if fileState(omp, "config.yml") != install.FileInSync || omp.Source != install.SourceCurrent || len(omp.Drift) != 0 {
		t.Fatalf("status after use over a matching config = %s (%s), drift %v", omp.Source, omp.SourceReason, omp.Drift)
	}
}

// TestUseOwnsHomeAgentsMDThroughPi installs ~/AGENTS.md with the Pi target, reports it
// in status, gives the adopted original back on use, and undoes each generation.
func TestUseOwnsHomeAgentsMDThroughPi(t *testing.T) {
	env := newUseTestHome(t)
	homeAgents := filepath.Join(env.home, "AGENTS.md")
	writeFile(t, homeAgents, "# my home rules\n")
	if err := os.MkdirAll(filepath.Join(env.home, ".pi", "agent"), 0o700); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(env.root, "global", "work-home.md"), "# work home\n")
	writeFile(t, filepath.Join(env.options.profiles, "work", "profile.yaml"), "route: route\nglobalInstructions:\n  home:\n    AGENTS.md: global/work-home.md\n")
	installWork := env.options
	installWork.targets, installWork.makeDefault = []string{"codex,pi"}, true
	if output := env.planThenApply(t, "work", installWork); !strings.Contains(output, "~/AGENTS.md") {
		t.Fatalf("plan does not name ~/AGENTS.md:\n%s", output)
	}
	assertFileContent(t, homeAgents, "# work home\n")
	pi := targetStatus(t, statusForTest(t, env.options), "pi")
	if pi.Profile != "work" || pi.Source != install.SourceCurrent || fileState(pi, "~/AGENTS.md") != install.FileInSync {
		t.Fatalf("pi status after install = %#v", pi)
	}
	env.planThenApply(t, "personal", env.useOptions())
	assertFileContent(t, homeAgents, "# my home rules\n")
	if state := fileState(targetStatus(t, statusForTest(t, env.options), "pi"), "~/AGENTS.md"); state != "" {
		t.Fatalf("released ~/AGENTS.md still reported as %s", state)
	}
	for _, want := range []string{"# work home\n", "# my home rules\n"} {
		preview, err := runUndoForTest(t, restoreOptions{target: "pi"})
		planID := regexp.MustCompile(`undo plan ([0-9a-f]{64}) \(ready\)`).FindStringSubmatch(preview)
		if err != nil || planID == nil {
			t.Fatalf("undo preview: %v\n%s", err, preview)
		}
		if out, err := runUndoForTest(t, restoreOptions{target: "pi", apply: true, yes: true, expectPlan: planID[1]}); err != nil {
			t.Fatalf("undo apply: %v\n%s", err, out)
		}
		assertFileContent(t, homeAgents, want)
	}
}

func assertFileContent(t *testing.T, path, want string) {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil || string(data) != want {
		t.Fatalf("%s = %q, err=%v; want %q", path, data, err, want)
	}
}

func (env useTestHome) useOptions() installOptions {
	options := env.options
	options.use = true
	return options
}
