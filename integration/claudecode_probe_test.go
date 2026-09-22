package integration_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestClaudeCodeConfigProbe(t *testing.T) {
	if os.Getenv("PROFILE_MANGO_RUN_CLAUDE_PROBE") != "1" {
		t.Skip("set PROFILE_MANGO_RUN_CLAUDE_PROBE=1 to run the Claude Code native probe")
	}

	repoRoot := absolutePath(t, "..")
	probeRoot := os.Getenv("PROFILE_MANGO_PROBE_ROOT")
	if probeRoot == "" {
		probeRoot = t.TempDir()
	}
	command := exec.Command("bash", filepath.Join(repoRoot, "scripts", "claudecode-config-probe.sh"))
	command.Dir = repoRoot
	command.Env = []string{
		"PATH=/usr/bin:/bin",
		"HOME=" + filepath.Join(probeRoot, "outer-home"),
		"PROFILE_MANGO_CLAUDE_BIN=" + os.Getenv("PROFILE_MANGO_CLAUDE_BIN"),
		"PROFILE_MANGO_PROBE_ROOT=" + probeRoot,
		"PROFILE_MANGO_PROBE_TIMEOUT_SECONDS=20",
		"PROFILE_MANGO_KEEP_PROBE=1",
	}
	output, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("Claude Code config probe failed: %v\n%s", err, output)
	}
	for name, expected := range map[string]string{
		"success":               "PASS Claude Code config probe",
		"exact version":         "claude-version=2.1.278 (Claude Code)",
		"exact binary":          "claude-binary-sha256=5c4735937844e84f8a93306e841a5b0e12252909b07870f789b190468da147ab",
		"model consumption":     "claude-model-sentinel=consumed-before-auth-or-provider",
		"settings preservation": "claude-settings-file=byte-identical-after-native-startup",
		"malformed rejection":   "claude-malformed-settings-argument=rejected",
		"credential boundary":   "claude-native-writes=sandbox-only",
		"network isolation":     "claude-isolation=env-i-network-unshared-timeout-bounded",
	} {
		if !strings.Contains(string(output), expected) {
			t.Errorf("probe output missing %s (%q):\n%s", name, expected, output)
		}
	}
}
