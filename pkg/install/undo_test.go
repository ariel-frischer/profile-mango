package install

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type undoTargetCase struct {
	name     string
	bindings string
	original string
}

func undoTargetCases() map[string]undoTargetCase {
	openAI := "routes:\n  primary:\n    provider: openai\n    transport: native\n    authentication: oauth\n    model: gpt-5.6\n    effort: high\n"
	anthropic := "routes:\n  primary:\n    provider: anthropic\n    transport: native\n    authentication: oauth\n    model: claude-sonnet-4-5\n    effort: high\n"
	return map[string]undoTargetCase{
		"claude-code": {"claude-code", anthropic, "{\n  \"model\": \"old/model\",\n  \"permissions\": {\"allow\": [\"Bash(ls)\"]}\n}\n"},
		"codex":       {"codex", openAI, "# keep\nunknown = true\n[features]\napps = false\n"},
		"pi":          {"pi", openAI, "{\n  \"theme\": \"dark\"\n}\n"},
		"oh-my-pi":    {"oh-my-pi", openAI, "# keep\ntheme: dark\n"},
		"openclaw":    {"openclaw", openAI, "{\n  \"gateway\": {\"port\": 1234}\n}\n"},
		"hermes":      {"hermes", openAI, "# keep\nmodel:\n  provider: old\n  default: old-model\nagent:\n  reasoning_effort: low\n"},
		"opencode":    {"opencode", openAI, "{\n  // keep\n  \"theme\": \"dark\"\n}\n"},
	}
}

func (test undoTargetCase) target() Target {
	target, err := DefaultRegistry().ResolveTarget(test.name)
	if err != nil {
		panic(err)
	}
	return target
}

// installAtDefault installs one target at its documented default path under a synthetic home.
func installAtDefault(t *testing.T, test undoTargetCase, original *string) (string, string) {
	t.Helper()
	request, root := testRequest(t, DefaultRegistry())
	writeInstallTestFile(t, request.BindingsPath, test.bindings)
	request.Override = true
	request.Env = syntheticPathEnv(t, filepath.Join(root, "home"), nil)
	request.Targets = []TargetRequest{{Target: test.target()}}
	config, err := DefaultRegistry().DefaultConfigPath(test.target(), request.Env)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Dir(config), 0o700); err != nil {
		t.Fatal(err)
	}
	if original != nil {
		if err := os.WriteFile(config, []byte(*original), 0o640); err != nil {
			t.Fatal(err)
		}
	}
	plan, err := BuildPlan(request)
	if err != nil || plan.Status != StatusReady {
		t.Fatalf("install plan status=%s err=%v diagnostics=%v", plan.Status, err, plan.Diagnostics)
	}
	if _, err := ApplyPlan(plan, ApplyOptions{ExpectedPlanID: plan.PlanID}); err != nil {
		t.Fatal(err)
	}
	return config, plan.PlanID
}

func applyUndo(t *testing.T, request UndoRequest) RestorePlan {
	t.Helper()
	plan, err := BuildUndoPlan(request)
	if err != nil {
		t.Fatal(err)
	}
	if err := ApplyRestorePlan(plan, plan.PlanID); err != nil {
		t.Fatal(err)
	}
	return plan
}

func TestUndoRestoresOriginalForEveryInstallableTarget(t *testing.T) {
	for name, test := range undoTargetCases() {
		t.Run(name, func(t *testing.T) {
			config, id := installAtDefault(t, test, &test.original)
			plan := applyUndo(t, UndoRequest{Target: test.target(), ConfigPath: config})
			if plan.OriginalPlanID != id {
				t.Fatalf("undo selected %s, want latest install %s", plan.OriginalPlanID, id)
			}
			assertInstallTestFile(t, config, test.original)
			if info, err := os.Stat(config); err != nil || info.Mode().Perm() != 0o640 {
				t.Fatalf("mode = %v, err = %v", info, err)
			}
			if _, err := os.Lstat(config + ".profile-mango.manifest.json"); !os.IsNotExist(err) {
				t.Fatalf("install-created manifest survived undo: %v", err)
			}
			if _, err := BuildUndoPlan(UndoRequest{Target: test.target(), ConfigPath: config}); err == nil {
				t.Fatal("second undo of the same install accepted")
			}
		})
	}
}

func TestUndoRemovesConfigCreatedByInstallForEveryTarget(t *testing.T) {
	for name, test := range undoTargetCases() {
		t.Run(name, func(t *testing.T) {
			config, _ := installAtDefault(t, test, nil)
			applyUndo(t, UndoRequest{Target: test.target(), ConfigPath: config})
			for _, path := range []string{config, config + ".profile-mango.manifest.json"} {
				if _, err := os.Lstat(path); !os.IsNotExist(err) {
					t.Fatalf("install-created %s survived undo: %v", path, err)
				}
			}
		})
	}
}

