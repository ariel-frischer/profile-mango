package install

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"
)

// setSkillTestTime moves path's modification time by offset from now, so tests decide
// which side of a skill comparison is newer instead of relying on write order.
func setSkillTestTime(t *testing.T, path string, offset time.Duration) {
	t.Helper()
	when := time.Now().Add(offset)
	if err := os.Chtimes(path, when, when); err != nil {
		t.Fatal(err)
	}
}

// skillDriftReport inspects status for request, comparing profile skills with global.
func skillDriftReport(t *testing.T, request Request, global string) []SkillDrift {
	t.Helper()
	report, err := InspectStatus(StatusRequest{ProfilesRoot: request.ProfilesRoot, ResourceRoot: request.ResourceRoot, BindingsPath: request.BindingsPath, Registry: request.Registry, Env: request.Env, Targets: request.Targets, GlobalSkillsRoot: global, StateDir: testStateDir})
	if err != nil {
		t.Fatal(err)
	}
	return report.Skills
}

// TestStatusReportsGlobalSkillDriftAndNewerSide installs a default codex profile, whose
// skills land in ~/.agents/skills, then changes one side at a time.
func TestStatusReportsGlobalSkillDriftAndNewerSide(t *testing.T) {
	tests := map[string]struct {
		change    func(t *testing.T, root, global string)
		files     []string
		newer     string
		hintParts []string
	}{
		"global copy edited later": {
			change: func(t *testing.T, root, global string) {
				writeInstallTestFile(t, filepath.Join(global, "review", "SKILL.md"), reviewSkill+"global edit\n")
				setSkillTestTime(t, filepath.Join(root, "skills", "review", "SKILL.md"), -time.Hour)
				setSkillTestTime(t, filepath.Join(root, "skills", "review", "scripts", "check.sh"), -time.Hour)
			},
			files: []string{"SKILL.md"}, newer: SkillNewerGlobal, hintParts: []string{"port its edits into skills/review", "--override"},
		},
		"global copy gained a file": {
			change: func(t *testing.T, _, global string) {
				writeInstallTestFile(t, filepath.Join(global, "review", "notes.md"), "notes\n")
				setSkillTestTime(t, filepath.Join(global, "review", "notes.md"), time.Hour)
			},
			files: []string{"notes.md"}, newer: SkillNewerGlobal,
		},
		"profile copy edited later": {
			change: func(t *testing.T, root, global string) {
				writeInstallTestFile(t, filepath.Join(root, "skills", "review", "scripts", "check.sh"), reviewScript+"echo newer\n")
				for _, file := range []string{"SKILL.md", "scripts/check.sh"} {
					setSkillTestTime(t, filepath.Join(global, "review", filepath.FromSlash(file)), -time.Hour)
				}
			},
			files: []string{"scripts/check.sh"}, newer: SkillNewerProfile, hintParts: []string{"mango use route-only", "do not copy the global copy over skills/review"},
		},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			request, root, global := skillRequest(t, skillTargets()["codex"])
			applySkillPlan(t, request)
			if drift := skillDriftReport(t, request, ""); len(drift) != 0 {
				t.Fatalf("drift right after install = %#v", drift)
			}
			test.change(t, root, global)
			drift := skillDriftReport(t, request, "")
			if len(drift) != 1 {
				t.Fatalf("drift = %#v", drift)
			}
			entry := drift[0]
			if entry.Profile != "route-only" || entry.Skill != "review" || entry.Snapshot != "skills/review" || entry.Global != "~/.agents/skills/review" || !reflect.DeepEqual(entry.Files, test.files) || entry.Newer != test.newer {
				t.Fatalf("drift entry = %#v", entry)
			}
			for _, part := range test.hintParts {
				if !strings.Contains(entry.Hint, part) {
					t.Fatalf("hint %q lacks %q", entry.Hint, part)
				}
			}
		})
	}
}

// TestStatusSkillDriftIgnoresRenderedPlaceholders: a global copy holding the rendered
// route is the profile's copy, not drift; a configured global directory is honored, and
// a skill absent from it is not reported.
func TestStatusSkillDriftIgnoresRenderedPlaceholders(t *testing.T) {
	request, root, _ := skillRequest(t, skillTargets()["oh-my-pi"])
	writeInstallTestFile(t, filepath.Join(root, "skills", "review", "SKILL.md"), reviewSkill+"Use {{route.primary.model}}.\n")
	plan := applySkillPlan(t, request)
	installed := filepath.Join(filepath.Dir(request.Targets[0].ConfigPath), "skills")
	if len(plan.Targets) != 1 {
		t.Fatalf("plan targets = %d", len(plan.Targets))
	}
	rendered, err := os.ReadFile(filepath.Join(installed, "review", "SKILL.md"))
	if err != nil || strings.Contains(string(rendered), "{{route.") {
		t.Fatalf("installed skill = %q, %v", rendered, err)
	}
	global := filepath.Join(root, "global-skills")
	writeInstallTestFile(t, filepath.Join(global, "review", "SKILL.md"), string(rendered))
	writeInstallTestFile(t, filepath.Join(global, "review", "scripts", "check.sh"), reviewScript)
	if drift := skillDriftReport(t, request, global); len(drift) != 0 {
		t.Fatalf("rendered global copy reported as drift: %#v", drift)
	}
	writeInstallTestFile(t, filepath.Join(global, "review", "SKILL.md"), string(rendered)+"edit\n")
	if drift := skillDriftReport(t, request, global); len(drift) != 1 || drift[0].Global != global+"/review" || drift[0].Skill != "review" {
		t.Fatalf("edited configured global copy drift = %#v", drift)
	}
}

// TestSkillInstallRefusesNewerUnownedCopy: an install never silently replaces a skill
// file profile-mango does not own when it is newer than the profile's copy.
func TestSkillInstallRefusesNewerUnownedCopy(t *testing.T) {
	for name, target := range skillTargets() {
		t.Run(name, func(t *testing.T) {
			request, _, skills := skillRequest(t, target)
			existing := filepath.Join(skills, "review", "SKILL.md")
			writeInstallTestFile(t, existing, reviewSkill+"hand edit\n")
			setSkillTestTime(t, existing, time.Hour)
			plan := buildSkillPlan(t, request, StatusBlocked)
			if plan.Targets[0].Status != StatusConflict || !hasDiagnostic(plan.Targets[0].Diagnostics, "install.skill_destination_newer") || !strings.Contains(plan.Targets[0].Reason, "--override") {
				t.Fatalf("newer unowned skill target = %#v", plan.Targets[0])
			}
			assertInstallTestFile(t, existing, reviewSkill+"hand edit\n")
			request.Override = true
			applySkillPlan(t, request)
			assertInstallTestFile(t, existing, reviewSkill)
		})
	}
}
