package install

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestHermesNamedInstallPreservesRecordedDefault(t *testing.T) {
	request, config := hermesNamedRequest(t, "sol-daily")
	request.ProfileName, request.Default = "default", true
	applySwitchTestPlan(t, request)
	before, err := os.ReadFile(config)
	if err != nil {
		t.Fatal(err)
	}
	request.ProfileName, request.Default = "sol-daily", false
	applySwitchTestPlan(t, request)
	assertInstallTestFile(t, config, string(before))
	if got := readSwitchManifest(t, config).Profile; got != "default" {
		t.Fatalf("recorded default = %q, want default", got)
	}
	if got := ownershipStatus(t, request).Profile; got != "default" {
		t.Fatalf("status profile = %q, want default", got)
	}
	request.Default = true
	writeInstallTestFile(t, request.BindingsPath, strings.Replace(hermesBindings(), "effort: high", "effort: low", 1))
	applySwitchTestPlan(t, request)
	if got := readSwitchManifest(t, config).Profile; got != "sol-daily" {
		t.Fatalf("explicit default install recorded %q", got)
	}
}

func TestStatusHermesNamedOwnedFieldsIgnoreSerialization(t *testing.T) {
	cases := map[string]struct {
		content string
		state   string
	}{
		"unquoted values":     {"model:\n  provider: openai\n  default: gpt-5.6\nagent:\n  reasoning_effort: high\n", FileOtherEdits},
		"changed owned value": {"model:\n  provider: openai\n  default: user-model\nagent:\n  reasoning_effort: high\n", FileEdited},
		"missing owned value": {"model:\n  provider: openai\nagent:\n  reasoning_effort: high\n", FileEdited},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			request, config := hermesNamedRequest(t, "sol-daily")
			applySwitchTestPlan(t, request)
			request.ProfileName, request.Default = "default", true
			applySwitchTestPlan(t, request)
			writeInstallTestFile(t, hermesProfileConfig(config, "sol-daily"), tc.content)
			status := ownershipStatus(t, request)
			assertOwnedStatus(t, status, "profiles/sol-daily/config.yaml", tc.state)
		})
	}
}

func TestStatusWholeOwnedFileMatchesSourcesDespiteStaleHash(t *testing.T) {
	for name, target := range map[string]string{"home AGENTS.md": "home", "Claude CLAUDE.md": "claude-code"} {
		t.Run(name, func(t *testing.T) {
			request, agents := staleGlobalRequest(t, target)
			applySwitchTestPlan(t, request)
			manifest := readSwitchManifest(t, request.Targets[0].ConfigPath)
			for index := range manifest.Files {
				if manifest.Files[index].Path == agents {
					manifest.Files[index].SHA256 = strings.Repeat("0", 64)
				}
			}
			data, err := json.Marshal(manifest)
			if err != nil {
				t.Fatal(err)
			}
			writeInstallTestFile(t, request.Targets[0].ConfigPath+".profile-mango.manifest.json", string(data))
			path := relativeOwnedPath(request.Targets[0].ConfigPath, agents, request.Env)
			assertOwnedStatus(t, ownershipStatus(t, request), path, FileInSync)
			writeInstallTestFile(t, agents, "real user edit\n")
			assertOwnedStatus(t, ownershipStatus(t, request), path, FileEdited)
		})
	}
}

func staleGlobalRequest(t *testing.T, target string) (Request, string) {
	t.Helper()
	request, root := piInstallTestRequest(t)
	file := "AGENTS.md"
	if target == "claude-code" {
		request, root = claudeCodeTestRequest(t)
		file = "CLAUDE.md"
	}
	home := filepath.Join(root, "home")
	if err := os.MkdirAll(home, 0o755); err != nil {
		t.Fatal(err)
	}
	request.Env = syntheticPathEnv(t, home, nil)
	request.Default, request.Override = true, false
	writeInstallTestFile(t, filepath.Join(root, "instructions.md"), globalTestWork)
	writeInstallTestFile(t, filepath.Join(request.ProfilesRoot, request.ProfileName, "profile.yaml"), "route: primary\nglobalInstructions:\n  "+target+":\n    "+file+": instructions.md\n")
	if target == "home" {
		return request, filepath.Join(home, file)
	}
	return request, filepath.Join(filepath.Dir(request.Targets[0].ConfigPath), file)
}

func ownershipStatus(t *testing.T, request Request) TargetStatus {
	t.Helper()
	report, err := InspectStatus(StatusRequest{ProfilesRoot: request.ProfilesRoot, ResourceRoot: request.ResourceRoot, BindingsPath: request.BindingsPath, Registry: request.Registry, Env: request.Env, Targets: request.Targets, StateDir: testStateDir})
	if err != nil {
		t.Fatal(err)
	}
	return report.Targets[0]
}

func assertOwnedStatus(t *testing.T, status TargetStatus, path, state string) {
	t.Helper()
	for _, file := range status.Files {
		if file.Path == path {
			if file.State != state {
				t.Fatalf("%s state = %s, want %s", path, file.State, state)
			}
			return
		}
	}
	t.Fatalf("owned file %s absent from %#v", path, status.Files)
}
