package install

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const openClawNamedBase = "// keep\n{agents:{defaults:{model:{primary:\"old/model\"}}},gateway:{port:1234}}\n"

// openClawNamedRequest plans the named profile beside a synthetic <home>/.openclaw/openclaw.json.
func openClawNamedRequest(t *testing.T, name string) (Request, string) {
	t.Helper()
	request, config := openClawInstallRequest(t)
	request.Default, request.ProfileName = false, name
	if err := os.MkdirAll(filepath.Join(request.ProfilesRoot, name), 0o755); err != nil {
		t.Fatal(err)
	}
	writeInstallTestFile(t, filepath.Join(request.ProfilesRoot, name, "profile.yaml"), "route: primary\n")
	writeInstallTestFile(t, config, openClawNamedBase)
	return request, config
}

func TestOpenClawNamedProfileWritesProfileConfigAndKeepsDefault(t *testing.T) {
	request, config := openClawNamedRequest(t, "coding")
	plan := applyNamed(t, request, "coding")
	want := InstallMode{Mode: InstallModeNamedProfile, ProfileName: "coding", UseCommand: "openclaw --profile coding"}
	if got := plan.Targets[0].Install; got == nil || *got != want {
		t.Fatalf("install mode = %#v, want %#v", got, want)
	}
	if !hasDiagnostic(plan.Targets[0].Diagnostics, "openclaw.install.profile_state_separate") {
		t.Fatalf("plan omits the separate-state note: %v", plan.Targets[0].Diagnostics)
	}
	assertInstallTestFile(t, config, openClawNamedBase)
	assertInstallTestFile(t, openClawProfileConfig(config, "coding"), "{agents:{defaults:{model:{primary:\"openai/gpt-5.6\"},thinkingDefault:\"high\"}}}\n")
	again, err := BuildPlan(request)
	if err != nil || again.Status != StatusNoop {
		t.Fatalf("reinstall status=%s err=%v", again.Status, err)
	}
}

func TestOpenClawNamedUndoRemovesProfileConfig(t *testing.T) {
	request, config := openClawNamedRequest(t, "coding")
	plan := applyNamed(t, request, "coding")
	undone := applyUndo(t, UndoRequest{Target: request.Targets[0].Target, ConfigPath: config, Registry: request.Registry})
	if undone.OriginalPlanID != plan.PlanID {
		t.Fatalf("undo selected %s, want %s", undone.OriginalPlanID, plan.PlanID)
	}
	for _, path := range []string{openClawProfileConfig(config, "coding"), config + manifestSuffix} {
		if _, err := os.Lstat(path); !os.IsNotExist(err) {
			t.Fatalf("%s survived undo: %v", path, err)
		}
	}
	assertInstallTestFile(t, config, openClawNamedBase)
}

// An explicit --config must be <home>/.openclaw/openclaw.json so the profile location is known.
func TestOpenClawNamedProfileBlocksUnderivableConfigPath(t *testing.T) {
	tests := map[string]string{
		"relocated file": filepath.Join("cfg", "openclaw.json"),
		"renamed file":   filepath.Join(".openclaw", "custom.json5"),
	}
	for name, relative := range tests {
		t.Run(name, func(t *testing.T) {
			request, _ := openClawNamedRequest(t, "coding")
			config := filepath.Join(t.TempDir(), relative)
			request.Targets[0].ConfigPath = config
			plan, err := BuildPlan(request)
			if err != nil {
				t.Fatal(err)
			}
			target := plan.Targets[0]
			if target.Status != StatusBlocked || !hasDiagnostic(target.Diagnostics, "install.named_profile_path_unsafe") || !strings.Contains(target.Reason, "<home>/.openclaw/openclaw.json") {
				t.Fatalf("target = %s %q %v", target.Status, target.Reason, target.Diagnostics)
			}
		})
	}
}

// `openclaw --profile default` reads the default config, so that profile patches it in place.
func TestOpenClawDefaultProfileIsTheDefaultConfig(t *testing.T) {
	request, config := openClawNamedRequest(t, "default")
	plan := applyNamed(t, request, "default")
	target := plan.Targets[0]
	if target.Install == nil || target.Install.UseCommand != "openclaw --profile default" || len(target.Fields) != 2 || hasDiagnostic(target.Diagnostics, "openclaw.install.profile_state_separate") {
		t.Fatalf("target = %#v fields=%v", target.Install, target.Fields)
	}
	data, err := os.ReadFile(config)
	if err != nil || !strings.Contains(string(data), `primary:"openai/gpt-5.6"`) || !strings.Contains(string(data), "gateway") {
		t.Fatalf("default config = %q err=%v", data, err)
	}
	if _, err := os.Lstat(openClawProfileConfig(config, "default")); !os.IsNotExist(err) {
		t.Fatalf("a separate default profile was written: %v", err)
	}
}
