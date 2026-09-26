package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/spf13/cobra"
	"gitlab.com/ariel-frischer/profile-mango/pkg/adapters/codex"
	"gitlab.com/ariel-frischer/profile-mango/pkg/adapters/ohmypi"
	"gitlab.com/ariel-frischer/profile-mango/pkg/install"
)

// runDoctorForTest builds a fresh doctor command for every call instead of
// reusing the package-level rootCmd singleton. newDoctorCmd's flags are bound
// to a options struct captured by its own closure, so reusing one instance
// across tests would leak a flag value (e.g. --profile) from one test into
// the next, exactly like the nonInteractive leak fixed for ap-uuz.11.
func runDoctorForTest(t *testing.T, args ...string) (string, error) {
	t.Helper()
	var output bytes.Buffer
	cmd := newDoctorCmd()
	cmd.SetOut(&output)
	cmd.SetErr(new(bytes.Buffer))
	cmd.SetArgs(args)
	err := cmd.Execute()
	return output.String(), err
}

// withDoctorHome points selectedHome() at a fixture profile-home directory
// for the duration of one test and restores the package-level override with
// t.Cleanup, mirroring how other tests in this package set homePathOverride
// directly instead of going through the root --home flag.
func withDoctorHome(t *testing.T, home string) {
	t.Helper()
	original := homePathOverride
	homePathOverride = home
	t.Cleanup(func() { homePathOverride = original })
}

// withShortDoctorProbeTimeout shortens the version-probe timeout for a test
// and restores it with t.Cleanup, so slow-binary tests don't wait out the
// real 3s production timeout.
func withShortDoctorProbeTimeout(t *testing.T, timeout time.Duration) {
	t.Helper()
	original := doctorProbeTimeout
	doctorProbeTimeout = timeout
	t.Cleanup(func() { doctorProbeTimeout = original })
}

