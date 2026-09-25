package integration_test

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"gitlab.com/ariel-frischer/profile-mango/pkg/install"
)

const openCodeAgentSHA = "f9dab32248695e9ebd56b16a1921798fd85112cf5a69c7dfd0cabc1e17be4a11"
const agentFirst = "MANGO_AGENT_FIRST_ORDERED_SENTINEL"
const agentSecond = "MANGO_AGENT_SECOND_ORDERED_SENTINEL"

type nativeAgent struct {
	Name  string `json:"name"`
	Mode  string `json:"mode"`
	Model struct {
		Provider string `json:"providerID"`
		ID       string `json:"modelID"`
	} `json:"model"`
	Prompt  string `json:"prompt"`
	Variant string `json:"variant"`
}

func TestOpenCodeGeneratedAgentNativeProbe(t *testing.T) {
	if os.Getenv("PROFILE_MANGO_RUN_OPENCODE_AGENT_PROBE") != "1" {
		t.Skip("set PROFILE_MANGO_RUN_OPENCODE_AGENT_PROBE=1 with PROFILE_MANGO_OPENCODE_BIN and PROFILE_MANGO_PROBE_ROOT")
	}
	binary := requiredProbePath(t, "PROFILE_MANGO_OPENCODE_BIN")
	root := requiredProbePath(t, "PROFILE_MANGO_PROBE_ROOT")
	if info, err := os.Lstat(binary); err != nil || !info.Mode().IsRegular() {
		t.Fatalf("pinned binary must be a regular file: %v", err)
	}
	if info, err := os.Lstat(root); err != nil || !info.IsDir() {
		t.Fatalf("disposable probe root must be an existing directory: %v", err)
	}
	for _, unsafe := range []string{"/", os.Getenv("HOME"), absolutePath(t, "..")} {
		if unsafe != "" && (root == unsafe || strings.HasPrefix(unsafe, root+string(os.PathSeparator))) {
			t.Fatalf("probe root contains a non-disposable path: %s", root)
		}
	}
	if resolved, err := filepath.EvalSymlinks(root); err != nil || resolved != root {
		t.Fatalf("probe root must have no symlink components: %v", err)
	}
	if got := agentFileHash(t, binary); got != openCodeAgentSHA {
		t.Fatalf("OpenCode binary SHA mismatch: %s", got)
	}
	if _, err := os.Stat("/usr/bin/bwrap"); err != nil {
		t.Fatalf("required bwrap: %v", err)
	}
	if _, err := os.Stat("/usr/bin/timeout"); err != nil {
		t.Fatalf("required timeout: %v", err)
	}
	for _, mode := range []string{"primary", "subagent"} {
		t.Run(mode, func(t *testing.T) { probeGeneratedAgent(t, root, binary, mode) })
	}
}

func requiredProbePath(t *testing.T, key string) string {
	t.Helper()
	value := os.Getenv(key)
	if value == "" || !filepath.IsAbs(value) || filepath.Clean(value) != value {
		t.Fatalf("%s requires an explicit clean absolute path", key)
	}
	return value
}

func agentFileHash(t *testing.T, path string) string {
	t.Helper()
	f, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := f.Close(); err != nil {
			t.Errorf("close hash input: %v", err)
		}
	}()
	hash := sha256.New()
	if _, err := io.Copy(hash, f); err != nil {
		t.Fatal(err)
	}
	return fmt.Sprintf("%x", hash.Sum(nil))
}

func probeGeneratedAgent(t *testing.T, root, binary, mode string) {
	t.Helper()
	const name = "mango-native-synthetic"
	w := newInstallWorkflow(t, "opencode@1.18.31", "openai", "gpt-5.6", "{\"model\":\"unmanaged\"}\n", "")
	mainConfig := w.config
	w.config = filepath.Join(w.root, "agents", name+".md")
	if err := os.MkdirAll(filepath.Dir(w.config), 0o700); err != nil {
		t.Fatal(err)
	}
	for i, arg := range w.args {
		if arg == "opencode@1.18.31="+mainConfig {
			w.args[i] = "opencode@1.18.31=" + w.config
		}
	}
	w.args = append(w.args, "--agent", "opencode@1.18.31="+mode+":"+name)
	profile := filepath.Join(w.root, "profiles", "minimal", "profile.yaml")
	writeWorkflowFile(t, profile, string(readWorkflowFile(t, profile))+"instructions:\n  append:\n    - instructions/first.md\n    - instructions/second.md\n")
	if err := os.MkdirAll(filepath.Join(w.root, "instructions"), 0o700); err != nil {
		t.Fatal(err)
	}
	writeWorkflowFile(t, filepath.Join(w.root, "instructions", "first.md"), agentFirst+"\n")
	writeWorkflowFile(t, filepath.Join(w.root, "instructions", "second.md"), agentSecond+"\n")
	plan := w.plan(t)
	if plan.Status != install.StatusReady || plan.Targets[0].Agent == nil || plan.Targets[0].Agent.Mode != mode {
		t.Fatalf("unexpected plan: %#v", plan)
	}
	if result := w.run(t, "--apply", "--yes", "--expect-plan", plan.PlanID); result.err != nil {
		t.Fatalf("Mango apply: %v %s", result.err, result.stderr)
	}
	generated := readWorkflowFile(t, w.config)
	assertWorkflowBytes(t, mainConfig, []byte(w.original))
	if !strings.Contains(string(generated), agentFirst) || !strings.Contains(string(generated), agentSecond) {
		t.Fatal("Mango did not generate both prompt sentinels")
	}
	runGeneratedAgentControls(t, root, binary, name, mode, generated)
	assertWorkflowBytes(t, w.config, generated)
	assertWorkflowBytes(t, filepath.Join(w.root, "outside-sentinel"), []byte("untouched"))
}

