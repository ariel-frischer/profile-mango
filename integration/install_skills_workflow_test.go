package integration_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gitlab.com/ariel-frischer/profile-mango/internal/installfs"
	"gitlab.com/ariel-frischer/profile-mango/pkg/install"
)

const workflowSkill = "---\nname: synthetic-profile\ndescription: Use only for disposable installation tests.\n---\n\nSynthetic skill body.\n"

func TestInstalledBinaryOpenCodeSkillInstall(t *testing.T) {
	w, source := newSkillWorkflow(t)
	plan := w.plan(t)
	if plan.Status != install.StatusReady || len(plan.Targets[0].Files) != 4 {
		t.Fatalf("unexpected skill plan: %#v", plan)
	}
	result := w.run(t, "--apply", "--yes", "--expect-plan", plan.PlanID)
	var report install.ApplyReport
	if err := json.Unmarshal([]byte(result.stdout), &report); result.err != nil || err != nil || report.Status != "committed" {
		t.Fatalf("apply failed: %v %v %s %s", result.err, err, result.stdout, result.stderr)
	}
	assertWorkflowBytes(t, filepath.Join(w.root, "SKILL.md"), []byte(workflowSkill))
	assertWorkflowBytes(t, installfs.BackupPath(w.config, plan.PlanID), []byte(w.original))
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
	assertWorkflowBytes(t, filepath.Join(w.root, "SKILL.md"), []byte(workflowSkill))
}

func TestInstalledBinaryOpenCodeSkillOmissionReconciles(t *testing.T) {
	w, _ := newSkillWorkflow(t)
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
	if _, err := os.Stat(filepath.Join(w.root, "SKILL.md")); !os.IsNotExist(err) {
		t.Fatalf("omitted skill remains after built-binary cleanup: %v", err)
	}
	var config struct {
		Skills struct {
			Paths []string `json:"paths"`
		} `json:"skills"`
	}
	if err := json.Unmarshal(readWorkflowFile(t, w.config), &config); err != nil {
		t.Fatal(err)
	}
	if len(config.Skills.Paths) != 0 {
		t.Fatalf("managed skills path remains after cleanup: %#v", config.Skills.Paths)
	}
}

func assertInstalledSkillConfig(t *testing.T, w installWorkflow) {
	t.Helper()
	var config struct {
		Model  string `json:"model"`
		Theme  string `json:"theme"`
		Skills struct {
			Paths []string `json:"paths"`
		} `json:"skills"`
	}
	if err := json.Unmarshal(readWorkflowFile(t, w.config), &config); err != nil {
		t.Fatal(err)
	}
	if config.Model != "openai/gpt-5.6" || config.Theme != "system" || len(config.Skills.Paths) != 1 || config.Skills.Paths[0] != filepath.ToSlash(w.root) {
		t.Fatalf("unexpected config: %#v", config)
	}
}

func TestInstalledBinarySkillDoesNotOverrideUnownedResource(t *testing.T) {
	w, _ := newSkillWorkflow(t)
	target := filepath.Join(w.root, "SKILL.md")
	if err := os.WriteFile(target, []byte("user-owned"), 0o600); err != nil {
		t.Fatal(err)
	}
	result := w.run(t)
	var plan install.Plan
	if err := json.Unmarshal([]byte(result.stdout), &plan); err != nil {
		t.Fatal(err)
	}
	if result.err == nil || plan.Status != install.StatusBlocked {
		t.Fatalf("unowned skill was not blocked: %#v %v", plan, result.err)
	}
	assertWorkflowBytes(t, target, []byte("user-owned"))
	assertWorkflowBytes(t, w.config, []byte(w.original))
}

func newSkillWorkflow(t *testing.T) (installWorkflow, string) {
	t.Helper()
	w := newInstallWorkflow(t, "opencode@1.18.31", "openai", "gpt-5.6", "{\"model\":\"openai/old\",\"theme\":\"system\"}\n", "")
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
