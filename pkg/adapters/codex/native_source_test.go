package codex

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"
)

const (
	nativeSourceTestEnabledEnv = "PROFILE_MANGO_CODEX_SOURCE_TEST"
	nativeSourceTestBinaryEnv  = "PROFILE_MANGO_CODEX_SOURCE_TEST_BIN"
	nativeSourceArchiveEnv     = "PROFILE_MANGO_CODEX_SOURCE_ARCHIVE"
	nativeSourceIdentityEnv    = "PROFILE_MANGO_CODEX_SOURCE_IDENTITY"
	nativeSourceFixtureEnv     = "PROFILE_MANGO_CODEX_SOURCE_FIXTURE"
	nativeSourceTestFilter     = "config::tests::load_profile_mango_route_fixture_into_effective_config"
	nativeDynamicModel         = "profile-mango-dynamic-model"
	nativeSourceCommit         = "6b9826e3aa83b1a5947db50f4332cb9c65f1b340"
	nativeSourceArchiveSHA256  = "848c7ffac62e21b14edc2048d2e5b7c82b31c556afa4b0d305da0f7d5aa794f4"
	nativeSourceFixtureSHA256  = "f3e634765552aa50cb6a42045536ff5808237af2bf9fcc8c01adb0f789f20fd3"
)

type nativeSourceIdentity struct {
	ArchiveSHA256        string `json:"archive_sha256"`
	FixtureSHA256        string `json:"fixture_sha256"`
	TestExecutableSHA256 string `json:"test_executable_sha256"`
}

func TestNativeInstallerFixtureMatchesQualification(t *testing.T) {
	fixture := nativeInstallerFixture(t, installRoute().Model)
	if got := hashBytes(fixture); got != nativeSourceFixtureSHA256 {
		t.Fatalf("PatchConfig fixture SHA-256 = %s, want %s", got, nativeSourceFixtureSHA256)
	}
}

func TestNativeSourceSandboxIsolated(t *testing.T) {
	args := nativeSourceSandboxArgs("/task/state", "/task/source-test")
	if !hasArgument(args, "--unshare-all") || !hasArgument(args, "-i") {
		t.Fatalf("sandbox does not clear namespaces and environment: %#v", args)
	}
	if !hasArgument(args, "CODEX_HOME=/state/nonexistent-codex-home") || hasArgumentPrefix(args, "CODEX_SQLITE_HOME=") {
		t.Fatalf("sandbox CODEX_HOME or CODEX_SQLITE_HOME contract changed: %#v", args)
	}
	for _, forbidden := range []string{"/etc", "/home", "/run", "--share-net"} {
		if hasArgument(args, forbidden) {
			t.Fatalf("sandbox exposes forbidden host surface %q: %#v", forbidden, args)
		}
	}
}

func TestNativeSourceOutputChecks(t *testing.T) {
	tests := map[string]struct {
		output string
		check  func(*testing.T, string)
	}{
		"positive": {
			output: "running 1 test\ntest result: ok. 1 passed; 0 failed; 0 ignored; 0 measured; 2455 filtered out; finished in 0.00s\n",
			check:  assertNativeSourcePositiveOutput,
		},
		"negative": {
			output: "running 1 test\nassertion failed: `(left == right)`\nDiff < left / right > :\n Some(\n<    \"profile-mango-dynamic-model\",\n>    \"gpt-5.6\",\n )\ntest result: FAILED. 0 passed; 1 failed; 0 ignored; 0 measured; 2455 filtered out; finished in 0.00s\n",
			check:  assertNativeSourceNegativeOutput,
		},
		"negative ANSI output": {
			output: "running 1 test\nassertion failed: `(left == right)`\n\x1b[1mDiff\x1b[0m \x1b[31m< left\x1b[0m / \x1b[32mright >\x1b[0m :\n Some(\n\x1b[31m<    \"p\x1b[0m\x1b[1;48;5;52;31mrofile\x1b[0m\x1b[31m-\x1b[0m\x1b[1;48;5;52;31mmango-dynamic-model\x1b[0m\x1b[31m\",\x1b[0m\n\x1b[32m>    \"\x1b[0m\x1b[1;48;5;22;32mg\x1b[0m\x1b[32mp\x1b[0m\x1b[1;48;5;22;32mt\x1b[0m\x1b[32m-\x1b[0m\x1b[1;48;5;22;32m5.6\x1b[0m\x1b[32m\",\x1b[0m\n )\ntest result: FAILED. 0 passed; 1 failed; 0 ignored; 0 measured; 2455 filtered out; finished in 0.00s\n",
			check:  assertNativeSourceNegativeOutput,
		},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			test.check(t, test.output)
		})
	}
}

