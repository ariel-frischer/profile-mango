package integration_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestOpenCodeConfigProbe(t *testing.T) {
	if os.Getenv("PROFILE_MANGO_RUN_OPENCODE_PROBE") != "1" {
		t.Skip("set PROFILE_MANGO_RUN_OPENCODE_PROBE=1 to run the OpenCode native probe")
	}

	repoRoot := absolutePath(t, "..")
	probeRoot := os.Getenv("PROFILE_MANGO_PROBE_ROOT")
	if probeRoot == "" {
		probeRoot = t.TempDir()
	}
	command := exec.Command("bash", filepath.Join(repoRoot, "scripts", "opencode-config-probe.sh"))
	command.Dir = repoRoot
	command.Env = []string{
		"PATH=/usr/bin:/bin",
		"HOME=" + filepath.Join(probeRoot, "home"),
		"PROFILE_MANGO_OPENCODE_BIN=" + os.Getenv("PROFILE_MANGO_OPENCODE_BIN"),
		"PROFILE_MANGO_OPENCODE_CANDIDATE=" + os.Getenv("PROFILE_MANGO_OPENCODE_CANDIDATE"),
		"PROFILE_MANGO_PROBE_ROOT=" + probeRoot,
		"PROFILE_MANGO_PROBE_TIMEOUT_SECONDS=20",
		"PROFILE_MANGO_KEEP_PROBE=1",
	}
	output, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("OpenCode config probe failed: %v\n%s", err, output)
	}
	assertOpenCodeProbeOutput(t, string(output))
}

func assertOpenCodeProbeOutput(t *testing.T, output string) {
	t.Helper()
	for name, expected := range map[string]string{
		"success":                     "PASS OpenCode config probe",
		"exact version":               "opencode-version=1.18.31",
		"exact binary":                "opencode-binary-sha256=f9dab32248695e9ebd56b16a1921798fd85112cf5a69c7dfd0cabc1e17be4a11",
		"native JSONC acceptance":     "opencode-jsonc-candidate=native-accepted",
		"content precedence":          "opencode-config-content-precedence=observed-over-global",
		"custom file consumption":     "opencode-custom-config-file=consumed",
		"custom file preservation":    "opencode-custom-config-file=byte-identical-after-inspection",
		"malformed rejection":         "opencode-malformed-config=rejected",
		"unknown-key boundary":        "opencode-unknown-key=accepted-and-ignored",
		"effective config boundary":   "opencode-effective-config=merged-output-without-field-provenance",
		"scratch-only writes":         "opencode-native-writes=scratch-only",
		"backup and restore evidence": "opencode-backup-restore=byte-for-byte-inventory-match",
	} {
		if !strings.Contains(output, expected) {
			t.Errorf("probe output missing %s (%q):\n%s", name, expected, output)
		}
	}
}
