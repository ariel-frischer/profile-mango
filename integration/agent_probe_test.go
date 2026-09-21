package integration_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestAgentConfigProbe(t *testing.T) {
	if os.Getenv("PROFILE_MANGO_RUN_AGENT_PROBE") != "1" {
		t.Skip("set PROFILE_MANGO_RUN_AGENT_PROBE=1 to run native target probes")
	}

	repoRoot := absolutePath(t, "..")
	probeRoot := os.Getenv("PROFILE_MANGO_PROBE_ROOT")
	if probeRoot == "" {
		probeRoot = t.TempDir()
	}
	command := exec.Command("bash", filepath.Join(repoRoot, "scripts", "agent-config-probe.sh"))
	command.Dir = repoRoot
	command.Env = []string{
		"PATH=/usr/bin:/bin",
		"HOME=" + filepath.Join(probeRoot, "home"),
		"PROFILE_MANGO_CODEX_BIN=" + os.Getenv("PROFILE_MANGO_CODEX_BIN"),
		"PROFILE_MANGO_JCODE_BIN=" + os.Getenv("PROFILE_MANGO_JCODE_BIN"),
		"PROFILE_MANGO_PROBE_ROOT=" + probeRoot,
		"PROFILE_MANGO_PROBE_TIMEOUT_SECONDS=10",
	}
	output, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("agent config probe failed: %v\n%s", err, output)
	}
	if !strings.Contains(string(output), "PASS agent config probe") {
		t.Fatalf("probe did not report success:\n%s", output)
	}
}
