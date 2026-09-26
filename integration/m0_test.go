package integration_test

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

type commandResult struct {
	stdout string
	stderr string
	err    error
}

func TestCleanOfflineInstallAndInstalledBinary(t *testing.T) {
	repoRoot := absolutePath(t, "..")
	tempRoot := t.TempDir()
	env := cleanInstallEnv(t, tempRoot)
	builtBinary := filepath.Join(tempRoot, "build", executableName())
	runGo(t, repoRoot, env, "build", "-o", builtBinary, "./cmd/profile-mango")
	runGo(t, repoRoot, env, "install", "./cmd/profile-mango")

	installedBinary := filepath.Join(tempRoot, "bin", executableName())
	fixtures := filepath.Join(repoRoot, "pkg", "profilemango", "testdata", "fixtures")
	bindings := filepath.Join(fixtures, "bindings.yaml")
	testInstalledCommands(t, installedBinary, repoRoot, fixtures, bindings, env)
	testInstalledRenderPreviews(t, installedBinary, repoRoot, fixtures, bindings, env)
	testInstalledExampleProfiles(t, installedBinary, repoRoot, env)
	testUnsupportedProfile(t, installedBinary, repoRoot, fixtures, bindings, env)
	if _, err := os.Stat(filepath.Join(tempRoot, "home", ".profile-mango")); !os.IsNotExist(err) {
		t.Fatalf("read-only installed commands created profile home: %v", err)
	}
}

func testInstalledRenderPreviews(t *testing.T, binary, repoRoot, fixtures, bindings string, env []string) {
	t.Helper()
	resourceRoot := filepath.Dir(fixtures)
	targets := map[string]struct {
		version      string
		candidate    string
		blocker      string
		evidenceHash string
	}{
		"claude-code": {version: "2.1.278", candidate: "preview/route-only.settings.json.preview", blocker: "claudecode.config.acceptance_unverified", evidenceHash: "d1fb51ab0a0234d1bd7f418ee9d6b6b124c2412b2ddaf3dfc3256bad8063f1c7"},
		"codex":       {version: "0.157.1", candidate: "preview/route-only.config.toml.preview", blocker: "codex.route.authentication_unverified", evidenceHash: "3e2584f3f3829a43a0495011a1cecb2facbe64a2403e2b682351fd9c2983f970"},
		"pi":          {version: "0.86.1", candidate: "preview/route-only.settings.json.preview", blocker: "pi.config.acceptance_unverified", evidenceHash: "8dff93e6fa03e0d498e72a78d2c7bb5f094f5e06ee268e6abd000ba2984a0b6a"},
		"oh-my-pi":    {version: "18.2.6", candidate: "preview/route-only.config.yml.preview", blocker: "ohmypi.config.inspector_unsafe", evidenceHash: "4d9558530fdd8c76798181545d7cde8b558731596515b2f300e61c8d403bcb6e"},
		"openclaw":    {version: "2026.9.5", candidate: "preview/route-only.config.json5.preview", blocker: "openclaw.config.inspector_unsafe", evidenceHash: "0e15e679795134cf7d488302f2bdaf0682ad4413e19a7f5c6cc22584f03d02a4"},
		"opencode":    {version: "1.18.31", candidate: "preview/route-only.opencode.jsonc.preview", blocker: "opencode.runtime.enforcement_unverified", evidenceHash: "76f69fe27ec2b44e23fa1749029e7c012eb7e975a0f0c7819e9458198dfd3896"},
		"hermes":      {version: "0.21.3", candidate: "preview/route-only.config.yaml.preview", blocker: "hermes.config.inspector_unsafe", evidenceHash: "71f2db39a64fbba282e3bd3be4b0f7b935585948a59a368d61deeec0f0827c47"},
		"jcode-fork":  {version: "0.83.909-dev (ca8017a3a)", candidate: "preview/route-only.config.toml.preview", blocker: "jcodefork.experimental_only", evidenceHash: "392ecafbb9ec20f49e78cf556a8a8bcb9040c54f2f92db7d6e112c0cf70ea992"},
	}
	for name, target := range targets {
		t.Run(name, func(t *testing.T) {
			out := filepath.Join(t.TempDir(), "preview")
			result := runCommand(binary, repoRoot, env, "render", "route-only", "--profiles", fixtures, "--resource-root", resourceRoot, "--bindings", bindings, "--target", name, "--target-version", target.version, "--out", out, "--preview", "--json")
			if result.err != nil {
				t.Fatalf("staged non-applicable preview must exit 0: %v\n%s", result.err, result.stderr)
			}
			var report struct {
				Target        string `json:"target"`
				TargetVersion string `json:"targetVersion"`
				Applicable    bool   `json:"applicable"`
				Preview       bool   `json:"preview"`
				Evidence      struct {
					SHA256 string `json:"sha256"`
				} `json:"evidence"`
			}
			if err := json.Unmarshal([]byte(result.stdout), &report); err != nil {
				t.Fatalf("render report: %v\n%s", err, result.stdout)
			}
			if report.Target != name || report.TargetVersion != target.version || report.Applicable || !report.Preview || report.Evidence.SHA256 != target.evidenceHash {
				t.Fatalf("unexpected report: %#v", report)
			}
			if !strings.Contains(result.stderr, "warning ") || !strings.Contains(result.stderr, target.blocker) {
				t.Fatalf("missing target blocker %q:\n%s", target.blocker, result.stderr)
			}
			assertInstalledFile(t, out, "render.json")
			assertInstalledFile(t, out, target.candidate)
			assertInstalledFile(t, out, "resources/fixtures/route-only/instructions/system.md")
		})
	}
}

