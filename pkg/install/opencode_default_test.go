package install

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestOpenCodeDefaultInstallCreatesNamedAgentAndLeavesMainConfig covers an install without
// --agent or --config: it writes the Mango-owned primary agent at agents/<profile>.md beside
// the documented default config, reports the named-profile install mode and use-it command,
// and leaves the main config (which does not even exist yet) untouched.
func TestOpenCodeDefaultInstallCreatesNamedAgentAndLeavesMainConfig(t *testing.T) {
	request, _ := openCodeTestRequest(t)
	config := request.Targets[0].ConfigPath
	plan, err := BuildPlan(request)
	if err != nil {
		t.Fatal(err)
	}
	want := InstallMode{Mode: InstallModeNamedProfile, ProfileName: "route-only", UseCommand: "opencode --agent route-only"}
	if got := plan.Targets[0].Install; got == nil || *got != want {
		t.Fatalf("install mode = %#v, want %#v", got, want)
	}
	if plan.Status != StatusReady || !hasFileAction(plan.Targets[0], ActionCreate) {
		t.Fatalf("plan = %#v", plan.Targets[0])
	}
	if _, err := ApplyPlan(plan, ApplyOptions{ExpectedPlanID: plan.PlanID}); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(config); !os.IsNotExist(err) {
		t.Fatalf("main config was created without --default: %v", err)
	}
	agentFile := filepath.Join(filepath.Dir(config), "agents", "route-only.md")
	data, err := os.ReadFile(agentFile)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "mode: primary") || !strings.Contains(string(data), `model: "openai/gpt-5.6"`) {
		t.Fatalf("agent definition = %s", data)
	}
	reapply, err := BuildPlan(request)
	if err != nil || reapply.Status != StatusNoop {
		t.Fatalf("reapply status=%s err=%v", reapply.Status, err)
	}
}

// TestOpenCodeDefaultFlagAlsoWritesMainConfigModel covers --default: it keeps writing the
// named agent file and additionally patches the exact top-level model field into the main
// config, exactly as the old default-config install did.
func TestOpenCodeDefaultFlagAlsoWritesMainConfigModel(t *testing.T) {
	request, _ := openCodeTestRequest(t)
	request.Default = true
	config := request.Targets[0].ConfigPath
	before := "{\n  // keep\n  \"model\": \"old/model\",\n  \"unknown\": true,\n}\n"
	writeInstallTestFile(t, config, before)
	plan, err := BuildPlan(request)
	if err != nil {
		t.Fatal(err)
	}
	if !plan.Targets[0].Install.SetsDefault {
		t.Fatalf("install mode = %#v", plan.Targets[0].Install)
	}
	if _, err := ApplyPlan(plan, ApplyOptions{ExpectedPlanID: plan.PlanID}); err != nil {
		t.Fatal(err)
	}
	want := strings.Replace(before, `"old/model"`, `"openai/gpt-5.6"`, 1)
	assertInstallTestFile(t, config, want)
	agentData := mustReadAgentFile(t, filepath.Join(filepath.Dir(config), "agents", "route-only.md"))
	if !strings.Contains(string(agentData), `model: "openai/gpt-5.6"`) {
		t.Fatalf("agent definition = %s", agentData)
	}
}

// TestOpenCodeDefaultUndoRemovesOnlyTheNamedAgent covers undo of a plain default install
// (no --default): it removes just the agent file and its manifest, since the main config was
// never touched.
func TestOpenCodeDefaultUndoRemovesOnlyTheNamedAgent(t *testing.T) {
	request, _ := openCodeTestRequest(t)
	config := request.Targets[0].ConfigPath
	agentFile := filepath.Join(filepath.Dir(config), "agents", "route-only.md")
	plan, err := BuildPlan(request)
	if err != nil || plan.Status != StatusReady {
		t.Fatalf("plan status=%s err=%v", plan.Status, err)
	}
	if _, err := ApplyPlan(plan, ApplyOptions{ExpectedPlanID: plan.PlanID}); err != nil {
		t.Fatal(err)
	}
	applyUndo(t, UndoRequest{Target: request.Targets[0].Target, ConfigPath: config, Registry: request.Registry, StateDir: testStateDir})
	if _, err := os.Lstat(agentFile); !os.IsNotExist(err) {
		t.Fatalf("agent file survived undo: %v", err)
	}
	if _, err := os.Lstat(config + manifestSuffix); !os.IsNotExist(err) {
		t.Fatalf("manifest survived undo: %v", err)
	}
	if _, err := os.Stat(config); !os.IsNotExist(err) {
		t.Fatalf("main config exists after an install that never wrote it: %v", err)
	}
}

// TestOpenCodeDefaultRejectsEditedNamedAgent covers the same edited-file conflict protection
// as the explicit --agent destination: an unowned or externally edited agents/<profile>.md is
// never silently overwritten.
func TestOpenCodeDefaultRejectsEditedNamedAgent(t *testing.T) {
	request, _ := openCodeTestRequest(t)
	config := request.Targets[0].ConfigPath
	agentFile := filepath.Join(filepath.Dir(config), "agents", "route-only.md")
	if err := os.MkdirAll(filepath.Dir(agentFile), 0o755); err != nil {
		t.Fatal(err)
	}
	writeInstallTestFile(t, agentFile, "user-owned agent\n")
	plan, err := BuildPlan(request)
	if err != nil {
		t.Fatal(err)
	}
	if plan.Status != StatusBlocked || plan.Targets[0].Status != StatusConflict {
		t.Fatalf("edited agent file was not protected: %#v", plan.Targets[0])
	}
	assertInstallTestFile(t, agentFile, "user-owned agent\n")
}

// Profile names are simple lowercase letters, digits and '-'; unusable OpenCode agent names
// (uppercase, dots) and reserved built-ins must be rejected before any file is read.
func TestOpenCodeNamedProfileFileRejectsUnusableNames(t *testing.T) {
	if _, err := (openCodeAdapter{}).NamedProfileFile("Has-Upper"); err == nil || !strings.Contains(err.Error(), "lowercase") {
		t.Fatalf("NamedProfileFile error = %v", err)
	}
	if _, err := (openCodeAdapter{}).NamedProfileFile("build"); err == nil || !strings.Contains(err.Error(), "reserved") {
		t.Fatalf("NamedProfileFile error = %v", err)
	}
	path, err := (openCodeAdapter{}).NamedProfileFile("coding")
	if err != nil || path != filepath.Join("agents", "coding.md") {
		t.Fatalf("NamedProfileFile = %q, %v", path, err)
	}
}

func mustReadAgentFile(t *testing.T, path string) []byte {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return data
}
