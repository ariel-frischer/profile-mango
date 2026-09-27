package install

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// openCodeResearchSkill is openCodeTestSkill named for the research folder.
var openCodeResearchSkill = strings.Replace(openCodeTestSkill, "profile-mango-synthetic", "research", 1)

func addOpenCodeTestSkill(t *testing.T, root string) {
	t.Helper()
	profile := "apiVersion: profilemango.dev/v1alpha1\nkind: PolicyProfile\nmetadata:\n  name: route-only\nspec:\n  routeRef: primary\n  skills:\n    - skills/research/SKILL.md\n"
	writeInstallTestFile(t, filepath.Join(root, "profiles", "route-only", "profile.yaml"), profile)
	resource := filepath.Join(root, "skills", "research", "SKILL.md")
	if err := os.MkdirAll(filepath.Dir(resource), 0o755); err != nil {
		t.Fatal(err)
	}
	writeInstallTestFile(t, resource, openCodeResearchSkill)
}