func assertInstalledFile(t *testing.T, root, relative string) {
	t.Helper()
	if _, err := os.Stat(filepath.Join(root, filepath.FromSlash(relative))); err != nil {
		t.Fatalf("missing installed render artifact %s: %v", relative, err)
	}
}

func testInstalledCommands(t *testing.T, binary, repoRoot, fixtures, bindings string, env []string) {
	t.Helper()
	profileHome := filepath.Join(filepath.Dir(filepath.Dir(binary)), "home", ".profile-mango")
	tests := map[string]struct {
		args       []string
		contains   []string
		notContain []string
	}{
		"help":       {args: []string{"--help"}, contains: []string{"mango", "home", "validate", "--home", "--no-color"}, notContain: []string{"config", "--config", "PROFILE_MANGO_CONFIG"}},
		"home":       {args: []string{"home"}, contains: []string{profileHome}},
		"version":    {args: []string{"version", "--plain"}, contains: []string{"mango dev", "go: "}},
		"route-only": {args: validateArgs(filepath.Join(fixtures, "route-only", "profile.yaml"), bindings), contains: []string{`"valid": true`, `"profileName": "route-only"`}},
		"read-only":  {args: validateArgs(filepath.Join(fixtures, "read-only", "profile.yaml"), bindings), contains: []string{`"valid": true`, `"profileName": "read-only"`}},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			result := runCommand(binary, repoRoot, env, test.args...)
			if result.err != nil {
				t.Fatalf("command failed: %v\nstderr:\n%s", result.err, result.stderr)
			}
			for _, want := range test.contains {
				if !strings.Contains(result.stdout, want) {
					t.Fatalf("stdout missing %q:\n%s", want, result.stdout)
				}
			}
			for _, unwanted := range test.notContain {
				if strings.Contains(result.stdout, unwanted) {
					t.Fatalf("stdout contains removed surface %q:\n%s", unwanted, result.stdout)
				}
			}
		})
	}
}

func testUnsupportedProfile(t *testing.T, binary, repoRoot, fixtures, bindings string, env []string) {
	t.Helper()
	profile := filepath.Join(fixtures, "unsupported", "profile.yaml")
	result := runCommand(binary, repoRoot, env, validateArgs(profile, bindings)...)
	if result.err == nil {
		t.Fatal("unsupported profile succeeded")
	}
	var output struct {
		Valid       bool `json:"valid"`
		Diagnostics []struct {
			Code string `json:"code"`
		} `json:"diagnostics"`
	}
	if err := json.Unmarshal([]byte(result.stdout), &output); err != nil {
		t.Fatalf("decode output: %v\n%s", err, result.stdout)
	}
	if output.Valid || len(output.Diagnostics) != 1 || output.Diagnostics[0].Code != "yaml.strict" {
		t.Fatalf("unstable unsupported diagnostic: %#v", output)
	}
}

func cleanInstallEnv(t *testing.T, root string) []string {
	t.Helper()
	result := runCommand("go", ".", os.Environ(), "env", "GOMODCACHE")
	if result.err != nil {
		t.Fatalf("go env GOMODCACHE: %v", result.err)
	}
	moduleCache := strings.TrimSpace(result.stdout)
	if moduleCache == "" {
		t.Fatal("go env GOMODCACHE returned empty output")
	}
	return replaceEnvironment(os.Environ(), []string{
		"HOME=" + filepath.Join(root, "home"),
		"XDG_CONFIG_HOME=" + filepath.Join(root, "config"),
		"CODEX_HOME=",
		"HERMES_HOME=",
		"PI_CODING_AGENT_DIR=",
		"OPENCLAW_CONFIG_PATH=",
		"GOCACHE=" + filepath.Join(root, "cache"),
		"GOBIN=" + filepath.Join(root, "bin"),
		"GOMODCACHE=" + moduleCache,
		"GOTOOLCHAIN=local",
		"GOPROXY=off",
	})
}

func replaceEnvironment(base, replacements []string) []string {
	replaced := make(map[string]bool, len(replacements))
	for _, entry := range replacements {
		name, _, _ := strings.Cut(entry, "=")
		replaced[name] = true
	}
	env := make([]string, 0, len(base)+len(replacements))
	for _, entry := range base {
		name, _, _ := strings.Cut(entry, "=")
		if !replaced[name] {
			env = append(env, entry)
		}
	}
	return append(env, replacements...)
}

func runGo(t *testing.T, directory string, env []string, args ...string) {
	t.Helper()
	result := runCommand("go", directory, env, args...)
	if result.err != nil {
		t.Fatalf("go %s: %v\nstderr:\n%s", strings.Join(args, " "), result.err, result.stderr)
	}
}

func runCommand(name, directory string, env []string, args ...string) commandResult {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	command := exec.Command(name, args...)
	command.Dir = directory
	command.Env = env
	command.Stdout = &stdout
	command.Stderr = &stderr
	err := command.Run()
	return commandResult{stdout: stdout.String(), stderr: stderr.String(), err: err}
}

func validateArgs(profile, bindings string) []string {
	return []string{"validate", profile, "--bindings", bindings, "--json"}
}

func executableName() string {
	if runtime.GOOS == "windows" {
		return "profile-mango.exe"
	}
	return "profile-mango"
}

func absolutePath(t *testing.T, path string) string {
	t.Helper()
	absolute, err := filepath.Abs(path)
	if err != nil {
		t.Fatal(err)
	}
	return absolute
}
