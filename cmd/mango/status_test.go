package main

import (
	"bytes"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/ariel-frischer/profile-mango/pkg/install"
	"github.com/spf13/cobra"
)

// newDriftTestHome installs the work profile into Oh My Pi's config.yml with a
// planner role and a subagent effort cap, so status owns modelRoles slots and
// task.maxEffort as well as the global instruction files.
func newDriftTestHome(t *testing.T) (useTestHome, string) {
	t.Helper()
	env := newUseTestHome(t)
	writeFile(t, env.options.bindings, "routes:\n  route:\n    provider: openai\n    transport: native\n    authentication: oauth\n    model: gpt-5.6\n    effort: high\n"+
		"    subagentMaxEffort: high\n    roles:\n      planner: {provider: anthropic, model: opus, effort: high}\n")
	installWork := env.options
	installWork.targets, installWork.makeDefault = []string{"oh-my-pi"}, true
	env.planThenApply(t, "work", installWork)
	return env, filepath.Join(env.ompDir, "config.yml")
}

func replaceInFile(t *testing.T, path, old, replacement string) {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil || !strings.Contains(string(data), old) {
		t.Fatalf("%s lacks %q (err=%v):\n%s", path, old, err, data)
	}
	writeFile(t, path, strings.Replace(string(data), old, replacement, 1))
}

func humanStatusForTest(t *testing.T, options installOptions) string {
	t.Helper()
	var output bytes.Buffer
	cmd := &cobra.Command{}
	cmd.SetOut(&output)
	if err := runStatus(cmd, options); err != nil {
		t.Fatalf("status: %v\n%s", err, output.String())
	}
	return output.String()
}

func TestStatusNamesDriftFromProfile(t *testing.T) {
	cases := map[string]struct {
		edit       func(t *testing.T, env useTestHome, config string)
		wantDrift  []install.FieldDrift
		wantSource string
		wantLines  []string
	}{
		"clean install has no drift": {
			edit:       func(*testing.T, useTestHome, string) {},
			wantSource: install.SourceCurrent,
		},
		"edited owned config keys are named with live and profile values": {
			edit: func(t *testing.T, _ useTestHome, config string) {
				replaceInFile(t, config, `maxEffort: "high"`, `maxEffort: "max"`)
				replaceInFile(t, config, `plan: "anthropic/opus:high"`, `plan: "user/other:low"`)
			},
			wantDrift: []install.FieldDrift{
				{Path: "config.modelRoles.plan", Live: "user/other:low", Profile: "anthropic/opus:high"},
				{Path: "config.task.maxEffort", Live: "max", Profile: "high"},
			},
			wantSource: install.SourceChanged,
			wantLines:  []string{"  drift:\n", "    config.modelRoles.plan  live user/other:low  profile anthropic/opus:high\n", "    config.task.maxEffort  live max  profile high\n"},
		},
		"removed owned config key reads as absent": {
			edit: func(t *testing.T, _ useTestHome, config string) {
				replaceInFile(t, config, "task:\n  maxEffort: \"high\"\n", "")
			},
			wantDrift:  []install.FieldDrift{{Path: "config.task.maxEffort", Profile: "high"}},
			wantSource: install.SourceChanged,
			wantLines:  []string{"    config.task.maxEffort  live (absent)  profile high\n"},
		},
		"edited global instruction is named by path": {
			edit: func(t *testing.T, env useTestHome, _ string) {
				writeFile(t, filepath.Join(env.ompDir, "AGENTS.md"), "# edited by hand\n")
			},
			wantDrift:  []install.FieldDrift{{Path: "AGENTS.md", State: install.FileEdited}},
			wantSource: install.SourceChanged,
			wantLines:  []string{"    AGENTS.md  edited\n"},
		},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			env, config := newDriftTestHome(t)
			tc.edit(t, env, config)
			omp := targetStatus(t, statusForTest(t, env.options), "oh-my-pi")
			if omp.Source != tc.wantSource || !reflect.DeepEqual(omp.Drift, tc.wantDrift) {
				t.Fatalf("source %s (%s), drift %#v; want %s, %#v", omp.Source, omp.SourceReason, omp.Drift, tc.wantSource, tc.wantDrift)
			}
			if len(tc.wantDrift) > 0 && !strings.Contains(omp.SourceReason, "--override") {
				t.Fatalf("live-edit source reason lacks --override: %s", omp.SourceReason)
			}
			human := humanStatusForTest(t, env.options)
			if len(tc.wantDrift) == 0 && strings.Contains(human, "drift:") {
				t.Fatalf("clean status prints drift:\n%s", human)
			}
			for _, line := range tc.wantLines {
				if !strings.Contains(human, line) {
					t.Fatalf("human status lacks %q:\n%s", line, human)
				}
			}
		})
	}
}