// TestNativeCodexSourceQualification is opt-in and requires a separately prepared
// exact-source test binary plus identity manifest. It never builds or acquires native code.
func TestNativeCodexSourceQualification(t *testing.T) {
	if os.Getenv(nativeSourceTestEnabledEnv) != "1" {
		t.Skip("set PROFILE_MANGO_CODEX_SOURCE_TEST=1 for the pinned disposable source probe")
	}
	binary, identity := nativeSourceQualificationInputs(t)
	fixture := nativeInstallerFixture(t, installRoute().Model)
	if got := hashBytes(fixture); got != identity.FixtureSHA256 {
		t.Fatalf("PatchConfig fixture SHA-256 = %s, want %s", got, identity.FixtureSHA256)
	}
	runNativeSourceQualification(t, binary, fixture)
}

func nativeSourceQualificationInputs(t *testing.T) (string, nativeSourceIdentity) {
	t.Helper()
	binary := requiredNativePath(t, nativeSourceTestBinaryEnv)
	identity := readNativeSourceIdentity(t)
	archive := requiredAbsoluteRegularFile(t, nativeSourceArchiveEnv)
	verifyNativeSourceIdentity(t, binary, archive, identity)
	return binary, identity
}

func runNativeSourceQualification(t *testing.T, binary string, fixture []byte) {
	t.Helper()
	root := nativeSourceRoot(t)
	fixturePath := filepath.Join(root, "fixture.toml")
	positiveLog := filepath.Join(root, "positive-test.log")
	negativeLog := filepath.Join(root, "negative-test.log")
	writeSourceFixture(t, fixturePath, fixture)
	before := nativeSourceInventory(t, root, fixturePath, positiveLog, negativeLog)
	runNativeSourcePositive(t, root, binary, positiveLog)
	dynamic := nativeInstallerFixture(t, nativeDynamicModel)
	writeSourceFixture(t, fixturePath, dynamic)
	runNativeSourceNegative(t, root, binary, negativeLog)
	if after := nativeSourceInventory(t, root, fixturePath, positiveLog, negativeLog); !reflect.DeepEqual(before, after) {
		t.Fatalf("source probe changed task state: before=%#v after=%#v", before, after)
	}
	if _, err := os.Stat(filepath.Join(root, "nonexistent-codex-home")); !os.IsNotExist(err) {
		t.Fatalf("synthetic CODEX_HOME was created: %v", err)
	}
}

func runNativeSourcePositive(t *testing.T, root, binary, logPath string) {
	t.Helper()
	output, err := runNativeSourceProbe(root, binary)
	writeNativeSourceLog(t, logPath, output)
	if err != nil {
		t.Fatalf("positive source probe failed: err=%v output=%s", err, output)
	}
	assertNativeSourcePositiveOutput(t, output)
}

func runNativeSourceNegative(t *testing.T, root, binary, logPath string) {
	t.Helper()
	output, err := runNativeSourceProbe(root, binary)
	writeNativeSourceLog(t, logPath, output)
	if err == nil {
		t.Fatalf("source probe ignored runtime fixture replacement: output=%s", output)
	}
	var exitError *exec.ExitError
	if !errors.As(err, &exitError) {
		t.Fatalf("negative source probe failed during setup: err=%v output=%s", err, output)
	}
	assertNativeSourceNegativeOutput(t, output)
}

