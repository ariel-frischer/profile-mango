package pi

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"time"

	"gitlab.com/ariel-frischer/profile-mango/pkg/profilemango"
)

// The probe intentionally imports only Pi's exact settings module. It bypasses
// full startup and therefore cannot establish provider, session, extension,
// hook, tool, permission, or runtime enforcement behavior.
const nativeSettingsProbe = `import { pathToFileURL } from "node:url";
import { join } from "node:path";
const root = process.argv[2];
const project = process.argv[3];
const agent = process.argv[4];
const mode = process.argv[5];
const { SettingsManager } = await import(pathToFileURL(join(root, "dist/core/settings-manager.js")).href);
if (mode === "positive") {
  const global = SettingsManager.create(project + "/absent", agent, { projectTrusted: true });
  if (global.getDefaultProvider() !== "sentinel-provider" || global.getDefaultModel() !== "sentinel-global-model" || global.getDefaultThinkingLevel() !== "high") throw new Error("global route settings not consumed");
}
const manager = SettingsManager.create(project, agent, { projectTrusted: true });
const errors = manager.drainErrors();
if (mode === "positive") {
  const result = { provider: manager.getDefaultProvider(), model: manager.getDefaultModel(), thinking: manager.getDefaultThinkingLevel(), unknown: manager.getGlobalSettings().unknown, projectUnknown: manager.getProjectSettings().projectUnknown, errors: errors.length };
  if (result.provider !== "sentinel-provider" || result.model !== "sentinel-project-model" || result.thinking !== "high" || result.unknown?.keep !== true || result.projectUnknown?.[0] !== "keep" || result.errors !== 0) throw new Error(JSON.stringify(result));
  console.log(JSON.stringify(result));
} else {
  if (errors.length !== 1 || errors[0].scope !== "global" || !errors[0].path.endsWith("/agent/settings.json")) throw new Error(JSON.stringify(errors));
  console.log(JSON.stringify({ malformed: true, scope: errors[0].scope, path: errors[0].path }));
}
`

func TestNativeSettingsModuleQualification(t *testing.T) {
	packageRoot := os.Getenv("PROFILE_MANGO_PI_NATIVE_PACKAGE")
	if packageRoot == "" {
		t.Skip("set PROFILE_MANGO_PI_NATIVE_PACKAGE to the exact 0.86.1 package root for native qualification")
	}
	var err error
	packageRoot, err = filepath.Abs(packageRoot)
	if err != nil {
		t.Fatalf("resolve exact Pi package root: %v", err)
	}
	assertNativePackage(t, packageRoot)
	node, err := exec.LookPath("node")
	if err != nil {
		t.Fatalf("find node: %v", err)
	}
	if _, err := os.Stat("/usr/bin/node"); err == nil {
		node = "/usr/bin/node"
	}
	tests := map[string]struct {
		mode string
	}{
		"positive sentinel and precedence": {mode: "positive"},
		"malformed settings":               {mode: "malformed"},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			runNativeSettingsProbe(t, node, packageRoot, test.mode)
		})
	}
}

func assertNativePackage(t *testing.T, packageRoot string) {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(packageRoot, "package.json"))
	if err != nil {
		t.Fatalf("read exact Pi package metadata: %v", err)
	}
	var metadata struct {
		Name    string            `json:"name"`
		Version string            `json:"version"`
		Bin     map[string]string `json:"bin"`
	}
	if err := json.Unmarshal(data, &metadata); err != nil {
		t.Fatalf("decode exact Pi package metadata: %v", err)
	}
	if metadata.Name != PackageName || metadata.Version != TargetVersion {
		t.Fatalf("native package = %s@%s, want %s@%s", metadata.Name, metadata.Version, PackageName, TargetVersion)
	}
	if metadata.Bin["pi"] != Entrypoint {
		t.Fatalf("native Pi bin = %q, want %q", metadata.Bin["pi"], Entrypoint)
	}
	module, err := os.ReadFile(filepath.Join(packageRoot, "dist", "core", "settings-manager.js"))
	if err != nil {
		t.Fatalf("read exact Pi settings module: %v", err)
	}
	if got := fmt.Sprintf("%x", sha256.Sum256(module)); got != "5368b155ec26d88374cec9e66b8e588b5041a0fb0047414f70b34e13892c4f48" {
		t.Fatalf("Pi imported settings module hash mismatch: %s", got)
	}
	entrypoint, err := os.ReadFile(filepath.Join(packageRoot, Entrypoint))
	if err != nil {
		t.Fatalf("read exact Pi entrypoint: %v", err)
	}
	digest := sha256.Sum256(entrypoint)
	if got := hex.EncodeToString(digest[:]); got != EntrypointSHA256 {
		t.Fatalf("Pi entrypoint hash = %s, want %s", got, EntrypointSHA256)
	}
	if tarball := os.Getenv("PROFILE_MANGO_PI_NATIVE_TARBALL"); tarball != "" {
		data, err := os.ReadFile(tarball)
		if err != nil {
			t.Fatalf("read exact Pi package tarball: %v", err)
		}
		digest := sha256.Sum256(data)
		if got := hex.EncodeToString(digest[:]); got != PackageTarballSHA256 {
			t.Fatalf("native package tarball hash = %s, want %s", got, PackageTarballSHA256)
		}
	}
}

