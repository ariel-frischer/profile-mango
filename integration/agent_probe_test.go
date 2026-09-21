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
	for name, expected := range map[string]string{
		"exact Codex version":             "codex-version=codex-cli 0.154.0",
		"exact Codex evidence hash":       "codex-sha256=3188814c35471432d4123203e0eb38e5bddc60226e3d7ddf0e59e649ea140022",
		"positive feature sentinel":       "codex-features-apps-synthetic=false",
		"negative feature sentinel":       "codex-features-apps-baseline=true",
		"runtime enable precedence":       "codex-features-apps-cli-enable=true",
		"runtime disable precedence":      "codex-features-apps-cli-disable=false",
		"malformed config rejection":      "codex-malformed-config=rejected",
		"config consumption boundary":     "codex-config-consumption=features-apps-only-observed",
		"route capability gap":            "codex-route-provider-model-effort=unverified",
		"authentication capability gap":   "codex-authentication-identity=unverified",
		"precedence capability gap":       "codex-precedence=feature-runtime-override-observed-route-project-unverified",
		"permissions capability gap":      "codex-permissions-tools=enforcement-unverified",
		"delivery capability gap":         "codex-instruction-skill-delivery=unverified",
		"exact Ariel custom-fork version": "jcode-version=jcode v0.83.909-dev (ca8017a3a)",
		"exact Ariel custom-fork hash":    "jcode-sha256=392ecafbb9ec20f49e78cf556a8a8bcb9040c54f2f92db7d6e112c0cf70ea992",
		"Ariel profile field boundary":    "jcode-profile-fields=provider-model-effort-tools-skills-instruction-presence",
		"Ariel tool override":             "jcode-tool-profile-override=observed",
		"Ariel malformed config":          "jcode-malformed-config=rejected",
		"Ariel invalid effort":            "jcode-invalid-effort=rejected",
		"Ariel unknown-key gap":           "jcode-unknown-key=accepted-and-ignored-by-current-custom-fork",
	} {
		if !strings.Contains(string(output), expected) {
			t.Errorf("probe output missing %s (%q):\n%s", name, expected, output)
		}
	}
}
