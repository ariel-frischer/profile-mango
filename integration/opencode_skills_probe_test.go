package integration_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestOpenCodeSkillsProbe(t *testing.T) {
	if os.Getenv("PROFILE_MANGO_RUN_OPENCODE_SKILLS_PROBE") != "1" {
		t.Skip("set PROFILE_MANGO_RUN_OPENCODE_SKILLS_PROBE=1 to run the OpenCode skills native probe")
	}

	repoRoot := absolutePath(t, "..")
	probeRoot := os.Getenv("PROFILE_MANGO_PROBE_ROOT")
	if probeRoot == "" {
		probeRoot = t.TempDir()
	}
	binary := os.Getenv("PROFILE_MANGO_OPENCODE_BIN")
	if binary == "" {
		t.Fatal("PROFILE_MANGO_OPENCODE_BIN must point to the retained exact OpenCode binary")
	}
	command := exec.Command("bash", filepath.Join(repoRoot, "scripts", "opencode-skills-probe.sh"))
	command.Dir = repoRoot
	command.Env = []string{
		"PATH=/usr/bin:/bin",
		"HOME=" + filepath.Join(probeRoot, "home"),
		"PROFILE_MANGO_OPENCODE_BIN=" + binary,
		"PROFILE_MANGO_PROBE_ROOT=" + probeRoot,
		"PROFILE_MANGO_PROBE_TIMEOUT_SECONDS=20",
		"PROFILE_MANGO_KEEP_PROBE=1",
	}
	output, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("OpenCode skills probe failed: %v\n%s", err, output)
	}
	for name, expected := range map[string]string{
		"success":             "PASS OpenCode skills probe",
		"exact version":       "opencode-version=1.18.31",
		"exact binary":        "opencode-binary-sha256=f9dab32248695e9ebd56b16a1921798fd85112cf5a69c7dfd0cabc1e17be4a11",
		"config acceptance":   "opencode-skills-config=natively-accepted",
		"skill discovery":     "opencode-skill-discovery=listed-and-loaded",
		"skill body":          "opencode-skill-body=observed",
		"scratch-only writes": "opencode-native-writes=scratch-only",
		"no agent session":    "opencode-no-agent-session=no-model-call-or-TUI",
		"restore":             "opencode-backup-restore=byte-for-byte-inventory-match",
	} {
		if !strings.Contains(string(output), expected) {
			t.Errorf("probe output missing %s (%q):\n%s", name, expected, output)
		}
	}
}
