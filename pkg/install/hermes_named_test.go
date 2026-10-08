package install

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// hermesNamedRequest plans the named profile below a synthetic <home>/config.yaml.
func hermesNamedRequest(t *testing.T, name string) (Request, string) {
	t.Helper()
	request, config := hermesInstallRequest(t)
	request.Default, request.ProfileName = false, name
	request.Env = syntheticPathEnv(t, t.TempDir(), map[string]string{"HERMES_HOME": filepath.Dir(config)})
	if err := os.MkdirAll(filepath.Join(request.ProfilesRoot, name), 0o755); err != nil {
		t.Fatal(err)
	}
	writeInstallTestFile(t, filepath.Join(request.ProfilesRoot, name, "profile.yaml"), "apiVersion: profilemango.dev/v1alpha1\nkind: PolicyProfile\nmetadata:\n  name: "+name+"\nspec:\n  routeRef: primary\n")
	writeInstallTestFile(t, config, hermesExistingConfig())
	return request, config
}

// hermesProfileConfig is where `hermes -p <name>` reads config below the Hermes home.
func hermesProfileConfig(config, name string) string {
	return filepath.Join(filepath.Dir(config), "profiles", name, "config.yaml")
}

func hermesGeneratedProfileConfig() string {
	return "model:\n  provider: \"openai\"\n  default: \"gpt-5.6\"\nagent:\n  reasoning_effort: \"high\"\n"
}

func TestHermesNamedProfileWritesProfileConfigAndKeepsDefault(t *testing.T) {
	request, config := hermesNamedRequest(t, "coding")
	plan := applyNamed(t, request, "coding")
	want := InstallMode{Mode: InstallModeNamedProfile, ProfileName: "coding", UseCommand: "hermes -p coding"}
	if got := plan.Targets[0].Install; got == nil || *got != want {
		t.Fatalf("install mode = %#v, want %#v", got, want)
	}
	if !hasDiagnostic(plan.Targets[0].Diagnostics, "hermes.install.profile_state_separate") {
		t.Fatalf("plan omits the separate-state note: %v", plan.Targets[0].Diagnostics)
	}
	assertInstallTestFile(t, config, hermesExistingConfig())
	assertInstallTestFile(t, hermesProfileConfig(config, "coding"), hermesGeneratedProfileConfig())
	again, err := BuildPlan(request)
	if err != nil || again.Status != StatusNoop {
		t.Fatalf("reinstall status=%s err=%v", again.Status, err)
	}
}

func TestHermesNamedProfileWithDefaultAlsoPatchesMainConfig(t *testing.T) {
	request, config := hermesNamedRequest(t, "coding")
	request.Default = true
	plan := applyNamed(t, request, "coding")
	if len(plan.Targets[0].Fields) != 6 {
		t.Fatalf("fields = %#v, want 3 named + 3 main", plan.Targets[0].Fields)
	}
	assertInstallTestFile(t, config, hermesPatchedConfig())
	assertInstallTestFile(t, hermesProfileConfig(config, "coding"), hermesGeneratedProfileConfig())
}

func TestHermesNamedUndoRemovesProfileConfig(t *testing.T) {
	tests := map[string]struct {
		preexisting bool
	}{
		"install created the profile dir": {},
		"profile dir already existed":     {preexisting: true},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			request, config := hermesNamedRequest(t, "coding")
			dir := filepath.Dir(hermesProfileConfig(config, "coding"))
			if test.preexisting {
				if err := os.MkdirAll(dir, 0o700); err != nil {
					t.Fatal(err)
				}
			}
			plan := applyNamed(t, request, "coding")
			undone := applyUndo(t, UndoRequest{Target: request.Targets[0].Target, ConfigPath: config, Registry: request.Registry})
			if undone.OriginalPlanID != plan.PlanID {
				t.Fatalf("undo selected %s, want %s", undone.OriginalPlanID, plan.PlanID)
			}
			for _, path := range []string{hermesProfileConfig(config, "coding"), config + manifestSuffix} {
				if _, err := os.Lstat(path); !os.IsNotExist(err) {
					t.Fatalf("%s survived undo: %v", path, err)
				}
			}
			// Hermes treats any profiles/<name> directory as a valid profile.
			if _, err := os.Lstat(dir); os.IsNotExist(err) == test.preexisting {
				t.Fatalf("profile dir exists=%v after undo, want %v", !os.IsNotExist(err), test.preexisting)
			}
			assertInstallTestFile(t, config, hermesExistingConfig())
		})
	}
}