func TestUndoRefusesDriftUnlessOverride(t *testing.T) {
	for name, test := range undoTargetCases() {
		t.Run(name, func(t *testing.T) {
			config, _ := installAtDefault(t, test, &test.original)
			installed, err := os.ReadFile(config)
			if err != nil {
				t.Fatal(err)
			}
			edited := string(installed) + "\n# edited after install\n"
			writeInstallTestFile(t, config, edited)
			request := UndoRequest{Target: test.target(), ConfigPath: config}
			_, err = BuildUndoPlan(request)
			var drift *DriftError
			if !errors.As(err, &drift) || !strings.Contains(err.Error(), "+# edited after install") && !strings.Contains(err.Error(), "-# edited after install") {
				t.Fatalf("drift error = %v", err)
			}
			assertInstallTestFile(t, config, edited)
			request.Override = true
			applyUndo(t, request)
			assertInstallTestFile(t, config, test.original)
		})
	}
}

func TestUndoRefusesEditedManifestEvenWithOverride(t *testing.T) {
	test := undoTargetCases()["claude-code"]
	config, _ := installAtDefault(t, test, &test.original)
	manifest := config + ".profile-mango.manifest.json"
	data, err := os.ReadFile(manifest)
	if err != nil {
		t.Fatal(err)
	}
	writeInstallTestFile(t, manifest, string(data)+"\n")
	if _, err := BuildUndoPlan(UndoRequest{Target: test.target(), ConfigPath: config, Override: true}); err == nil {
		t.Fatal("edited manifest accepted")
	}
}

func TestUndoStepsBackThroughSuccessiveInstalls(t *testing.T) {
	test := undoTargetCases()["codex"]
	config, _ := installAtDefault(t, test, &test.original)
	first, err := os.ReadFile(config)
	if err != nil {
		t.Fatal(err)
	}
	test.bindings = strings.Replace(test.bindings, "gpt-5.6", "gpt-5.6-mini", 1)
	request, _ := testRequest(t, DefaultRegistry())
	writeInstallTestFile(t, request.BindingsPath, test.bindings)
	request.Override = true
	request.Targets = []TargetRequest{{Target: test.target(), ConfigPath: config}}
	plan, err := BuildPlan(request)
	if err != nil || plan.Status != StatusReady {
		t.Fatalf("second install status=%s err=%v", plan.Status, err)
	}
	if _, err := ApplyPlan(plan, ApplyOptions{ExpectedPlanID: plan.PlanID}); err != nil {
		t.Fatal(err)
	}
	undone := applyUndo(t, UndoRequest{Target: test.target(), ConfigPath: config})
	if undone.OriginalPlanID != plan.PlanID {
		t.Fatalf("first undo selected %s, want %s", undone.OriginalPlanID, plan.PlanID)
	}
	assertInstallTestFile(t, config, string(first))
	applyUndo(t, UndoRequest{Target: test.target(), ConfigPath: config})
	assertInstallTestFile(t, config, test.original)
}

func TestUndoWithoutJournalExplainsNothingToUndo(t *testing.T) {
	root := t.TempDir()
	config := filepath.Join(root, "settings.json")
	writeInstallTestFile(t, config, "{}\n")
	_, err := BuildUndoPlan(UndoRequest{Target: undoTargetCases()["claude-code"].target(), ConfigPath: config})
	if err == nil {
		t.Fatal("undo without an install accepted")
	}
}

func TestUnifiedDiffRedactsSensitiveLines(t *testing.T) {
	diff := unifiedDiff("/x/config", []byte("a\napi_key = \"SECRET\"\nz\n"), []byte("a\nmodel = \"m\"\nz\n"))
	if strings.Contains(diff, "SECRET") || !strings.Contains(diff, "+model = \"m\"") || !strings.Contains(diff, "--- /x/config") {
		t.Fatalf("diff = %q", diff)
	}
}

func TestUndoOfMultiTargetInstallLeavesOtherTargets(t *testing.T) {
	request, root := testRequest(t, DefaultRegistry())
	writeInstallTestFile(t, request.BindingsPath, undoTargetCases()["codex"].bindings)
	request.Override = true
	codexConfig, piConfig := filepath.Join(root, "a-config.toml"), filepath.Join(root, "b-settings.json")
	writeInstallTestFile(t, codexConfig, "# codex original\n")
	writeInstallTestFile(t, piConfig, "{}\n")
	codexTarget, piTarget := undoTargetCases()["codex"].target(), undoTargetCases()["pi"].target()
	request.Targets = []TargetRequest{{Target: codexTarget, ConfigPath: codexConfig}, {Target: piTarget, ConfigPath: piConfig}}
	plan, err := BuildPlan(request)
	if err != nil || plan.Status != StatusReady {
		t.Fatalf("plan status=%s err=%v", plan.Status, err)
	}
	if _, err := ApplyPlan(plan, ApplyOptions{ExpectedPlanID: plan.PlanID}); err != nil {
		t.Fatal(err)
	}
	installedPi, err := os.ReadFile(piConfig)
	if err != nil {
		t.Fatal(err)
	}
	// The shared journal sits beside the lexically first path, so only that target is discoverable.
	if _, err := BuildUndoPlan(UndoRequest{Target: piTarget, ConfigPath: piConfig}); err == nil {
		t.Fatal("undo found a journal that is not beside the pi config")
	}
	applyUndo(t, UndoRequest{Target: codexTarget, ConfigPath: codexConfig})
	assertInstallTestFile(t, codexConfig, "# codex original\n")
	assertInstallTestFile(t, piConfig, string(installedPi))
}
