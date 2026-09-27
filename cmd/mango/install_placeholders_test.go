package main

import (
	"bytes"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ariel-frischer/profile-mango/pkg/install"
	"github.com/spf13/cobra"
)

const placeholderBindings = "routes:\n  route:\n    provider: openai\n    transport: native\n    authentication: oauth\n    model: gpt-5.6\n    effort: high\n" +
	"    targets:\n      codex:\n        model: gpt-5.6-codex\n    roles:\n      planner: {provider: anthropic, model: opus, effort: high}\n"

// TestInstallRendersRoutePlaceholdersPerTarget pins that global instruction files
// resolve {{route.…}} from each agent's effective route, that status compares the
// rendered bytes (clean after install, drift once a binding it names changes), and that a
// placeholder naming an unknown route fails before anything is planned.
func TestInstallRendersRoutePlaceholdersPerTarget(t *testing.T) {
	env := newUseTestHome(t)
	writeFile(t, env.options.bindings, placeholderBindings)
	text := "Main model {{route.route.model}} ({{route.route.provider}}); planner {{route.route.roles.planner.model}}; keep {{ literal }}.\n"
	writeFile(t, filepath.Join(env.root, "global", "work-codex.md"), text)
	writeFile(t, filepath.Join(env.root, "global", "work-omp.md"), text)
	options := env.options
	options.targets, options.makeDefault = []string{"codex", "oh-my-pi"}, true
	env.planThenApply(t, "work", options)

	assertFileContent(t, filepath.Join(env.codexDir, "AGENTS.md"), "Main model gpt-5.6-codex (openai); planner opus; keep {{ literal }}.\n")
	assertFileContent(t, filepath.Join(env.ompDir, "AGENTS.md"), "Main model gpt-5.6 (openai); planner opus; keep {{ literal }}.\n")
	report := statusForTest(t, env.options)
	for _, name := range []string{"codex", "oh-my-pi"} {
		if target := targetStatus(t, report, name); target.Source != install.SourceCurrent || len(target.Drift) != 0 {
			t.Fatalf("%s after install: source %s (%s), drift %v", name, target.Source, target.SourceReason, target.Drift)
		}
	}

	replaceInFile(t, env.options.bindings, "model: opus,", "model: opus-6,")
	if codex := targetStatus(t, statusForTest(t, env.options), "codex"); codex.Source != install.SourceChanged || len(codex.Drift) != 1 || codex.Drift[0].Path != "AGENTS.md" {
		t.Fatalf("codex after binding change: source %s, drift %v", codex.Source, codex.Drift)
	}

	writeFile(t, filepath.Join(env.root, "global", "work-omp.md"), "{{route.nope.model}}\n")
	var output bytes.Buffer
	cmd := &cobra.Command{}
	cmd.SetOut(&output)
	err := runInstall(cmd, "work", options)
	if err == nil || !strings.Contains(err.Error(), `globalInstructions.oh-my-pi.AGENTS.md: global/work-omp.md: line 1: {{route.nope.model}}: route "nope" is not in the bindings file`) {
		t.Fatalf("install with unknown placeholder: %v\n%s", err, output.String())
	}
}
