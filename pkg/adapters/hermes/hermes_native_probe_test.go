package hermes

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/ariel-frischer/profile-mango/pkg/profilemango"
)

func TestHermesNativeProbeSandboxContract(t *testing.T) {
	probe, payload := hermesNativeProbeFiles(t)
	script := string(probe)
	for _, required := range []string{
		"--unshare-net", "--unshare-pid", "--unshare-ipc", "--unshare-uts",
		"--clearenv", "--tmpfs /home", "--tmpfs /root", "--tmpfs /run", "--tmpfs /tmp",
		"--ro-bind \"$source_root\" /opt/hermes-source",
		"--ro-bind \"$site_packages\" /opt/hermes-site-packages",
		"--tmpfs /opt/hermes-source/plugins",
		"--timeout", "--kill-after=2s",
	} {
		if !strings.Contains(script, required) {
			t.Fatalf("probe script omitted required control %q", required)
		}
	}
	if strings.Contains(script, "--ro-bind / /") {
		t.Fatal("probe script contains a forbidden broad root mount")
	}
	if !strings.Contains(script, "--ro-bind /usr /usr") || !strings.Contains(script, "--ro-bind /lib /lib") {
		t.Fatal("probe script omitted explicit runtime mounts")
	}
	payloadText := string(payload)
	for _, required := range []string{
		"socket APIs and audit events trapped", "subprocess APIs and audit events trapped",
		"sqlite3.connect = forbidden", "threading.Thread.start = forbidden",
		"EXPECTED_CONFIG_MODULE_SHA256", "load_config_readonly",
	} {
		if !strings.Contains(payloadText, required) {
			t.Fatalf("probe payload omitted required control %q", required)
		}
	}
	if NativeConfigEvidenceLevel != "isolated-native-config-effective-merge" {
		t.Fatalf("unexpected Hermes native evidence level: %q", NativeConfigEvidenceLevel)
	}
}

func TestHermesNativeProbeGeneratedPatch(t *testing.T) {
	if os.Getenv("PROFILE_MANGO_HERMES_NATIVE_PROBE") != "1" {
		t.Skip("set PROFILE_MANGO_HERMES_NATIVE_PROBE=1 with the pinned source inputs to run the native probe")
	}
	sourceRoot := hermesNativeProbeEnv(t, "HERMES_NATIVE_SOURCE_ROOT")
	sourceArchive := hermesNativeProbeEnv(t, "HERMES_NATIVE_ARCHIVE")
	pythonPath := hermesNativeProbeEnv(t, "HERMES_NATIVE_PYTHON")
	sitePackages := hermesNativeProbeEnv(t, "HERMES_NATIVE_SITE_PACKAGES")
	probePath := hermesNativeProbePath(t)
	stateRoot := t.TempDir()
	if retained := os.Getenv("HERMES_NATIVE_EVIDENCE_ROOT"); retained != "" {
		if !filepath.IsAbs(retained) {
			t.Fatalf("HERMES_NATIVE_EVIDENCE_ROOT must be absolute: %q", retained)
		}
		if err := os.MkdirAll(retained, 0o755); err != nil {
			t.Fatal(err)
		}
		stateRoot = retained
	}
	configPath := filepath.Join(stateRoot, "home", ".hermes", "config.yaml")
	if err := os.MkdirAll(filepath.Dir(configPath), 0o755); err != nil {
		t.Fatal(err)
	}
	patch, err := PatchConfig([]byte(hermesNativeProbeSource()), nativeProbeRoute())
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(configPath, patch.Content, 0o600); err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "/bin/bash", probePath,
		"--source-root", sourceRoot,
		"--source-archive", sourceArchive,
		"--python", pythonPath,
		"--site-packages", sitePackages,
		"--state-root", stateRoot,
		"--timeout", "45",
	)
	cmd.Env = []string{
		"PATH=/usr/bin:/bin",
		"HOME=" + stateRoot,
		"GIT_CONFIG_NOSYSTEM=1",
		"GIT_CONFIG_GLOBAL=/dev/null",
		"HERMES_NATIVE_HOST_SENTINEL=must-not-enter-sandbox",
	}
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("native probe failed: %v\n%s", err, output)
	}
	if ctx.Err() != nil {
		t.Fatalf("native probe context ended: %v", ctx.Err())
	}
	resultBytes, err := os.ReadFile(filepath.Join(stateRoot, "result", "hermes-native-result.json"))
	if err != nil {
		t.Fatal(err)
	}
	var result map[string]any
	if err := json.Unmarshal(resultBytes, &result); err != nil {
		t.Fatalf("invalid native probe result: %v\n%s", err, resultBytes)
	}
	assertNativeProbeString(t, result, "status", "ok")
	assertNativeProbeString(t, result, "config_module_sha256", NativeConfigModuleSHA256)
	assertNativeProbeString(t, result, "version_module_sha256", NativeVersionModuleSHA256)
	assertNativeProbeString(t, result, "source_commit", SourceCommit)
	assertNativeProbeString(t, result, "source_tree", SourceTree)
	assertNativeProbeString(t, result, "source_archive_sha256", SourceArchiveSHA256)
	assertNativeProbeString(t, result, "config_path", "/mnt/home/.hermes/config.yaml")
	assertNativeProbeString(t, result, "runtime_scope", "config module import and effective merge only")
	assertNativeProbeBool(t, result, "clearenv")
	assertNativeProbeBool(t, result, "hidden_real_home_root_run_paths")
	assertNativeProbeBool(t, result, "pid_namespace_isolated")
	selected, ok := result["selected"].(map[string]any)
	if !ok {
		t.Fatalf("selected result has type %T: %#v", result["selected"], result["selected"])
	}
	model, ok := selected["model"].(map[string]any)
	if !ok {
		t.Fatalf("selected model has type %T: %#v", selected["model"], selected["model"])
	}
	assertNativeMapString(t, model, "provider", "openai")
	assertNativeMapString(t, model, "default", "gpt-5.6")
	agent, ok := selected["agent"].(map[string]any)
	if !ok {
		t.Fatalf("selected agent has type %T: %#v", selected["agent"], selected["agent"])
	}
	assertNativeMapString(t, agent, "reasoning_effort", "high")
	unknown, ok := result["unknown_sentinel"].(map[string]any)
	if !ok {
		t.Fatalf("unknown sentinel has type %T: %#v", result["unknown_sentinel"], result["unknown_sentinel"])
	}
	assertNativeMapString(t, unknown, "value", "SYNTHETIC_SECRET_SENTINEL")
	t.Logf("native evidence result: %s", filepath.Join(stateRoot, "result", "hermes-native-result.json"))
}