// `hermes -p <name>` resolves profiles under its home (HERMES_HOME, its grandparent when
// it is itself a profile dir, else ~/.hermes), not beside whatever config was given.
func TestHermesNamedProfileRequiresHermesHomeConfig(t *testing.T) {
	tests := map[string]struct {
		home    func(config string) map[string]string
		blocked string
	}{
		"HERMES_HOME is the config dir": {home: func(config string) map[string]string { return map[string]string{"HERMES_HOME": filepath.Dir(config)} }},
		"config outside default home":   {home: func(string) map[string]string { return nil }, blocked: "can only be derived from"},
		"config outside HERMES_HOME": {home: func(config string) map[string]string {
			return map[string]string{"HERMES_HOME": filepath.Join(config, "..", "..", "elsewhere")}
		}, blocked: "can only be derived from"},
		"HERMES_HOME is a profile dir": {home: func(config string) map[string]string {
			return map[string]string{"HERMES_HOME": filepath.Join(filepath.Dir(config), "profiles", "work")}
		}},
		"relative HERMES_HOME": {home: func(string) map[string]string { return map[string]string{"HERMES_HOME": "rel"} }, blocked: "HERMES_HOME must be an absolute path"},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			request, config := hermesNamedRequest(t, "coding")
			request.Env = syntheticPathEnv(t, t.TempDir(), test.home(config))
			plan, err := BuildPlan(request)
			if err != nil {
				t.Fatal(err)
			}
			target := plan.Targets[0]
			if test.blocked == "" {
				if target.Status != StatusReady {
					t.Fatalf("target = %s %q", target.Status, target.Reason)
				}
				return
			}
			if target.Status != StatusBlocked || !hasDiagnostic(target.Diagnostics, "install.named_profile_path_unsafe") || !strings.Contains(target.Reason, test.blocked) {
				t.Fatalf("target = %s %q %v", target.Status, target.Reason, target.Diagnostics)
			}
		})
	}
}

// `hermes -p default` reads the default config, so that profile patches it in place.
func TestHermesDefaultProfileIsTheDefaultConfig(t *testing.T) {
	request, config := hermesNamedRequest(t, "default")
	plan := applyNamed(t, request, "default")
	target := plan.Targets[0]
	if target.Install == nil || target.Install.UseCommand != "hermes -p default" || len(target.Fields) != 3 || hasDiagnostic(target.Diagnostics, "hermes.install.profile_state_separate") {
		t.Fatalf("target = %#v fields=%v", target.Install, target.Fields)
	}
	assertInstallTestFile(t, config, hermesPatchedConfig())
	if _, err := os.Lstat(hermesProfileConfig(config, "default")); !os.IsNotExist(err) {
		t.Fatalf("a separate default profile was written: %v", err)
	}
}

// "tmp" satisfies profile-mango's own project-profile name schema but is one of
// hermes_cli/profiles.py:120 _RESERVED_NAMES, which `hermes -p tmp` refuses.
func TestHermesNamedProfileBlocksReservedName(t *testing.T) {
	request, _ := hermesNamedRequest(t, "tmp")
	plan, err := BuildPlan(request)
	if err != nil {
		t.Fatal(err)
	}
	target := plan.Targets[0]
	if target.Status != StatusBlocked || !hasDiagnostic(target.Diagnostics, "install.named_profile_path_unsafe") || !strings.Contains(target.Reason, "profile names") {
		t.Fatalf("target = %s %q %v", target.Status, target.Reason, target.Diagnostics)
	}
}
