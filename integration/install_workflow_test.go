package integration_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"gitlab.com/ariel-frischer/profile-mango/internal/installfs"
	"gitlab.com/ariel-frischer/profile-mango/pkg/install"
)

type installWorkflow struct {
	binary, root, config, original, expected string
	env, args                                []string
}

func TestInstalledBinaryNoninteractiveInstallWorkflow(t *testing.T) {
	tests := map[string]struct {
		provider, model, original, expected string
	}{
		"opencode@1.18.31":    {"openai", "gpt-5.6", "{\n  // keep user comment\n  \"model\": \"openai/old\",\n  \"theme\": \"system\"\n}\n", "{\n  // keep user comment\n  \"model\": \"openai/gpt-5.6\",\n  \"theme\": \"system\"\n}\n"},
		"claude-code@2.1.278": {"anthropic", "claude-sonnet-4-5", "{\n  \"model\": \"old-model\",\n  \"unknown\": true\n}\n", "{\n  \"model\": \"claude-sonnet-4-5\",\n  \"unknown\": true\n}\n"},
		"hermes@0.21.3":       {"openai", "gpt-5.6", "# keep\nmodel:\n  provider: old\n  default: old\nagent:\n  reasoning_effort: low\nunknown: true\n", "# keep\nmodel:\n  provider: \"openai\"\n  default: \"gpt-5.6\"\nagent:\n  reasoning_effort: \"high\"\nunknown: true\n"},
		"openclaw@2026.9.5":   {"openai", "gpt-5.6", "// keep\n{agents:{defaults:{model:{primary:\"old\"},thinkingDefault:\"low\"}},unknown:true}\n", "// keep\n{agents:{defaults:{model:{primary:\"openai/gpt-5.6\"},thinkingDefault:\"high\"}},unknown:true}\n"},
		"pi@0.86.1":           {"openai", "gpt-5.6", "{\n  \"defaultProvider\": \"old\",\n  \"defaultModel\": \"old\",\n  \"defaultThinkingLevel\": \"low\",\n  \"unknown\": true\n}\n", "{\n  \"defaultProvider\": \"openai\",\n  \"defaultModel\": \"gpt-5.6\",\n  \"defaultThinkingLevel\": \"high\",\n  \"unknown\": true\n}\n"},
	}
	for target, test := range tests {
		t.Run(target, func(t *testing.T) {
			w := newInstallWorkflow(t, target, test.provider, test.model, test.original, test.expected)
			w.checkInstall(t)
		})
	}
}

func (w installWorkflow) checkInstall(t *testing.T) {
	t.Helper()
	before := readWorkflowFile(t, w.config)
	plan := w.plan(t)
	if plan.Status != install.StatusReady || !plan.Backup || len(plan.Targets[0].Fields) == 0 {
		t.Fatalf("unexpected plan: %#v", plan)
	}
	if next := w.plan(t); next.PlanID != plan.PlanID {
		t.Fatal("unchanged plan is not deterministic")
	}
	assertWorkflowBytes(t, w.config, before)
	w.rejectUnconfirmedApply(t, plan.PlanID)
	result := w.run(t, "--apply", "--yes", "--expect-plan", plan.PlanID)
	if result.err != nil {
		t.Fatalf("automated apply failed: %v\n%s", result.err, result.stderr)
	}
	var report install.ApplyReport
	if err := json.Unmarshal([]byte(result.stdout), &report); err != nil || report.Status != "committed" {
		t.Fatalf("expected one committed JSON report: err=%v output=%s", err, result.stdout)
	}
	assertWorkflowBytes(t, w.config, []byte(w.expected))
	backup := installfs.BackupPath(w.config, plan.PlanID)
	assertWorkflowBytes(t, backup, before)
	w.checkNoopAndStale(t, plan.PlanID)
	if err := os.WriteFile(w.config, readWorkflowFile(t, backup), 0o600); err != nil {
		t.Fatal(err)
	}
	assertWorkflowBytes(t, w.config, before)
	assertWorkflowBytes(t, filepath.Join(w.root, "outside-sentinel"), []byte("untouched"))
}

func TestInstalledBinaryMultiTargetInstall(t *testing.T) {
	w := newInstallWorkflow(t, "claude-code@2.1.278", "anthropic", "claude-sonnet-4-5",
		"{\"model\":\"old\",\"keep\":true}\n", "{\"model\":\"claude-sonnet-4-5\",\"keep\":true}\n")
	piPath := filepath.Join(w.root, "pi-settings.json")
	before := []byte("{\"defaultProvider\":\"old\",\"defaultModel\":\"old\",\"defaultThinkingLevel\":\"low\",\"keep\":true}\n")
	if err := os.WriteFile(piPath, before, 0o600); err != nil {
		t.Fatal(err)
	}
	w.args = append(w.args, "--target", "pi@0.86.1", "--config-path", "pi="+piPath)
	plan := w.plan(t)
	if len(plan.Targets) != 2 {
		t.Fatalf("expected two targets, got %#v", plan.Targets)
	}
	w.checkInstall(t)
	want := []byte("{\"defaultProvider\":\"anthropic\",\"defaultModel\":\"claude-sonnet-4-5\",\"defaultThinkingLevel\":\"high\",\"keep\":true}\n")
	assertWorkflowBytes(t, piPath, want)
	backup := installfs.BackupPath(piPath, plan.PlanID)
	assertWorkflowBytes(t, backup, before)
	if err := os.WriteFile(piPath, readWorkflowFile(t, backup), 0o600); err != nil {
		t.Fatal(err)
	}
	assertWorkflowBytes(t, piPath, before)
}