// doctorFakeBinaries writes one executable shell script per binary name into
// a fresh directory and points PATH at only that directory, so binary
// detection is deterministic and independent of whatever is actually
// installed on the machine running the test. Script bodies must stick to
// POSIX shell builtins (printf, :, while, echo, ...); PATH carries nothing
// else for them to call out to.
func doctorFakeBinaries(t *testing.T, scripts map[string]string) {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("fake shell-script binaries require a POSIX shell")
	}
	dir := t.TempDir()
	for name, body := range scripts {
		path := filepath.Join(dir, name)
		if err := os.WriteFile(path, []byte("#!/bin/sh\n"+body+"\n"), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	t.Setenv("PATH", dir)
}

// writeDoctorProfileHome scaffolds a minimal profile home a codex plan can
// resolve: <home>/profiles/<name>/profile.yaml and <home>/bindings/local.yaml.
func writeDoctorProfileHome(t *testing.T, home, profileName string) {
	t.Helper()
	writeFile(t, filepath.Join(home, "profiles", profileName, "profile.yaml"),
		"apiVersion: profilemango.dev/v1alpha1\nkind: PolicyProfile\nmetadata:\n  name: "+profileName+"\nspec:\n  routeRef: route\n")
	writeFile(t, filepath.Join(home, "bindings", "local.yaml"),
		"routes:\n  route:\n    provider: openai\n    transport: native\n    authentication: oauth\n    model: gpt-5.6\n    effort: high\n")
}

func decodeDoctorReport(t *testing.T, output string) doctorReport {
	t.Helper()
	var report doctorReport
	if err := json.Unmarshal([]byte(output), &report); err != nil {
		t.Fatalf("decode doctor JSON: %v\n%s", err, output)
	}
	return report
}

func doctorTargetNamed(t *testing.T, report doctorReport, name string) doctorTarget {
	t.Helper()
	for _, target := range report.Targets {
		if strings.HasPrefix(target.Target, name+"@") {
			return target
		}
	}
	t.Fatalf("target %s missing from report: %#v", name, report.Targets)
	return doctorTarget{}
}

func TestDoctorHidesArielJcodeUnlessExperimental(t *testing.T) {
	doctorFakeBinaries(t, nil)
	output, err := runDoctorForTest(t, "--help")
	if err != nil {
		t.Fatalf("doctor --help: %v\n%s", err, output)
	}
	for _, term := range []string{"jcode-fork", "jcode"} {
		if strings.Contains(output, term) {
			t.Fatalf("doctor --help mentions hidden target %q:\n%s", term, output)
		}
	}
	if flag := newDoctorCmd().Flags().Lookup("experimental"); flag == nil || !flag.Hidden {
		t.Fatal("doctor --experimental flag must exist and stay hidden")
	}

	registry := install.DefaultRegistry()
	if names := doctorTargetNames(doctorTargets(registry, false)); contains(names, "jcode-fork") {
		t.Fatalf("default doctor targets include jcode-fork: %v", names)
	}
	if names := doctorTargetNames(doctorTargets(registry, true)); !contains(names, "jcode-fork") {
		t.Fatalf("--experimental doctor targets omit jcode-fork: %v", names)
	}
}

func doctorTargetNames(targets []install.Target) []string {
	names := make([]string, 0, len(targets))
	for _, target := range targets {
		names = append(names, target.Name)
	}
	return names
}

func contains(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}

func TestDoctorWritesNothing(t *testing.T) {
	doctorFakeBinaries(t, map[string]string{"codex": "echo 'codex-cli 0.154.0'"})
	root := t.TempDir()
	home := filepath.Join(root, "home")
	t.Setenv("HOME", home)
	t.Setenv("CODEX_HOME", "")

	output, err := runDoctorForTest(t)
	if err != nil {
		t.Fatalf("doctor: %v\n%s", err, output)
	}
	if !strings.Contains(output, "codex@0.154.0") {
		t.Fatalf("doctor output missing codex row:\n%s", output)
	}
	if _, statErr := os.Stat(home); !os.IsNotExist(statErr) {
		t.Fatalf("doctor created %s: %v", home, statErr)
	}
}

func TestDoctorShowsProbeProgressOnStderrAndKeepsJSONClean(t *testing.T) {
	doctorFakeBinaries(t, map[string]string{"codex": "while :; do :; done"})
	withShortDoctorProbeTimeout(t, 100*time.Millisecond)
	withDoctorHome(t, t.TempDir())
	cmd := newDoctorCmd()
	cmd.SetArgs([]string{"--json"})
	var stdout, stderr bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	if err := cmd.Execute(); err != nil {
		t.Fatalf("doctor: %v", err)
	}
	if !strings.Contains(stderr.String(), "checking") || !strings.Contains(stderr.String(), "codex") || !strings.Contains(stderr.String(), "checked") {
		t.Fatalf("missing version probe progress: %q", stderr.String())
	}
	decodeDoctorReport(t, stdout.String())
}

// TestDoctorRegisteredOnRootCommand exercises doctor through the real
// rootCmd, proving it is wired into root.go's command tree and inherits the
// --home persistent flag machinery, without passing any doctor-specific flag
// (so it cannot leak a non-default value into another test's singleton run).
func TestDoctorRegisteredOnRootCommand(t *testing.T) {
	doctorFakeBinaries(t, nil)
	t.Setenv("HOME", filepath.Join(t.TempDir(), "home"))
	output := executeCommand(t, "doctor")
	if !strings.Contains(output, "TARGET") {
		t.Fatalf("doctor via root command missing table header:\n%s", output)
	}
}

func TestDoctorHumanOutputHasSuggestion(t *testing.T) {
	doctorFakeBinaries(t, nil)
	t.Setenv("HOME", filepath.Join(t.TempDir(), "home"))

	output, err := runDoctorForTest(t)
	if err != nil {
		t.Fatalf("doctor: %v\n%s", err, output)
	}
	if !strings.Contains(output, "TARGET") || !strings.Contains(output, "PLAN") {
		t.Fatalf("doctor output missing table header:\n%s", output)
	}
	if !strings.Contains(output, "not found") {
		t.Fatalf("doctor output should report missing binaries:\n%s", output)
	}
	if !strings.Contains(output, "next: run mango init") {
		t.Fatalf("doctor output missing next-step suggestion:\n%s", output)
	}
}

func TestDoctorJSONReportsVersionRange(t *testing.T) {
	doctorFakeBinaries(t, map[string]string{
		"codex":  "echo 'codex-cli " + codex.TargetVersion + "'",
		"claude": "echo '9.9.9 (Claude Code)'",
		"pi":     "echo '0.86.4'",
	})
	t.Setenv("HOME", filepath.Join(t.TempDir(), "home"))

	output, err := runDoctorForTest(t, "--json")
	if err != nil {
		t.Fatalf("doctor --json: %v\n%s", err, output)
	}
	report := decodeDoctorReport(t, output)
	if report.APIVersion == "" || report.Kind != "DoctorReport" || report.Profile != "default" {
		t.Fatalf("report envelope = %#v", report)
	}
	codexResult := doctorTargetNamed(t, report, "codex")
	if !codexResult.BinaryFound || !codexResult.VersionMatch || !codexResult.InRange || codexResult.CompatibleRange != ">=0.154.0 <0.155.0" || codexResult.DetectedVersion != "codex-cli "+codex.TargetVersion {
		t.Fatalf("codex result = %#v", codexResult)
	}
	claudeResult := doctorTargetNamed(t, report, "claude-code")
	if !claudeResult.BinaryFound || claudeResult.VersionMatch || claudeResult.InRange || claudeResult.CompatibleRange != ">=2.1.278 <2.2.0" {
		t.Fatalf("claude-code result = %#v", claudeResult)
	}
	piResult := doctorTargetNamed(t, report, "pi")
	if !piResult.BinaryFound || piResult.VersionMatch || !piResult.InRange {
		t.Fatalf("pi patch update should be in range without an exact match: %#v", piResult)
	}
	hermesResult := doctorTargetNamed(t, report, "hermes")
	if hermesResult.BinaryFound || hermesResult.BinaryPath != "" || hermesResult.InRange || hermesResult.CompatibleRange == "" {
		t.Fatalf("hermes (no fake binary) result = %#v", hermesResult)
	}
}

func TestDoctorHumanTableShowsInRange(t *testing.T) {
	doctorFakeBinaries(t, map[string]string{
		"codex":  "echo 'codex-cli 0.154.9'",
		"claude": "echo '2.2.0 (Claude Code)'",
	})
	t.Setenv("HOME", filepath.Join(t.TempDir(), "home"))
	output, err := runDoctorForTest(t)
	if err != nil {
		t.Fatalf("doctor: %v\n%s", err, output)
	}
	if !strings.Contains(output, "IN RANGE") || strings.Contains(output, "MATCH") {
		t.Fatalf("doctor header should say IN RANGE:\n%s", output)
	}
	for prefix, want := range map[string]string{"codex@": "codex-cli 0.154.9  yes  ", "claude-code@": "(Claude Code)  no (tested >=2.1.278 <2.2.0)  "} {
		if !strings.Contains(doctorRow(output, prefix), want) {
			t.Fatalf("doctor row %s missing %q:\n%s", prefix, want, output)
		}
	}
}

// doctorRow returns the table row for a target with column padding collapsed
// to two spaces (plus a trailing separator), so assertions ignore widths.
func doctorRow(output, prefix string) string {
	for _, line := range strings.Split(output, "\n") {
		if !strings.HasPrefix(line, prefix) {
			continue
		}
		for strings.Contains(line, "   ") {
			line = strings.ReplaceAll(line, "   ", "  ")
		}
		return line + "  "
	}
	return ""
}

func TestDoctorProbeTimesOut(t *testing.T) {
	withShortDoctorProbeTimeout(t, 50*time.Millisecond)
	doctorFakeBinaries(t, map[string]string{"codex": "while :; do :; done"})
	t.Setenv("HOME", filepath.Join(t.TempDir(), "home"))

	output, err := runDoctorForTest(t, "--json")
	if err != nil {
		t.Fatalf("doctor --json: %v\n%s", err, output)
	}
	report := decodeDoctorReport(t, output)
	codexResult := doctorTargetNamed(t, report, "codex")
	if codexResult.ProbeError == "" || !strings.Contains(codexResult.ProbeError, "timed out") {
		t.Fatalf("codex probe = %#v, want a timeout error", codexResult)
	}
}

func TestDoctorProbeCapsOutput(t *testing.T) {
	doctorFakeBinaries(t, map[string]string{"codex": "i=0\nwhile [ $i -lt 5000 ]; do printf 'aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa'; i=$((i+1)); done"})
	t.Setenv("HOME", filepath.Join(t.TempDir(), "home"))

	output, err := runDoctorForTest(t, "--json")
	if err != nil {
		t.Fatalf("doctor --json: %v\n%s", err, output)
	}
	report := decodeDoctorReport(t, output)
	codexResult := doctorTargetNamed(t, report, "codex")
	if len(codexResult.DetectedVersion) > doctorProbeCapBytes {
		t.Fatalf("detected version length = %d, want <= %d", len(codexResult.DetectedVersion), doctorProbeCapBytes)
	}
	if len(codexResult.DetectedVersion) == 0 {
		t.Fatal("expected capped but non-empty detected version")
	}
}

func TestDoctorConfigPathExistsFlag(t *testing.T) {
	doctorFakeBinaries(t, nil)
	home := filepath.Join(t.TempDir(), "home")
	t.Setenv("HOME", home)
	t.Setenv("CODEX_HOME", "")
	configPath := filepath.Join(home, ".codex", "config.toml")
	writeFile(t, configPath, "# existing\n")

	output, err := runDoctorForTest(t, "--json")
	if err != nil {
		t.Fatalf("doctor --json: %v\n%s", err, output)
	}
	report := decodeDoctorReport(t, output)
	codexResult := doctorTargetNamed(t, report, "codex")
	if codexResult.ConfigPath != configPath || !codexResult.ConfigExists {
		t.Fatalf("codex settings-path result = %#v, want path %s existing", codexResult, configPath)
	}
}

func TestDoctorPlanStatusReadyWithProfileHome(t *testing.T) {
	doctorFakeBinaries(t, map[string]string{"codex": "echo 'codex-cli " + codex.TargetVersion + "'"})
	// HOME still isolates the real version probes and settings-path checks;
	// the profile home is a distinct fixture directory so the test does not
	// depend on the "<HOME>/.profile-mango" default nesting.
	codexHome := filepath.Join(t.TempDir(), "codex-home")
	if err := os.MkdirAll(filepath.Join(codexHome, ".codex"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("HOME", codexHome)
	t.Setenv("CODEX_HOME", "")
	packageHome := t.TempDir()
	writeDoctorProfileHome(t, packageHome, "default")
	withDoctorHome(t, packageHome)

	output, err := runDoctorForTest(t, "--json")
	if err != nil {
		t.Fatalf("doctor --json: %v\n%s", err, output)
	}
	report := decodeDoctorReport(t, output)
	codexResult := doctorTargetNamed(t, report, "codex")
	if codexResult.PlanStatus != install.StatusReady {
		t.Fatalf("codex plan status = %q, reason = %q, want ready", codexResult.PlanStatus, codexResult.PlanReason)
	}
	if _, statErr := os.Stat(filepath.Join(codexHome, ".codex", "config.toml")); !os.IsNotExist(statErr) {
		t.Fatalf("read-only plan created config: %v", statErr)
	}
}

func TestDoctorProfileFlagSelectsNamedProfile(t *testing.T) {
	doctorFakeBinaries(t, map[string]string{"codex": "echo 'codex-cli " + codex.TargetVersion + "'"})
	codexHome := filepath.Join(t.TempDir(), "codex-home")
	if err := os.MkdirAll(filepath.Join(codexHome, ".codex"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("HOME", codexHome)
	t.Setenv("CODEX_HOME", "")
	packageHome := t.TempDir()
	writeDoctorProfileHome(t, packageHome, "team")
	withDoctorHome(t, packageHome)

	output, err := runDoctorForTest(t, "--profile", "team", "--json")
	if err != nil {
		t.Fatalf("doctor --profile team --json: %v\n%s", err, output)
	}
	report := decodeDoctorReport(t, output)
	if report.Profile != "team" {
		t.Fatalf("report profile = %q, want team", report.Profile)
	}
	codexResult := doctorTargetNamed(t, report, "codex")
	if codexResult.PlanStatus != install.StatusReady {
		t.Fatalf("codex plan status = %q, reason = %q, want ready", codexResult.PlanStatus, codexResult.PlanReason)
	}
}

// TestDoctorPlanMatchesInstallForUnownedConfig guards the shared readiness
// verdict: with an existing agent config profile-mango does not own, a plain
// install plans a backed-up adoption, so doctor must report the same status
// instead of the "requires a backup" conflict a no-backup plan would give.
func TestDoctorPlanMatchesInstallForUnownedConfig(t *testing.T) {
	doctorFakeBinaries(t, map[string]string{"omp": "echo 'omp/" + ohmypi.TargetVersion + "'"})
	home := filepath.Join(t.TempDir(), "home")
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", "")
	writeFile(t, filepath.Join(home, ".omp", "agent", "config.yml"), "theme: dark\n")
	packageHome := t.TempDir()
	writeDoctorProfileHome(t, packageHome, "default")
	withDoctorHome(t, packageHome)
	withAgentDetection(t)

	output, err := runDoctorForTest(t, "--target", "oh-my-pi", "--json")
	if err != nil {
		t.Fatalf("doctor --json: %v\n%s", err, output)
	}
	doctorResult := doctorTargetNamed(t, decodeDoctorReport(t, output), "oh-my-pi")
	var data bytes.Buffer
	cmd := &cobra.Command{}
	cmd.SetOut(&data)
	cmd.SetErr(new(bytes.Buffer))
	if err := runInstall(cmd, "default", installOptions{targets: []string{"oh-my-pi"}, jsonOutput: true}); err != nil {
		t.Fatalf("install plan: %v\n%s", err, data.String())
	}
	var plan install.Plan
	if err := json.Unmarshal(data.Bytes(), &plan); err != nil || len(plan.Targets) != 1 {
		t.Fatalf("decode install plan: %v\n%s", err, data.String())
	}
	if doctorResult.PlanStatus != install.StatusReady || doctorResult.PlanStatus != plan.Targets[0].Status {
		t.Fatalf("doctor plan %q (%s), install plan %q; want both ready", doctorResult.PlanStatus, doctorResult.PlanReason, plan.Targets[0].Status)
	}
}

func TestDoctorShowsVersionLineBehindStderrWarning(t *testing.T) {
	warning := "echo 'WARNING: proceeding, even though we could not create PATH aliases' >&2"
	cases := map[string]struct {
		script string
		want   string
	}{
		"warning on stderr": {script: warning + "\necho 'codex-cli 0.154.9'", want: "codex-cli 0.154.9 (+1 more line(s))  yes  "},
		"warning on stdout": {script: "echo 'WARNING: proceeding'\necho 'codex-cli 0.154.9'", want: "codex-cli 0.154.9 (+1 more line(s))  yes  "},
		"version only":      {script: "echo 'codex-cli 0.154.9'", want: "codex-cli 0.154.9  yes  "},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			doctorFakeBinaries(t, map[string]string{"codex": tc.script})
			t.Setenv("HOME", filepath.Join(t.TempDir(), "home"))
			output, err := runDoctorForTest(t, "--target", "codex")
			if err != nil {
				t.Fatalf("doctor: %v\n%s", err, output)
			}
			if row := doctorRow(output, "codex@"); !strings.Contains(row, tc.want) {
				t.Fatalf("codex row %q missing %q:\n%s", row, tc.want, output)
			}
		})
	}
}