func writeNativeSourceLog(t *testing.T, path, output string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(output), 0o600); err != nil {
		t.Fatalf("write native source probe log: %v", err)
	}
	t.Logf("native source probe log %s:\n%s", filepath.Base(path), output)
}

func assertNativeSourcePositiveOutput(t *testing.T, output string) {
	t.Helper()
	normalized := stripANSISequences(output)
	rejectNativeSourceProbeErrors(t, normalized)
	assertNativeSourceSummary(t, normalized, "ok", 1, 0)
}

func assertNativeSourceNegativeOutput(t *testing.T, output string) {
	t.Helper()
	normalized := stripANSISequences(output)
	rejectNativeSourceProbeErrors(t, normalized)
	assertNativeSourceSummary(t, normalized, "FAILED", 0, 1)
	for _, marker := range []string{"assertion failed: `(left == right)`", "Diff < left / right >", `<    "` + nativeDynamicModel + `"`, `>    "gpt-5.6"`} {
		if !strings.Contains(normalized, marker) {
			t.Fatalf("negative source probe output missing %q: %s", marker, output)
		}
	}
}

func stripANSISequences(output string) string {
	var builder strings.Builder
	for index := 0; index < len(output); index++ {
		if output[index] != 0x1b || index+1 >= len(output) || output[index+1] != '[' {
			builder.WriteByte(output[index])
			continue
		}
		index += 2
		for index < len(output) && (output[index] < 0x40 || output[index] > 0x7e) {
			index++
		}
	}
	return builder.String()
}

func assertNativeSourceSummary(t *testing.T, output, status string, passed, failed int) {
	t.Helper()
	if strings.Count(output, "running 1 test") != 1 {
		t.Fatalf("source probe did not run exactly one test: %s", output)
	}
	summary := ""
	for _, line := range strings.Split(output, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "test result: ") {
			if summary != "" {
				t.Fatalf("source probe emitted multiple summaries: %s", output)
			}
			summary = line
		}
	}
	want := fmt.Sprintf("test result: %s. %d passed; %d failed;", status, passed, failed)
	if !strings.HasPrefix(summary, want) {
		t.Fatalf("source probe summary = %q, want prefix %q", summary, want)
	}
}

func rejectNativeSourceProbeErrors(t *testing.T, output string) {
	t.Helper()
	lower := strings.ToLower(output)
	for _, marker := range []string{
		"timed out", "timeout", "context deadline exceeded", "signal: killed",
		"bwrap:", "failed to execute", "permission denied", "no such file or directory",
	} {
		if strings.Contains(lower, marker) {
			t.Fatalf("source probe output contains setup or timeout failure %q: %s", marker, output)
		}
	}
}

func readNativeSourceIdentity(t *testing.T) nativeSourceIdentity {
	t.Helper()
	path := requiredAbsoluteRegularFile(t, nativeSourceIdentityEnv)
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read source identity: %v", err)
	}
	var identity nativeSourceIdentity
	if err := json.Unmarshal(data, &identity); err != nil {
		t.Fatalf("decode source identity: %v", err)
	}
	if identity.ArchiveSHA256 == "" || identity.FixtureSHA256 == "" || identity.TestExecutableSHA256 == "" {
		t.Fatalf("source identity is incomplete: %#v", identity)
	}
	return identity
}

func verifyNativeSourceIdentity(t *testing.T, binary, archive string, identity nativeSourceIdentity) {
	t.Helper()
	if identity.ArchiveSHA256 != nativeSourceArchiveSHA256 {
		t.Fatalf("source archive identity = %s, want %s", identity.ArchiveSHA256, nativeSourceArchiveSHA256)
	}
	if !strings.Contains(filepath.Base(archive), nativeSourceCommit) {
		t.Fatalf("source archive %q does not identify commit %s", archive, nativeSourceCommit)
	}
	if got := nativeSHA256(t, archive); got != identity.ArchiveSHA256 {
		t.Fatalf("source archive SHA-256 = %s, want %s", got, identity.ArchiveSHA256)
	}
	if got := nativeSHA256(t, binary); got != identity.TestExecutableSHA256 {
		t.Fatalf("source test binary SHA-256 = %s, want %s", got, identity.TestExecutableSHA256)
	}
}

