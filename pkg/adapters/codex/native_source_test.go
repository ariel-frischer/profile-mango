package codex

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
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

// TestNativeCodexSourceQualification is opt-in and requires a separately prepared
// exact-source test binary plus identity manifest. It never builds or acquires native code.
func TestNativeCodexSourceQualification(t *testing.T) {
	if os.Getenv(nativeSourceTestEnabledEnv) != "1" {
		t.Skip("set PROFILE_MANGO_CODEX_SOURCE_TEST=1 for the pinned disposable source probe")
	}
	binary := requiredNativePath(t, nativeSourceTestBinaryEnv)
	identity := readNativeSourceIdentity(t)
	archive := requiredAbsoluteRegularFile(t, nativeSourceArchiveEnv)
	verifyNativeSourceIdentity(t, binary, archive, identity)
	fixture := nativeInstallerFixture(t, installRoute().Model)
	if got := hashBytes(fixture); got != identity.FixtureSHA256 {
		t.Fatalf("PatchConfig fixture SHA-256 = %s, want %s", got, identity.FixtureSHA256)
	}
	root := nativeSourceRoot(t)
	fixturePath := filepath.Join(root, "fixture.toml")
	writeSourceFixture(t, fixturePath, fixture)
	before := nativeSourceInventory(t, root, fixturePath)
	output, err := runNativeSourceProbe(root, binary)
	if err != nil || !strings.Contains(output, "1 passed") {
		t.Fatalf("source probe failed: err=%v output=%s", err, output)
	}
	dynamic := nativeInstallerFixture(t, "profile-mango-dynamic-model")
	writeSourceFixture(t, fixturePath, dynamic)
	if output, err = runNativeSourceProbe(root, binary); err == nil {
		t.Fatalf("source probe ignored runtime fixture replacement: output=%s", output)
	}
	if after := nativeSourceInventory(t, root, fixturePath); !reflect.DeepEqual(before, after) {
		t.Fatalf("source probe changed task state: before=%#v after=%#v", before, after)
	}
	if _, err := os.Stat(filepath.Join(root, "nonexistent-codex-home")); !os.IsNotExist(err) {
		t.Fatalf("synthetic CODEX_HOME was created: %v", err)
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

func nativeSourceInventory(t *testing.T, root, excluded string) map[string]string {
	t.Helper()
	inventory := make(map[string]string)
	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if path == excluded {
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
	process.Env = nil
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
