package install

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const ohMyPiNamedBase = "# keep\nmodelRoles:\n  default: old/model # mine\n  smol: old/fast:low\ndefaultThinkingLevel: low\nunknown:\n  apiKey: SYNTHETIC\n"

// ohMyPiNamedRequest plans named overlays beside a synthetic config.yml, with
// request.Default left false so tests cover the overlay-only install.
func ohMyPiNamedRequest(t *testing.T, names ...string) (Request, string) {
	t.Helper()
	request, _ := ohMyPiTestRequest(t)
	request.Default = false
	for _, name := range names {
		if err := os.MkdirAll(filepath.Join(request.ProfilesRoot, name), 0o755); err != nil {
			t.Fatal(err)
		}
		writeInstallTestFile(t, filepath.Join(request.ProfilesRoot, name, "profile.yaml"), "apiVersion: profilemango.dev/v1alpha1\nkind: PolicyProfile\nmetadata:\n  name: "+name+"\nspec:\n  routeRef: primary\n")
	}
	config := request.Targets[0].ConfigPath
	writeInstallTestFile(t, config, ohMyPiNamedBase)
	return request, config
}

func ohMyPiOverlay(config, name string) string {
	return filepath.Join(filepath.Dir(config), "profiles", name+".yml")
}

func TestOhMyPiNamedProfilesWriteOverlaysAndKeepConfig(t *testing.T) {
	request, config := ohMyPiNamedRequest(t, "coding", "review")
	writeInstallTestFile(t, request.BindingsPath, ohMyPiRolesBindings)
	plan := applyNamed(t, request, "coding")
	want := InstallMode{Mode: InstallModeNamedProfile, ProfileName: "coding", UseCommand: "omp --config " + ShellQuote(ohMyPiOverlay(config, "coding"))}
	if got := plan.Targets[0].Install; got == nil || *got != want {
		t.Fatalf("install mode = %#v, want %#v", got, want)
	}
	applyNamed(t, request, "review")
	assertInstallTestFile(t, config, ohMyPiNamedBase)
	overlay := "modelRoles:\n  default: \"anthropic/claude-opus-5-5:medium\"\n  commit: \"opencode-go/glm-5.3-flash:low\"\n  plan: \"anthropic/claude-opus-5-5:high\"\n" +
		"  slow: \"anthropic/claude-opus-5-5:high\"\n  smol: \"opencode-go/gpt-6-luna:high\"\n  task: \"anthropic/claude-opus-5-5:medium\"\n  tiny: \"opencode-go/glm-5.3-flash:low\"\ntask:\n  maxEffort: \"high\"\n"
	for _, name := range []string{"coding", "review"} {
		assertInstallTestFile(t, ohMyPiOverlay(config, name), overlay+ohMyPiRolesPresetGolden)
	}
	for _, name := range []string{"review", "coding"} {
		request.ProfileName = name
		again, err := BuildPlan(request)
		if err != nil || again.Status != StatusNoop {
			t.Fatalf("reinstall %s status=%s err=%v", name, again.Status, err)
		}
	}
}

func TestOhMyPiNamedDefaultWritesBothAndUndoRestoresBytes(t *testing.T) {
	request, config := ohMyPiNamedRequest(t, "coding")
	request.Default = true
	plan := applyNamed(t, request, "coding")
	if !plan.Targets[0].Install.SetsDefault {
		t.Fatalf("install mode = %#v", plan.Targets[0].Install)
	}
	assertInstallTestFile(t, config, strings.Replace(ohMyPiNamedBase, "old/model", "\"openai/gpt-5.6:high\"", 1)+ohMyPiPrimaryPresetGolden)
	assertInstallTestFile(t, ohMyPiOverlay(config, "coding"), "modelRoles:\n  default: \"openai/gpt-5.6:high\"\n")
	applyUndo(t, UndoRequest{Target: request.Targets[0].Target, ConfigPath: config, Registry: request.Registry, StateDir: testStateDir})
	assertInstallTestFile(t, config, ohMyPiNamedBase)
	for _, path := range []string{ohMyPiOverlay(config, "coding"), config + manifestSuffix} {
		if _, err := os.Lstat(path); !os.IsNotExist(err) {
			t.Fatalf("%s survived undo: %v", filepath.Base(path), err)
		}
	}
}

// A reinstall with a new route rewrites the owned overlay; undo gives back its
// previous bytes, and a second undo removes it.
func TestOhMyPiNamedUndoRestoresPreviousOverlay(t *testing.T) {
	request, config := ohMyPiNamedRequest(t, "coding")
	applyNamed(t, request, "coding")
	overlay := ohMyPiOverlay(config, "coding")
	first, err := os.ReadFile(overlay)
	if err != nil {
		t.Fatal(err)
	}
	writeInstallTestFile(t, request.BindingsPath, ohMyPiRolesBindings)
	applyNamed(t, request, "coding")
	applyUndo(t, UndoRequest{Target: request.Targets[0].Target, ConfigPath: config, Registry: request.Registry, StateDir: testStateDir})
	assertInstallTestFile(t, overlay, string(first))
	applyUndo(t, UndoRequest{Target: request.Targets[0].Target, ConfigPath: config, Registry: request.Registry, StateDir: testStateDir})
	if _, err := os.Lstat(overlay); !os.IsNotExist(err) {
		t.Fatalf("overlay survived second undo: %v", err)
	}
	assertInstallTestFile(t, config, ohMyPiNamedBase)
}

func TestOhMyPiNamedProfileFileRejectsUnusableNames(t *testing.T) {
	if _, err := (ohMyPiAdapter{}).NamedProfileFile("has.dot"); err == nil || !strings.Contains(err.Error(), "letters, digits") {
		t.Fatalf("NamedProfileFile error = %v", err)
	}
}