// TestUseOverDriftStillConflictsWithoutOverride pins that status planning with
// override does not leak into install: a live edit of an owned file still blocks use.
func TestUseOverDriftStillConflictsWithoutOverride(t *testing.T) {
	cases := map[string]struct {
		edit func(t *testing.T, env useTestHome, config string)
	}{
		"edited config key": {edit: func(t *testing.T, _ useTestHome, config string) {
			replaceInFile(t, config, `maxEffort: "high"`, `maxEffort: "max"`)
		}},
		"edited global instruction": {edit: func(t *testing.T, env useTestHome, _ string) {
			writeFile(t, filepath.Join(env.ompDir, "AGENTS.md"), "# edited by hand\n")
		}},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			env, config := newDriftTestHome(t)
			tc.edit(t, env, config)
			var output bytes.Buffer
			cmd := &cobra.Command{}
			cmd.SetOut(&output)
			cmd.SetErr(&output)
			if err := runInstall(cmd, "work", env.useOptions()); err == nil {
				t.Fatalf("use over a drifted owned file planned without --override:\n%s", output.String())
			}
			if !strings.Contains(output.String(), "conflict") {
				t.Fatalf("use output does not report a conflict:\n%s", output.String())
			}
		})
	}
}

// TestStatusAcceptsManifestRecordedAtOlderTargetVersion covers manifests written before
// an adapter was requalified: the same target name at another version still loads.
func TestStatusAcceptsManifestRecordedAtOlderTargetVersion(t *testing.T) {
	cases := map[string]struct {
		recorded string
		wantErr  bool
	}{
		"older version of the same target": {recorded: `"name": "oh-my-pi",` + "\n    " + `"version": "18.0.0"`},
		"different target name":            {recorded: `"name": "codex",` + "\n    " + `"version": "18.0.0"`, wantErr: true},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			env, config := newDriftTestHome(t)
			manifest := config + ".profile-mango.manifest.json"
			current := targetStatus(t, statusForTest(t, env.options), "oh-my-pi").Target
			replaceInFile(t, manifest, `"name": "oh-my-pi",`+"\n    "+`"version": "`+current.Version+`"`, tc.recorded)
			var output bytes.Buffer
			cmd := &cobra.Command{}
			cmd.SetOut(&output)
			options := env.options
			options.jsonOutput = true
			err := runStatus(cmd, options)
			if tc.wantErr {
				if err == nil || !strings.Contains(err.Error(), "does not match") {
					t.Fatalf("status over another target's manifest = %v", err)
				}
				return
			}
			omp := targetStatus(t, statusForTest(t, env.options), "oh-my-pi")
			if omp.RecordedVersion != "18.0.0" || omp.Source != install.SourceChanged || len(omp.Drift) != 0 {
				t.Fatalf("status over older manifest = %#v", omp)
			}
			if human := humanStatusForTest(t, env.options); !strings.Contains(human, "(recorded at 18.0.0)") {
				t.Fatalf("human status lacks the recorded version:\n%s", human)
			}
			env.planThenApply(t, "work", env.useOptions())
			omp = targetStatus(t, statusForTest(t, env.options), "oh-my-pi")
			if omp.RecordedVersion != "" || omp.Source != install.SourceCurrent {
				t.Fatalf("status after re-apply = %#v", omp)
			}
		})
	}
}