func newInstallWorkflow(t *testing.T, target, provider, model, original, expected string) installWorkflow {
	t.Helper()
	repo := absolutePath(t, "..")
	root := t.TempDir()
	w := installWorkflow{root: root, binary: filepath.Join(root, executableName()), env: cleanInstallEnv(t, root), original: original, expected: expected}
	runGo(t, repo, w.env, "build", "-o", w.binary, "./cmd/mango")
	profile := filepath.Join(root, "profiles", "minimal")
	if err := os.MkdirAll(profile, 0o700); err != nil {
		t.Fatal(err)
	}
	files := map[string]string{
		filepath.Join(profile, "profile.yaml"):  "apiVersion: profilemango.dev/v1alpha1\nkind: PolicyProfile\nmetadata:\n  name: minimal\nspec:\n  routeRef: primary\n",
		filepath.Join(root, "bindings.yaml"):    fmt.Sprintf("routes:\n  primary:\n    provider: %s\n    transport: native\n    authentication: oauth\n    model: %s\n    effort: high\n", provider, model),
		filepath.Join(root, "outside-sentinel"): "untouched",
	}
	w.config = filepath.Join(root, "target.json")
	files[w.config] = w.original
	for path, content := range files {
		if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	w.args = []string{"--non-interactive", "install", "minimal", "--profiles", filepath.Join(root, "profiles"), "--resource-root", root,
		"--bindings", filepath.Join(root, "bindings.yaml"), "--target", target, "--config-path", target + "=" + w.config, "--override", "--json"}
	return w
}

func (w installWorkflow) run(t *testing.T, extra ...string) commandResult {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, w.binary, append(append([]string(nil), w.args...), extra...)...)
	cmd.Dir, cmd.Env = w.root, w.env
	var stdout, stderr bytes.Buffer
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	err := cmd.Run()
	if ctx.Err() != nil {
		t.Fatal("noninteractive install hung instead of completing or rejecting input")
	}
	return commandResult{stdout: stdout.String(), stderr: stderr.String(), err: err}
}

func (w installWorkflow) plan(t *testing.T) install.Plan {
	t.Helper()
	result := w.run(t)
	if result.err != nil {
		t.Fatalf("plan failed: %v\n%s", result.err, result.stderr)
	}
	var plan install.Plan
	if err := json.Unmarshal([]byte(result.stdout), &plan); err != nil {
		t.Fatalf("decode plan: %v\n%s", err, result.stdout)
	}
	return plan
}

func (w installWorkflow) rejectUnconfirmedApply(t *testing.T, planID string) {
	t.Helper()
	tests := map[string][]string{
		"no consent": {"--apply"},
		"no hash":    {"--apply", "--yes"},
		"wrong hash": {"--apply", "--yes", "--expect-plan", "not-" + planID},
	}
	for name, args := range tests {
		t.Run(name, func(t *testing.T) {
			if result := w.run(t, args...); result.err == nil {
				t.Fatal("unconfirmed apply succeeded")
			}
			assertWorkflowBytes(t, w.config, []byte(w.original))
		})
	}
}

func (w installWorkflow) checkNoopAndStale(t *testing.T, oldID string) {
	t.Helper()
	after := readWorkflowFile(t, w.config)
	assertWorkflowBytes(t, w.config, []byte(w.expected))
	plan := w.plan(t)
	if plan.Status != install.StatusNoop {
		t.Fatalf("reapply status=%s, want noop", plan.Status)
	}
	if result := w.run(t, "--apply", "--yes", "--expect-plan", oldID); result.err == nil {
		t.Fatal("stale consent was accepted")
	}
	if result := w.run(t, "--apply", "--yes", "--expect-plan", plan.PlanID); result.err != nil {
		t.Fatalf("no-op reapply failed: %v\n%s", result.err, result.stderr)
	}
	assertWorkflowBytes(t, w.config, after)
}

func readWorkflowFile(t *testing.T, path string) []byte {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return data
}

func assertWorkflowBytes(t *testing.T, path string, want []byte) {
	t.Helper()
	if got := readWorkflowFile(t, path); !bytes.Equal(got, want) {
		t.Fatalf("unexpected contents at %s: %q, want %q", path, got, want)
	}
}