func nativeInstallerFixture(t *testing.T, model string) []byte {
	t.Helper()
	source := []byte("# synthetic installer input, not a personal configuration\nmodel_provider = \"previous-provider\"\nmodel = \"previous-model\"\nmodel_reasoning_effort = \"low\"\n")
	route := installRoute()
	route.Model = model
	patch, err := PatchConfig(source, route)
	if err != nil {
		t.Fatal(err)
	}
	if len(patch.Fields) != 3 {
		t.Fatalf("PatchConfig fields = %#v, want three route fields", patch.Fields)
	}
	return patch.Content
}

func nativeSourceRoot(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	for _, name := range []string{"home", "xdg", "project", "runtime"} {
		if err := os.Mkdir(filepath.Join(root, name), 0o700); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(root, "home", "outside-target-sentinel"), []byte("sentinel\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	return root
}

func writeSourceFixture(t *testing.T, path string, content []byte) {
	t.Helper()
	if err := os.WriteFile(path, content, 0o600); err != nil {
		t.Fatal(err)
	}
}

func nativeSourceInventory(t *testing.T, root string, excluded ...string) map[string]string {
	t.Helper()
	inventory := make(map[string]string)
	excludedPaths := make(map[string]struct{}, len(excluded))
	for _, path := range excluded {
		excludedPaths[path] = struct{}{}
	}
	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if _, skip := excludedPaths[path]; skip {
			return nil
		}
		relative, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		if entry.IsDir() {
			inventory[relative] = "dir"
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		inventory[relative] = hashBytes(data)
		return nil
	})
	if err != nil {
		t.Fatalf("inventory source probe state: %v", err)
	}
	return inventory
}

func runNativeSourceProbe(root, binary string) (string, error) {
	bwrap, err := exec.LookPath("bwrap")
	if err != nil {
		return "", fmt.Errorf("find bubblewrap: %w", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	process := exec.CommandContext(ctx, bwrap, nativeSourceSandboxArgs(root, binary)...)
	process.Env = []string{}
	output, err := process.CombinedOutput()
	if ctx.Err() != nil {
		return string(output), ctx.Err()
	}
	return string(output), err
}

func nativeSourceSandboxArgs(root, binary string) []string {
	return []string{
		"--die-with-parent", "--new-session", "--unshare-all", "--cap-drop", "ALL",
		"--ro-bind", "/usr", "/usr", "--ro-bind", "/bin", "/bin",
		"--ro-bind", "/lib", "/lib", "--ro-bind", "/lib64", "/lib64",
		"--dir", "/probe", "--ro-bind", binary, "/probe/source-test",
		"--bind", root, "/state", "--proc", "/proc", "--dev", "/dev", "--tmpfs", "/tmp",
		"--chdir", "/state/project", "/usr/bin/env", "-i",
		"PATH=/usr/bin:/bin", "HOME=/state/home", "XDG_CONFIG_HOME=/state/xdg",
		"CODEX_HOME=/state/nonexistent-codex-home", nativeSourceFixtureEnv + "=/state/fixture.toml",
		"NO_COLOR=1", "TERM=dumb", "/probe/source-test", nativeSourceTestFilter, "--exact", "--nocapture",
	}
}

func requiredAbsoluteRegularFile(t *testing.T, envName string) string {
	t.Helper()
	path := os.Getenv(envName)
	if path == "" || !filepath.IsAbs(path) {
		t.Fatalf("%s must be an absolute path", envName)
	}
	info, err := os.Stat(path)
	if err != nil || !info.Mode().IsRegular() {
		t.Fatalf("%s is not a regular file: %s", envName, path)
	}
	return path
}

func hashBytes(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

func hasArgument(args []string, wanted string) bool {
	for _, arg := range args {
		if arg == wanted {
			return true
		}
	}
	return false
}

func hasArgumentPrefix(args []string, prefix string) bool {
	for _, arg := range args {
		if strings.HasPrefix(arg, prefix) {
			return true
		}
	}
	return false
}