func runGeneratedAgentControls(t *testing.T, root, binary, name, mode string, generated []byte) {
	t.Helper()
	work, err := os.MkdirTemp(root, "mango-opencode-agent-")
	if err != nil {
		t.Fatal(err)
	}
	state := filepath.Join(work, "state")
	target := filepath.Join(state, "config-dir", "agents", name+".md")
	for _, dir := range []string{"home", "config-dir/agents", "managed", "project", "xdg/config", "xdg/data", "xdg/cache", "xdg/state"} {
		if err := os.MkdirAll(filepath.Join(state, dir), 0o700); err != nil {
			t.Fatal(err)
		}
	}
	writeWorkflowFile(t, target, string(generated))
	writeWorkflowFile(t, filepath.Join(state, "candidate.json"), "{\"autoupdate\":false}\n")
	sentinel := filepath.Join(work, "outside-sentinel")
	writeWorkflowFile(t, sentinel, "untouched")
	baseline := agentInventory(t, state)
	backup := filepath.Join(work, "backup")
	copyAgentTree(t, state, backup)
	defer func() {
		assertWorkflowBytes(t, sentinel, []byte("untouched"))
		if agentInventory(t, backup) != baseline {
			t.Error("backup changed during native probe")
		}
		if err := os.RemoveAll(state); err != nil {
			t.Fatal(err)
		}
		copyAgentTree(t, backup, state)
		if agentInventory(t, state) != baseline {
			t.Error("restored state differs from baseline")
		}
	}()
	positive := resolveNativeAgent(t, binary, state, name)
	assertResolvedAgent(t, positive, name, mode)
	assertWorkflowBytes(t, target, generated)
	controls := map[string]struct {
		data  []byte
		check func(nativeAgent) bool
	}{
		"changed mode":    {replaceAgentBytes(t, generated, "mode: "+mode, "mode: "+otherAgentMode(mode)), func(a nativeAgent) bool { return a.Mode == otherAgentMode(mode) }},
		"changed model":   {replaceAgentBytes(t, generated, `model: "openai/gpt-5.6"`, `model: "openai/mango-negative-control"`), func(a nativeAgent) bool { return a.Model.ID == "mango-negative-control" }},
		"changed variant": {replaceAgentBytes(t, generated, `variant: "high"`, `variant: "low"`), func(a nativeAgent) bool { return a.Variant == "low" }},
		"missing variant": {replaceAgentBytes(t, generated, "variant: \"high\"\n", ""), func(a nativeAgent) bool { return a.Variant == "" }},
		"changed prompt": {replaceAgentBytes(t, generated, agentFirst, "MANGO_NEGATIVE_PROMPT_CONTROL"), func(a nativeAgent) bool {
			return !strings.Contains(a.Prompt, agentFirst) && strings.Contains(a.Prompt, "MANGO_NEGATIVE_PROMPT_CONTROL")
		}},
		"missing prompt": {replaceAgentBytes(t, generated, agentFirst, ""), func(a nativeAgent) bool { return !strings.Contains(a.Prompt, agentFirst) }},
	}
	for label, control := range controls {
		t.Run(label, func(t *testing.T) {
			writeWorkflowFile(t, target, string(control.data))
			if got := resolveNativeAgent(t, binary, state, name); !control.check(got) {
				t.Fatalf("native negative control did not affect %s: %#v", label, got)
			}
			assertWorkflowBytes(t, target, control.data)
		})
	}
	writeWorkflowFile(t, target, string(generated))
	missing := filepath.Join(filepath.Dir(target), "wrong-name.md")
	if err := os.Rename(target, missing); err != nil {
		t.Fatal(err)
	}
	if got, err := nativeAgentResult(binary, state, name); err == nil && strings.Contains(got.Prompt, agentFirst) {
		t.Fatal("native resolver consumed wrong-name definition")
	}
	if err := os.Rename(missing, target); err != nil {
		t.Fatal(err)
	}
	assertWorkflowBytes(t, target, generated)
}

func replaceAgentBytes(t *testing.T, original []byte, old, replacement string) []byte {
	t.Helper()
	if bytes.Count(original, []byte(old)) != 1 {
		t.Fatalf("expected exactly one %q in Mango output", old)
	}
	return bytes.Replace(original, []byte(old), []byte(replacement), 1)
}