func hermesNativeProbeFiles(t *testing.T) ([]byte, []byte) {
	t.Helper()
	directory := filepath.Dir(hermesNativeProbePath(t))
	probe, err := os.ReadFile(filepath.Join(directory, "hermes_native_probe.sh"))
	if err != nil {
		t.Fatal(err)
	}
	payload, err := os.ReadFile(filepath.Join(directory, "hermes_native_probe.py"))
	if err != nil {
		t.Fatal(err)
	}
	return probe, payload
}

func hermesNativeProbePath(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("could not locate Hermes native probe files")
	}
	return filepath.Join(filepath.Dir(file), "hermes_native_probe.sh")
}

func hermesNativeProbeEnv(t *testing.T, name string) string {
	t.Helper()
	value := os.Getenv(name)
	if value == "" {
		t.Fatalf("%s must name the retained Hermes native qualification input", name)
	}
	return value
}

func assertNativeProbeString(t *testing.T, result map[string]any, key, want string) {
	t.Helper()
	got, ok := result[key].(string)
	if !ok || got != want {
		t.Fatalf("native probe %s = %#v, want %q", key, result[key], want)
	}
}

func assertNativeProbeBool(t *testing.T, result map[string]any, key string) {
	t.Helper()
	got, ok := result[key].(bool)
	if !ok || !got {
		t.Fatalf("native probe %s = %#v, want true", key, result[key])
	}
}

func assertNativeMapString(t *testing.T, values map[string]any, key, want string) {
	t.Helper()
	got, ok := values[key].(string)
	if !ok || got != want {
		t.Fatalf("native probe value %s = %#v, want %q", key, values[key], want)
	}
}

func hermesNativeProbeSource() string {
	return `# retain header
model:
  provider: old-provider # retain comment
  default: old-model
  extra: keep
agent:
  reasoning_effort: low # retain comment
unknown_sentinel:
  value: SYNTHETIC_SECRET_SENTINEL
`
}

func nativeProbeRoute() profilemango.RouteBinding {
	return profilemango.RouteBinding{
		Provider:       "openai",
		Transport:      "native",
		Authentication: "oauth",
		Model:          "gpt-5.6",
		Effort:         "high",
	}
}
