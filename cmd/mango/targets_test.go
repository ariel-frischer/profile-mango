package main

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/ariel-frischer/profile-mango/pkg/install"
)

func TestTargetList(t *testing.T) {
	cases := map[string]struct {
		input []string
		want  string
		bad   bool
	}{
		"comma and repeats": {[]string{"codex, opencode", "codex", "pi@1.0"}, "codex,opencode,pi@1.0", false},
		"empty trailing":    {[]string{"codex,"}, "", true},
		"empty repeated":    {[]string{"codex", " "}, "", true},
		"none":              {nil, "", false},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			got, err := targetList(tc.input)
			if (err != nil) != tc.bad || (!tc.bad && strings.Join(got, ",") != tc.want) {
				t.Fatalf("targetList(%q) = %q, %v", tc.input, got, err)
			}
		})
	}
}

func TestInstallTargetListEquivalent(t *testing.T) {
	registry := install.DefaultRegistry()
	one, err := installTargets(installOptions{targets: []string{"codex,opencode"}}, registry)
	if err != nil {
		t.Fatal(err)
	}
	repeated, err := installTargets(installOptions{targets: []string{"codex", "opencode", "codex"}}, registry)
	if err != nil || len(one) != 2 || len(repeated) != 2 || one[0] != repeated[0] || one[1] != repeated[1] {
		t.Fatalf("list=%v repeated=%v error=%v", one, repeated, err)
	}
	for _, value := range []string{"codex,,opencode", "codex,unknown"} {
		if _, err := installTargets(installOptions{targets: []string{value}}, registry); err == nil {
			t.Fatalf("accepted %q", value)
		}
	}
	paths, err := parseTargetBindings("--config", []string{"codex=/tmp/a,b"})
	if err != nil || paths[0].value != "/tmp/a,b" {
		t.Fatalf("comma in mapping altered: %v %v", paths, err)
	}
}

func TestDoctorTargetListBeforeProbes(t *testing.T) {
	registry := install.DefaultRegistry()
	selected, err := selectedDoctorTargets(registry, doctorOptions{targets: []string{"opencode,codex", "codex"}})
	if err != nil || len(selected) != 2 || selected[0].Name != "codex" || selected[1].Name != "opencode" {
		t.Fatalf("selected=%v error=%v", selected, err)
	}
	for _, value := range []string{"unknown", "jcode-fork", "codex@invalid", "codex,"} {
		if _, err := selectedDoctorTargets(registry, doctorOptions{targets: []string{value}}); err == nil {
			t.Fatalf("accepted %q", value)
		}
	}
}

func TestRenderRejectsMultipleTargetsBeforeStaging(t *testing.T) {
	cmd := newRenderCmd()
	cmd.SetArgs([]string{"default", "-t", "codex,opencode", "--out", t.TempDir() + "/preview"})
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	if err := cmd.Execute(); err == nil || !strings.Contains(err.Error(), "exactly one") {
		t.Fatalf("multi-render: %v", err)
	}
}

func TestUndoRejectsMultiApplyBeforeMutation(t *testing.T) {
	cmd := newRestoreCmd()
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{"-t", "codex,opencode", "--apply", "--yes", "--expect-plan", strings.Repeat("a", 64)})
	if err := cmd.Execute(); err == nil || !strings.Contains(err.Error(), "preview-only") {
		t.Fatalf("multi-undo apply: %v", err)
	}
}

func TestResolvedSelectorAliasesDeduplicate(t *testing.T) {
	undo, err := resolvedUndoTargets([]string{"codex", "codex@0.154.0", "opencode"})
	if err != nil || strings.Join(undo, ",") != "codex@0.154.0,opencode@1.18.31" {
		t.Fatalf("undo targets=%v error=%v", undo, err)
	}
	render, err := resolvedRenderTargets([]string{"codex", "codex@0.154.0"}, "")
	if err != nil || len(render) != 1 || render[0] != "codex@0.154.0" {
		t.Fatalf("render targets=%v error=%v", render, err)
	}
}

func TestDoctorTargetListJSON(t *testing.T) {
	home := t.TempDir()
	withDoctorHome(t, home)
	t.Setenv("HOME", home)
	t.Setenv("CODEX_HOME", "")
	doctorFakeBinaries(t, nil)
	out, err := runDoctorForTest(t, "-t", "codex,opencode", "--json")
	if err != nil {
		t.Fatal(err)
	}
	var report doctorReport
	if err := json.Unmarshal([]byte(out), &report); err != nil || len(report.Targets) != 2 {
		t.Fatalf("invalid selected doctor report: %v %s", err, out)
	}
}

func TestAgentsCheckRejectsUnknownListBeforeNetwork(t *testing.T) {
	cmd := newAgentsCheckCmd()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"-t", "codex,missing-target", "--manifest", "../../docs/dev/agents/sources.json"})
	if err := cmd.Execute(); err == nil || !strings.Contains(err.Error(), "missing-target") {
		t.Fatalf("source selector unexpectedly proceeded: %v %s", err, out.String())
	}
}
