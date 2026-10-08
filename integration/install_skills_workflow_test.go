package integration_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ariel-frischer/profile-mango/pkg/install"
)

const workflowSkill = "---\nname: research\ndescription: Use only for disposable installation tests.\n---\n\nSynthetic skill body.\n"

func TestInstalledBinaryOpenCodeSkillInstall(t *testing.T) {
	w, source := newSkillWorkflow(t)
	plan := w.plan(t)
	if plan.Status != install.StatusReady || len(plan.Targets[0].Skills) != 1 || plan.Targets[0].Skills[0] != "research" {
		t.Fatalf("unexpected skill plan: %#v", plan)
	}
	result := w.run(t, "--apply", "--yes", "--expect-plan", plan.PlanID)
	var report install.ApplyReport
	if err := json.Unmarshal([]byte(result.stdout), &report); result.err != nil || err != nil || report.Status != "committed" {
		t.Fatalf("apply failed: %v %v %s %s", result.err, err, result.stdout, result.stderr)
	}
	installed := workflowInstalledSkill(w)
	assertWorkflowBytes(t, installed, []byte(workflowSkill))
	assertWorkflowBytes(t, workflowBackup(t, w.root, plan.PlanID, w.config), []byte(w.original))
	assertInstalledSkillConfig(t, w)
	noop := w.plan(t)
	if noop.Status != install.StatusNoop {
		t.Fatalf("expected no-op: %#v", noop)
	}
	if err := os.WriteFile(source, []byte(workflowSkill+"Changed source.\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if result := w.run(t, "--apply", "--yes", "--expect-plan", noop.PlanID); result.err == nil {
		t.Fatal("stale source consent was accepted")
	}
	assertWorkflowBytes(t, installed, []byte(workflowSkill))
}

func TestInstalledBinaryOpenCodeSkillOmissionReconciles(t *testing.T) {
	w, _ := newSkillWorkflow(t)
	w.args[1] = "use"
	w.args = withoutWorkflowArg(w.args, "--default")
	initial := w.plan(t)
	result := w.run(t, "--apply", "--yes", "--expect-plan", initial.PlanID)
	if result.err != nil {
		t.Fatalf("initial apply failed: %v\n%s", result.err, result.stderr)
	}
	profile := filepath.Join(w.root, "profiles", "minimal", "profile.yaml")
	content := strings.Replace(string(readWorkflowFile(t, profile)), "skills:\n  - skills/research/SKILL.md\n", "", 1)
	if err := os.WriteFile(profile, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	cleanup := w.plan(t)
	if cleanup.Status != install.StatusReady {
		t.Fatalf("unexpected cleanup plan: %#v", cleanup)
	}
	result = w.run(t, "--apply", "--yes", "--expect-plan", cleanup.PlanID)
	if result.err != nil {
		t.Fatalf("cleanup apply failed: %v\n%s", result.err, result.stderr)
	}
	if _, err := os.Stat(filepath.Dir(workflowInstalledSkill(w))); !os.IsNotExist(err) {
		t.Fatalf("omitted skill folder remains after built-binary cleanup: %v", err)
	}
}

func assertInstalledSkillConfig(t *testing.T, w installWorkflow) {
	t.Helper()
	var config map[string]any
	if err := json.Unmarshal(readWorkflowFile(t, w.config), &config); err != nil {
		t.Fatal(err)
	}
	if config["model"] != "openai/gpt-5.6" || config["theme"] != "system" || config["skills"] != nil {
		t.Fatalf("unexpected config: %#v", config)
	}
}

func TestInstalledBinarySkillAdoptionRequiresBackup(t *testing.T) {
	w, _ := newSkillWorkflow(t)
	target := workflowInstalledSkill(w)
	if err := os.MkdirAll(filepath.Dir(target), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(target, []byte("user-owned"), 0o600); err != nil {
		t.Fatal(err)
	}
	w.args = withoutWorkflowArg(w.args, "--override")
	result := w.run(t, "--no-backup")
	var plan install.Plan
	if err := json.Unmarshal([]byte(result.stdout), &plan); err != nil {
		t.Fatalf("decode plan: %v\n%s\n%s", err, result.stdout, result.stderr)
	}
	if result.err == nil || plan.Status != install.StatusBlocked {
		t.Fatalf("unowned skill adoption without a backup was not blocked: %#v %v", plan, result.err)
	}
	assertWorkflowBytes(t, target, []byte("user-owned"))
	assertWorkflowBytes(t, w.config, []byte(w.original))
}

func withoutWorkflowArg(args []string, drop string) []string {
	kept := make([]string, 0, len(args))
	for _, arg := range args {
		if arg != drop {
			kept = append(kept, arg)
		}
	}
	return kept
}

// workflowInstalledSkill is where OpenCode reads the research skill: skills/ beside its config.
func workflowInstalledSkill(w installWorkflow) string {
	return filepath.Join(filepath.Dir(w.config), "skills", "research", "SKILL.md")
}

// newSkillWorkflow installs a one-skill profile into an OpenCode config kept in its own
// directory, apart from the package's skills/ folder.
func newSkillWorkflow(t *testing.T) (installWorkflow, string) {
	t.Helper()
	w := newInstallWorkflow(t, "opencode@1.18.31", "openai", "gpt-5.6", "{\"model\":\"openai/old\",\"theme\":\"system\"}\n", "")
	config := filepath.Join(w.root, "opencode", "opencode.json")
	if err := os.MkdirAll(filepath.Dir(config), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.Rename(w.config, config); err != nil {
		t.Fatal(err)
	}
	for index, arg := range w.args {
		if arg == "opencode@1.18.31="+w.config {
			w.args[index] = "opencode@1.18.31=" + config
		}
	}
	w.config = config
	w.args = append(w.args, "--default")
	profile := filepath.Join(w.root, "profiles", "minimal", "profile.yaml")
	content := append(readWorkflowFile(t, profile), []byte("skills:\n  - skills/research/SKILL.md\n")...)
	if err := os.WriteFile(profile, content, 0o600); err != nil {
		t.Fatal(err)
	}
	source := filepath.Join(w.root, "skills", "research", "SKILL.md")
	if err := os.MkdirAll(filepath.Dir(source), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(source, []byte(workflowSkill), 0o600); err != nil {
		t.Fatal(err)
	}
	return w, source
}
