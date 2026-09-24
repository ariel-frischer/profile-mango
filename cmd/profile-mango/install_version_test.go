package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/spf13/cobra"
	"gitlab.com/ariel-frischer/profile-mango/pkg/install"
)

// withAgentDetection restores the real PATH-based detector for one test. It
// must be paired with doctorFakeBinaries so only fake commands are probed.
func withAgentDetection(t *testing.T) {
	t.Helper()
	original := installVersionDetector
	installVersionDetector = detectAgentVersion
	t.Cleanup(func() { installVersionDetector = original })
}

func codexVersionInstallOptions(t *testing.T, jsonOutput bool) installOptions {
	t.Helper()
	root := t.TempDir()
	profiles, bindings := writeCodexInstallInputs(t, root)
	config := filepath.Join(root, "target", "config.toml")
	if err := os.MkdirAll(filepath.Dir(config), 0o755); err != nil {
		t.Fatal(err)
	}
	return installOptions{profiles: profiles, resourceRoot: root, bindings: bindings,
		targets: []string{"codex"}, configs: []string{"codex=" + config}, jsonOutput: jsonOutput}
}

func runInstallCapture(t *testing.T, options installOptions) string {
	t.Helper()
	var output bytes.Buffer
	cmd := &cobra.Command{}
	cmd.SetOut(&output)
	cmd.SetErr(new(bytes.Buffer))
	if err := runInstall(cmd, "route-only", options); err != nil {
		t.Fatalf("install: %v\n%s", err, output.String())
	}
	return output.String()
}

func TestInstallReportsInstalledAgentVersion(t *testing.T) {
	tests := map[string]struct {
		scripts map[string]string
		status  string
		want    []string
		absent  string
	}{
		"in range": {scripts: map[string]string{"codex": "echo 'codex-cli 0.154.3'"}, status: install.VersionInRange,
			want: []string{"    installed version: 0.154.3 (in tested range >=0.154.0 <0.155.0)\n"}, absent: "warning: installed codex"},
		"out of range": {scripts: map[string]string{"codex": "echo 'codex-cli 0.155.1'"}, status: install.VersionOutOfRange,
			want: []string{"    installed version: 0.155.1 (outside tested range >=0.154.0 <0.155.0)\n",
				"    warning: installed codex 0.155.1 is outside the tested range >=0.154.0 <0.155.0; settings are written for 0.154.0 and may not be honored\n"}},
		"not found": {scripts: nil, status: install.VersionNotFound,
			want: []string{"    installed version: codex not found on PATH\n", "    warning: codex not found on PATH; installing settings for 0.154.0\n"}},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			doctorFakeBinaries(t, test.scripts)
			withAgentDetection(t)
			options := codexVersionInstallOptions(t, false)
			options.verbose = true
			human := runInstallCapture(t, options)
			if !strings.Contains(human, "(ready)\n") {
				t.Fatalf("version check must not block the plan:\n%s", human)
			}
			for _, want := range test.want {
				if !strings.Contains(human, want) {
					t.Fatalf("human plan missing %q:\n%s", want, human)
				}
			}
			if test.absent != "" && strings.Contains(human, test.absent) {
				t.Fatalf("in-range plan should not warn:\n%s", human)
			}
			assertPlanVersionStatus(t, runInstallCapture(t, codexVersionInstallOptions(t, true)), test.status)
		})
	}
}

func TestInstallShowsVersionProbeProgressWithoutPollutingJSON(t *testing.T) {
	started := make(chan struct{})
	release := make(chan struct{})
	defer close(release)
	original := installVersionDetector
	installVersionDetector = func(target install.Target) install.VersionDetection {
		close(started)
		<-release
		return install.VersionDetection{Binary: target.Name, Found: true, Output: "codex-cli 0.154.0"}
	}
	t.Cleanup(func() { installVersionDetector = original })
	options := codexVersionInstallOptions(t, true)
	cmd := &cobra.Command{}
	var stdout, stderr bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	done := make(chan error, 1)
	go func() { done <- runInstall(cmd, "route-only", options) }()
	select {
	case <-started:
	case <-time.After(time.Second):
		t.Fatal("version probe did not start")
	}
	if got := stderr.String(); !strings.Contains(got, "checking") || !strings.Contains(got, "codex") {
		t.Fatalf("no progress while version probe is pending: %q", got)
	}
	if stdout.Len() != 0 {
		t.Fatalf("stdout before plan = %q", stdout.String())
	}
	release <- struct{}{}
	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("install: %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("install did not finish")
	}
	assertPlanVersionStatus(t, stdout.String(), install.VersionInRange)
	if !strings.Contains(stderr.String(), "checked") {
		t.Fatalf("no completed progress: %q", stderr.String())
	}
}

func assertPlanVersionStatus(t *testing.T, data, status string) {
	t.Helper()
	var plan install.Plan
	if err := json.Unmarshal([]byte(data), &plan); err != nil {
		t.Fatalf("decode plan: %v\n%s", err, data)
	}
	check := plan.Targets[0].VersionCheck
	if plan.Status != install.StatusReady || check == nil || check.Status != status || check.Range != ">=0.154.0 <0.155.0" || check.Binary != "codex" {
		t.Fatalf("plan JSON version check = %#v (plan status %s)", check, plan.Status)
	}
}

func TestInstallExplicitVersionStaysExact(t *testing.T) {
	doctorFakeBinaries(t, map[string]string{"codex": "echo 'codex-cli 0.155.1'"})
	withAgentDetection(t)
	options := codexVersionInstallOptions(t, false)
	options.targets = []string{"codex@0.155.1"}
	options.configs = []string{"codex@0.155.1=" + filepath.Join(t.TempDir(), "config.toml")}
	var output bytes.Buffer
	cmd := &cobra.Command{}
	cmd.SetOut(&output)
	err := runInstall(cmd, "route-only", options)
	if err == nil || !strings.Contains(output.String(), "codex@0.155.1: blocked") || strings.Contains(output.String(), "installed version") {
		t.Fatalf("explicit unqualified version must stay blocked: err=%v\n%s", err, output.String())
	}
}
