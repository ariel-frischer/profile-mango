package integration_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gitlab.com/ariel-frischer/profile-mango/pkg/install"
)

func TestInstalledBinaryNamedOpenCodeAgent(t *testing.T) {
	w := newInstallWorkflow(t, "opencode@1.18.31", "openai", "gpt-5.6", "{\"model\":\"unmanaged\"}\n", "")
	mainConfig := w.config
	w.config = filepath.Join(w.root, "agents", "mango-synthetic.md")
	if err := os.MkdirAll(filepath.Dir(w.config), 0o700); err != nil {
		t.Fatal(err)
	}
	for index, arg := range w.args {
		if strings.HasPrefix(arg, "opencode@1.18.31="+mainConfig) {
			w.args[index] = "opencode@1.18.31=" + w.config
		}
	}
	w.args = append(w.args, "--agent", "opencode@1.18.31=subagent:mango-synthetic")
	profile := filepath.Join(w.root, "profiles", "minimal", "profile.yaml")
	writeWorkflowFile(t, profile, string(readWorkflowFile(t, profile))+"  instructions:\n    append:\n      - instructions/synthetic.md\n")
	if err := os.MkdirAll(filepath.Join(w.root, "instructions"), 0o700); err != nil {
		t.Fatal(err)
	}
	writeWorkflowFile(t, filepath.Join(w.root, "instructions", "synthetic.md"), "Synthetic ordered instruction.\n")
	plan := w.plan(t)
	if plan.Status != install.StatusReady || plan.Targets[0].Agent == nil || plan.Targets[0].Agent.Mode != "subagent" {
		t.Fatalf("plan: %#v", plan)
	}
	if result := w.run(t, "--apply", "--yes", "--expect-plan", plan.PlanID); result.err != nil {
		t.Fatalf("apply: %v %s", result.err, result.stderr)
	}
	text := string(readWorkflowFile(t, w.config))
	if !strings.Contains(text, "Synthetic ordered instruction.") || !strings.Contains(text, `model: "openai/gpt-5.6"`) {
		t.Fatalf("definition: %s", text)
	}
	assertWorkflowBytes(t, mainConfig, []byte(w.original))
	if next := w.plan(t); next.Status != install.StatusNoop {
		t.Fatalf("reapply: %#v", next)
	}
	writeWorkflowFile(t, w.config, "user edit\n")
	if result := w.run(t); result.err == nil {
		t.Fatal("edited definition was accepted despite override")
	}
	assertWorkflowBytes(t, w.config, []byte("user edit\n"))
}

func writeWorkflowFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
}