func otherAgentMode(mode string) string {
	if mode == "primary" {
		return "subagent"
	}
	return "primary"
}

func assertResolvedAgent(t *testing.T, got nativeAgent, name, mode string) {
	t.Helper()
	first, second := strings.Index(got.Prompt, agentFirst), strings.Index(got.Prompt, agentSecond)
	if got.Name != name || got.Mode != mode || got.Model.Provider != "openai" || got.Model.ID != "gpt-5.6" || got.Variant != "high" || first < 0 || second <= first {
		t.Fatalf("generated agent not consumed in order: name=%q mode=%q model=%+v variant=%q prompt=%q", got.Name, got.Mode, got.Model, got.Variant, got.Prompt)
	}
}

func resolveNativeAgent(t *testing.T, binary, state, name string) nativeAgent {
	t.Helper()
	got, err := nativeAgentResult(binary, state, name)
	if err != nil {
		t.Fatal(err)
	}
	return got
}

func nativeAgentResult(binary, state, name string) (nativeAgent, error) {
	var result nativeAgent
	if name != "mango-native-synthetic" {
		return result, fmt.Errorf("unsafe native agent name")
	}
	args := []string{"20", "/usr/bin/bwrap", "--die-with-parent", "--new-session", "--unshare-net", "--unshare-pid", "--unshare-ipc", "--cap-drop", "ALL"}
	for _, path := range []string{"/usr", "/bin", "/lib", "/lib64"} {
		if _, err := os.Stat(path); err == nil {
			args = append(args, "--ro-bind", path, path)
		}
	}
	args = append(args, "--dir", "/etc", "--dir", "/probe", "--ro-bind", binary, "/probe/opencode", "--bind", state, "/state", "--proc", "/proc", "--dev", "/dev", "--tmpfs", "/tmp", "--chdir", "/state/project", "/usr/bin/env", "-i")
	args = append(args, nativeAgentEnv()...)
	args = append(args, "/probe/opencode", "debug", "agent", name)
	ctx, cancel := context.WithTimeout(context.Background(), 25*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "/usr/bin/timeout", args...)
	cmd.Env = []string{"PATH=/usr/bin:/bin"}
	output, err := cmd.Output()
	if err != nil {
		return result, fmt.Errorf("isolated native debug agent: %w", err)
	}
	if err := json.Unmarshal(output, &result); err != nil {
		return result, fmt.Errorf("decode isolated native result: %w", err)
	}
	return result, nil
}

func nativeAgentEnv() []string {
	return []string{
		"PATH=/usr/bin:/bin", "HOME=/state/home", "XDG_CONFIG_HOME=/state/xdg/config", "XDG_DATA_HOME=/state/xdg/data", "XDG_CACHE_HOME=/state/xdg/cache", "XDG_STATE_HOME=/state/xdg/state", "OPENCODE_TEST_HOME=/state/home", "OPENCODE_CONFIG_DIR=/state/config-dir", "OPENCODE_CONFIG=/state/candidate.json", "OPENCODE_DISABLE_PROJECT_CONFIG=1", "OPENCODE_AUTH_CONTENT={}", "OPENCODE_DB=:memory:", "OPENCODE_TEST_MANAGED_CONFIG_DIR=/state/managed", "OPENCODE_DISABLE_DEFAULT_PLUGINS=1", "OPENCODE_DISABLE_EXTERNAL_SKILLS=1", "OPENCODE_DISABLE_CLAUDE_CODE=1", "OPENCODE_DISABLE_CLAUDE_CODE_SKILLS=1", "OPENCODE_DISABLE_LSP_DOWNLOAD=1", "OPENCODE_DISABLE_AUTOUPDATE=1", "OPENCODE_DISABLE_MODELS_FETCH=1", "OPENCODE_DISABLE_PRUNE=1", "OPENCODE_DISABLE_SHARE=1", "OPENCODE_PURE=1", "NO_COLOR=1", "TERM=dumb",
	}
}

func agentInventory(t *testing.T, root string) string {
	t.Helper()
	var entries []string
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && !info.Mode().IsRegular() {
			return fmt.Errorf("non-regular probe entry: %s", path)
		}
		relative, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		entry := fmt.Sprintf("%s:%v:%o", relative, info.IsDir(), info.Mode().Perm())
		if info.Mode().IsRegular() {
			entry += ":" + agentFileHash(t, path)
		}
		entries = append(entries, entry)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return strings.Join(entries, "\n")
}

func copyAgentTree(t *testing.T, source, target string) {
	t.Helper()
	err := filepath.Walk(source, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && !info.Mode().IsRegular() {
			return fmt.Errorf("non-regular probe entry: %s", path)
		}
		relative, err := filepath.Rel(source, path)
		if err != nil {
			return err
		}
		dest := filepath.Join(target, relative)
		if info.IsDir() {
			return os.MkdirAll(dest, info.Mode().Perm())
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		return os.WriteFile(dest, data, info.Mode().Perm())
	})
	if err != nil {
		t.Fatal(err)
	}
}