func runNativeSettingsProbe(t *testing.T, node, packageRoot, mode string) {
	t.Helper()
	bwrap, err := exec.LookPath("bwrap")
	if err != nil {
		t.Fatalf("find bwrap for network-isolated native probe: %v", err)
	}
	root := t.TempDir()
	agent := filepath.Join(root, "agent")
	project := filepath.Join(root, "project")
	if err := os.MkdirAll(filepath.Join(project, ".pi"), 0o755); err != nil {
		t.Fatal(err)
	}
	if mode == "positive" {
		route := profilemango.RouteBinding{Provider: "sentinel-provider", Transport: "native", Authentication: "oauth", Model: "sentinel-global-model", Effort: "high"}
		before := []byte("{\n  \"defaultProvider\": \"old-provider\",\n  \"defaultModel\": \"old-model\",\n  \"defaultThinkingLevel\": \"low\",\n  \"unknown\": {\"keep\": true}\n}\n")
		patch, err := PatchSettings(before, true, route)
		if err != nil {
			t.Fatalf("build native probe settings from Pi patch: %v", err)
		}
		writeNativeFile(t, filepath.Join(agent, "settings.json"), string(patch.Content))
		writeNativeFile(t, filepath.Join(project, ".pi", "settings.json"), "{\n  \"defaultModel\": \"sentinel-project-model\",\n  \"projectUnknown\": [\"keep\"]\n}\n")
	} else {
		writeNativeFile(t, filepath.Join(agent, "settings.json"), "{ malformed\n")
	}
	script := filepath.Join(root, "probe.mjs")
	writeNativeFile(t, script, nativeSettingsProbe)
	before := nativeSandboxSnapshot(t, root)
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, bwrap, nativeProbeArgs(root, packageRoot, node, mode)...)
	cmd.Env = nativeProbeEnv(node)
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("native Pi settings probe: %v\n%s", err, output)
	}
	if after := nativeSandboxSnapshot(t, root); before != after {
		t.Fatalf("native Pi settings probe changed disposable state\nbefore:\n%s\nafter:\n%s", before, after)
	}
	if !strings.Contains(string(output), "\"scope\":\"global\"") && mode == "malformed" {
		t.Fatalf("malformed probe omitted global scope: %s", output)
	}
}

func nativeProbeArgs(root, packageRoot, node, mode string) []string {
	return []string{
		"--die-with-parent", "--new-session", "--unshare-net", "--unshare-pid", "--unshare-ipc", "--cap-drop", "ALL",
		"--ro-bind", "/usr", "/usr", "--ro-bind", "/lib", "/lib", "--ro-bind", "/lib64", "/lib64",
		"--ro-bind", "/bin", "/bin", "--ro-bind", "/etc", "/etc", "--proc", "/proc", "--dev", "/dev",
		"--tmpfs", "/run", "--tmpfs", "/tmp", "--tmpfs", "/var/tmp", "--dir", "/probe", "--ro-bind", node, "/probe/node",
		"--bind", root, "/home/ari", "--ro-bind", packageRoot, "/package", "--chdir", "/home/ari/project",
		"--clearenv", "--setenv", "HOME", "/home/ari", "--setenv", "PI_CODING_AGENT_DIR", "/home/ari/agent",
		"--setenv", "PI_CODING_AGENT_SESSION_DIR", "/home/ari/sessions", "--setenv", "XDG_CONFIG_HOME", "/home/ari/xdg/config",
		"--setenv", "XDG_DATA_HOME", "/home/ari/xdg/data", "--setenv", "XDG_CACHE_HOME", "/home/ari/xdg/cache",
		"--setenv", "XDG_RUNTIME_DIR", "/run", "--setenv", "TMPDIR", "/home/ari/tmp", "--setenv", "PI_OFFLINE", "1",
		"--setenv", "PI_SKIP_VERSION_CHECK", "1", "--setenv", "PATH", "/usr/bin:/bin", "--", "/probe/node",
		"/home/ari/probe.mjs", "/package", "/home/ari/project", "/home/ari/agent", mode,
	}
}

func nativeProbeEnv(node string) []string {
	path := filepath.Dir(node)
	if path == "." {
		path = "/usr/bin"
	}
	return []string{"PATH=" + path + ":/usr/bin:/bin"}
}

func nativeSandboxSnapshot(t *testing.T, root string) string {
	t.Helper()
	entries := make([]string, 0, 16)
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		relative, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		if relative == "." {
			return nil
		}
		entry := fmt.Sprintf("%s %o %d", relative, info.Mode().Perm(), info.Size())
		if info.Mode().IsRegular() {
			data, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			digest := sha256.Sum256(data)
			entry += " " + hex.EncodeToString(digest[:])
		}
		entries = append(entries, entry)
		return nil
	})
	if err != nil {
		t.Fatalf("snapshot native probe sandbox: %v", err)
	}
	sort.Strings(entries)
	return strings.Join(entries, "\n")
}

func writeNativeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(fmt.Errorf("write native probe file: %w", err))
	}
}
