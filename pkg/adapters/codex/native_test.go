package codex

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// TestNativeCodexConfigConsumption proves only exact-version TOML parsing and a feature sentinel.
// It does not claim route selection, authentication, precedence, or policy enforcement.
func TestNativeCodexConfigConsumption(t *testing.T) {
	if os.Getenv("PROFILE_MANGO_CODEX_NATIVE_TEST") != "1" {
		t.Skip("set PROFILE_MANGO_CODEX_NATIVE_TEST=1 for the pinned disposable probe")
	}
	bin := requiredNativePath(t, "PROFILE_MANGO_CODEX_BIN")
	if got := nativeSHA256(t, bin); got != EvidenceSHA256 {
		t.Fatalf("Codex binary hash = %s, want %s", got, EvidenceSHA256)
	}
	root := nativeTestRoot(t)
	writeNativeConfig(t, root, validNativeConfig(t))
	version, err := runNativeCodex(root, bin, "--version")
	if err != nil || strings.TrimSpace(version) != "codex-cli "+TargetVersion {
		t.Fatalf("version = %q, err=%v", version, err)
	}
	features, err := runNativeCodex(root, bin, "features", "list")
	if err != nil || !nativeFeatureDisabled(features, "apps") {
		t.Fatalf("valid config was not consumed: err=%v output=%s", err, features)
	}
	testNativeInvalidConfigs(t, root, bin)
}

func requiredNativePath(t *testing.T, name string) string {
	t.Helper()
	path := os.Getenv(name)
	if path == "" || !filepath.IsAbs(path) {
		t.Fatalf("%s must be an absolute path", name)
	}
	info, err := os.Stat(path)
	if err != nil || !info.Mode().IsRegular() || info.Mode()&0o111 == 0 {
		t.Fatalf("%s is not an executable regular file: %s", name, path)
	}
	return path
}

func nativeSHA256(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

func nativeTestRoot(t *testing.T) string {
	t.Helper()
	root := os.Getenv("PROFILE_MANGO_CODEX_NATIVE_ROOT")
	if root == "" || !filepath.IsAbs(root) {
		t.Fatal("PROFILE_MANGO_CODEX_NATIVE_ROOT must be an absolute task-owned directory")
	}
	if err := os.MkdirAll(root, 0o700); err != nil {
		t.Fatal(err)
	}
	runRoot, err := os.MkdirTemp(root, "codex-native-")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(runRoot) })
	for _, name := range []string{"home", "xdg", "codex", "project", "runtime"} {
		if err := os.Mkdir(filepath.Join(runRoot, name), 0o700); err != nil {
			t.Fatal(err)
		}
	}
	return runRoot
}

func validNativeConfig(t *testing.T) []byte {
	t.Helper()
	patch, err := PatchConfig(nil, installRoute())
	if err != nil {
		t.Fatal(err)
	}
	return append(patch.Content, []byte("\n[features]\napps = false\n")...)
}

func writeNativeConfig(t *testing.T, root string, content []byte) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(root, "codex", "config.toml"), content, 0o600); err != nil {
		t.Fatal(err)
	}
}

func nativeFeatureDisabled(output, name string) bool {
	for _, line := range strings.Split(output, "\n") {
		fields := strings.Fields(line)
		if len(fields) > 1 && fields[0] == name && fields[len(fields)-1] == "false" {
			return true
		}
	}
	return false
}

func testNativeInvalidConfigs(t *testing.T, root, bin string) {
	t.Helper()
	invalid := map[string]string{
		"model_provider":         "model_provider = 42\nmodel = \"sentinel-model\"\nmodel_reasoning_effort = \"high\"\n",
		"model":                  "model_provider = \"openai\"\nmodel = 42\nmodel_reasoning_effort = \"high\"\n",
		"model_reasoning_effort": "model_provider = \"openai\"\nmodel = \"sentinel-model\"\nmodel_reasoning_effort = 42\n",
	}
	for field, content := range invalid {
		writeNativeConfig(t, root, []byte(content))
		output, err := runNativeCodex(root, bin, "features", "list")
		if err == nil || !strings.Contains(output, field) {
			t.Fatalf("invalid %s config result: err=%v output=%s", field, err, output)
		}
	}
	writeNativeConfig(t, root, []byte("[features\n"))
	output, err := runNativeCodex(root, bin, "features", "list")
	if err == nil || !strings.Contains(output, "TOML parse error") {
		t.Fatalf("malformed config result: err=%v output=%s", err, output)
	}
}

func runNativeCodex(root, bin string, command ...string) (string, error) {
	bwrap, err := exec.LookPath("bwrap")
	if err != nil {
		return "", fmt.Errorf("find bubblewrap: %w", err)
	}
	args := nativeSandboxArgs(root, bin)
	args = append(args, command...)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	process := exec.CommandContext(ctx, bwrap, args...)
	process.Env = []string{"PATH=/usr/bin:/bin"}
	output, err := process.CombinedOutput()
	if ctx.Err() != nil {
		return string(output), ctx.Err()
	}
	return string(output), err
}

func nativeSandboxArgs(root, bin string) []string {
	return []string{
		"--die-with-parent", "--new-session", "--unshare-net", "--unshare-pid",
		"--unshare-uts", "--unshare-ipc", "--cap-drop", "ALL",
		"--ro-bind", "/usr", "/usr", "--ro-bind", "/bin", "/bin",
		"--ro-bind", "/lib", "/lib", "--ro-bind", "/lib64", "/lib64",
		"--ro-bind", "/etc", "/etc", "--dir", "/probe",
		"--ro-bind", bin, "/probe/codex", "--bind", root, "/state",
		"--proc", "/proc", "--dev", "/dev", "--tmpfs", "/tmp",
		"--chdir", "/state/project", "/usr/bin/env", "-i",
		"PATH=/usr/bin:/bin", "HOME=/state/home", "XDG_CONFIG_HOME=/state/xdg",
		"CODEX_HOME=/state/codex", "NO_COLOR=1", "TERM=dumb", "/probe/codex",
	}
}
