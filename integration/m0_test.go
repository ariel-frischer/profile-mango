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
	testUnsupportedProfile(t, installedBinary, repoRoot, fixtures, bindings, env)
	if _, err := os.Stat(filepath.Join(tempRoot, "home", ".profile-mango")); !os.IsNotExist(err) {
		t.Fatalf("read-only installed commands created profile home: %v", err)
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
		"help":       {args: []string{"--help"}, contains: []string{"profile-mango", "home", "validate", "--home", "--no-color"}, notContain: []string{"config", "--config", "PROFILE_MANGO_CONFIG"}},
		"home":       {args: []string{"home"}, contains: []string{profileHome}},
		"version":    {args: []string{"version", "--plain"}, contains: []string{"profile-mango dev", "go: "}},
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
