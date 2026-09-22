package integration_test

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"gitlab.com/ariel-frischer/profile-mango/internal/installfs"
	"gitlab.com/ariel-frischer/profile-mango/pkg/install"
)

type installWorkflow struct {
	binary, root, config, original string
	env, args                      []string
}

func TestInstalledBinaryNoninteractiveInstallWorkflow(t *testing.T) {
	w := newInstallWorkflow(t)
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
	if !strings.Contains(string(readWorkflowFile(t, w.config)), `"model": "openai/gpt-5.6"`) {
		t.Fatal("install did not update the active config")
	}
	backup := installfs.BackupPath(w.config, plan.PlanID)
	assertWorkflowBytes(t, backup, before)
	w.checkNoopAndStale(t, plan.PlanID)
	if err := os.WriteFile(w.config, readWorkflowFile(t, backup), 0o600); err != nil {
		t.Fatal(err)
	}
	assertWorkflowBytes(t, w.config, before)
	assertWorkflowBytes(t, filepath.Join(w.root, "outside-sentinel"), []byte("untouched"))
}

func newInstallWorkflow(t *testing.T) installWorkflow {
	t.Helper()
	repo := absolutePath(t, "..")
	root := t.TempDir()
	w := installWorkflow{root: root, binary: filepath.Join(root, executableName()), env: cleanInstallEnv(t, root)}
	runGo(t, repo, w.env, "build", "-o", w.binary, "./cmd/profile-mango")
	profile := filepath.Join(root, "profiles", "minimal")
	if err := os.MkdirAll(profile, 0o700); err != nil {
		t.Fatal(err)
	}
	files := map[string]string{
		filepath.Join(profile, "profile.yaml"):  "apiVersion: profilemango.dev/v1alpha1\nkind: PolicyProfile\nmetadata:\n  name: minimal\nspec:\n  routeRef: primary\n",
		filepath.Join(root, "bindings.yaml"):    "routes:\n  primary:\n    provider: openai\n    transport: native\n    authentication: oauth\n    model: gpt-5.6\n    effort: high\n",
		filepath.Join(root, "outside-sentinel"): "untouched",
	}
	w.config = filepath.Join(root, "opencode.jsonc")
	w.original = "{\n  // keep user comment\n  \"model\": \"openai/old\",\n  \"theme\": \"system\"\n}\n"
	files[w.config] = w.original
	for path, content := range files {
		if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	w.args = []string{"install", "minimal", "--profiles", filepath.Join(root, "profiles"), "--resource-root", root,
		"--bindings", filepath.Join(root, "bindings.yaml"), "--target", "opencode@1.18.31", "--config-path", "opencode@1.18.31=" + w.config, "--override", "--json"}
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
	want := strings.Replace(w.original, `"openai/old"`, `"openai/gpt-5.6"`, 1)
	assertWorkflowBytes(t, w.config, []byte(want))
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
