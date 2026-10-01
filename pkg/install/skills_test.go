package install

import (
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/ariel-frischer/profile-mango/internal/installfs"
	"github.com/ariel-frischer/profile-mango/pkg/profilemango"
)

const (
	reviewSkill   = "---\nname: review\ndescription: Use for reviews.\n---\nRun scripts/check.sh.\n"
	reviewScript  = "#!/bin/sh\necho ok\n"
	vendoredSkill = "---\nname: vendored\ndescription: Use for vendored work.\n---\nVendored body.\n"
	skillCommit   = "0123456789abcdef0123456789abcdef01234567"
)

// skillTarget builds one target's install request and names its skill root. namedWrites
// marks targets that also write skills for a named profile, into that profile's own root.
type skillTarget struct {
	request     func(*testing.T) (Request, string)
	root        func(request Request, home string) string
	namedWrites bool
}

func skillTargets() map[string]skillTarget {
	beside := func(request Request, _ string) string {
		return filepath.Join(filepath.Dir(request.Targets[0].ConfigPath), "skills")
	}
	return map[string]skillTarget{
		"claude-code": {request: claudeCodeTestRequest, root: beside},
		"codex": {request: func(t *testing.T) (Request, string) {
			request, root := codexTestRequest(t)
			request.Registry = DefaultRegistry()
			return request, root
		}, root: func(_ Request, home string) string { return filepath.Join(home, ".agents", "skills") }},
		"oh-my-pi": {request: ohMyPiTestRequest, root: beside},
		"openclaw": {request: func(t *testing.T) (Request, string) {
			request, config := openClawInstallRequest(t)
			return request, filepath.Dir(filepath.Dir(filepath.Dir(config)))
		}, root: beside, namedWrites: true},
		"opencode": {request: openCodeTestRequest, root: beside},
	}
}

// skillRequest returns a default, backed-up install of the two-skill profile for target,
// the package root, the user home, and the target's skill root.
func skillRequest(t *testing.T, target skillTarget) (Request, string, string) {
	t.Helper()
	request, root := target.request(t)
	home := filepath.Join(root, "home")
	request.Env = PathEnv{UserHome: func() (string, error) { return home, nil }}
	request.Default, request.Backup, request.Override, request.Release = true, true, false, true
	writeSkillPackage(t, root, true)
	return request, root, target.root(request, home)
}

// writeSkillPackage writes a review skill with an executable script and a vendored skill
// pinned by its tree digest; withSkills false writes the same profile without skills.
func writeSkillPackage(t *testing.T, root string, withSkills bool) {
	t.Helper()
	writeInstallTestFile(t, filepath.Join(root, "skills", "review", "SKILL.md"), reviewSkill)
	script := filepath.Join(root, "skills", "review", "scripts", "check.sh")
	writeInstallTestFile(t, script, reviewScript)
	if err := os.Chmod(script, 0o700); err != nil {
		t.Fatal(err)
	}
	writeInstallTestFile(t, filepath.Join(root, "skills", "vendored", "SKILL.md"), vendoredSkill)
	digest := profilemango.SkillTreeDigest([]profilemango.SkillTreeFile{{Path: "SKILL.md", Mode: 0o644, SHA256: installfs.Hash([]byte(vendoredSkill))}})
	profile := "name: route-only\nroute: primary\n"
	if withSkills {
		profile += "skills:\n  - skills/review/SKILL.md\n  - path: skills/vendored/SKILL.md\n    source: {repo: https://github.com/example/skills, commit: " + skillCommit + ", path: vendored, sha256: " + digest + "}\n"
	}
	writeInstallTestFile(t, filepath.Join(root, "profiles", "route-only", "profile.yaml"), profile)
}

func buildSkillPlan(t *testing.T, request Request, want string) Plan {
	t.Helper()
	plan, err := BuildPlan(request)
	if err != nil {
		t.Fatal(err)
	}
	if plan.Status != want {
		t.Fatalf("plan status = %s, want %s: %v %#v", plan.Status, want, plan.Diagnostics, plan.Targets)
	}
	return plan
}

func applySkillPlan(t *testing.T, request Request) Plan {
	t.Helper()
	plan := buildSkillPlan(t, request, StatusReady)
	if _, err := ApplyPlan(plan, ApplyOptions{ExpectedPlanID: plan.PlanID}); err != nil {
		t.Fatal(err)
	}
	return plan
}

func assertSkillFile(t *testing.T, path, content string, mode fs.FileMode) {
	t.Helper()
	assertInstallTestFile(t, path, content)
	info, err := os.Stat(path)
	if err != nil || info.Mode().Perm() != mode {
		t.Fatalf("%s mode = %v, want %v (err %v)", path, info.Mode().Perm(), mode, err)
	}
}

func skillStatus(t *testing.T, request Request) TargetStatus {
	t.Helper()
	report, err := InspectStatus(StatusRequest{ProfilesRoot: request.ProfilesRoot, ResourceRoot: request.ResourceRoot, BindingsPath: request.BindingsPath, Registry: request.Registry, Env: request.Env, Targets: request.Targets})
	if err != nil || len(report.Targets) != 1 {
		t.Fatalf("status = %#v, %v", report, err)
	}
	return report.Targets[0]
}

func TestSkillInstallLifecycle(t *testing.T) {
	for name, target := range skillTargets() {
		t.Run(name, func(t *testing.T) {
			request, root, skills := skillRequest(t, target)
			plan := applySkillPlan(t, request)
			if got := plan.Targets[0].Skills; !reflect.DeepEqual(got, []string{"review", "vendored"}) {
				t.Fatalf("planned skills = %v", got)
			}
			assertSkillFile(t, filepath.Join(skills, "review", "SKILL.md"), reviewSkill, 0o644)
			assertSkillFile(t, filepath.Join(skills, "review", "scripts", "check.sh"), reviewScript, 0o755)
			assertSkillFile(t, filepath.Join(skills, "vendored", "SKILL.md"), vendoredSkill, 0o644)
			if status := skillStatus(t, request); status.Source != SourceCurrent {
				t.Fatalf("status after install = %s (%s)", status.Source, status.SourceReason)
			}
			writeInstallTestFile(t, filepath.Join(root, "skills", "review", "scripts", "check.sh"), reviewScript+"echo edited\n")
			if status := skillStatus(t, request); status.Source != SourceChanged {
				t.Fatalf("status after bundle edit = %s (%s)", status.Source, status.SourceReason)
			}
			writeSkillPackage(t, root, false)
			applySkillPlan(t, request)
			if _, err := os.Lstat(filepath.Join(skills, "review")); !os.IsNotExist(err) {
				t.Fatalf("released skill folder remains: %v", err)
			}
			applyUndo(t, UndoRequest{Target: request.Targets[0].Target, ConfigPath: request.Targets[0].ConfigPath, Registry: request.Registry})
			assertSkillFile(t, filepath.Join(skills, "review", "scripts", "check.sh"), reviewScript, 0o755)
			// Undoing the first install removes the skill files it created at their source modes.
			applyUndo(t, UndoRequest{Target: request.Targets[0].Target, ConfigPath: request.Targets[0].ConfigPath, Registry: request.Registry})
			for _, folder := range []string{"review", "vendored"} {
				if _, err := os.Lstat(filepath.Join(skills, folder)); !os.IsNotExist(err) {
					t.Fatalf("created skill folder %s survived undo: %v", folder, err)
				}
			}
		})
	}
}

// A skill file that already holds the profile's bytes is claimed without a write; undo
// releases it untouched and removes only what the install created.
func TestSkillUndoKeepsFileClaimedAsIs(t *testing.T) {
	for name, target := range skillTargets() {
		t.Run(name, func(t *testing.T) {
			request, _, skills := skillRequest(t, target)
			existing := filepath.Join(skills, "vendored", "SKILL.md")
			writeInstallTestFile(t, existing, vendoredSkill)
			if err := os.Chmod(existing, 0o644); err != nil {
				t.Fatal(err)
			}
			applySkillPlan(t, request)
			applyUndo(t, UndoRequest{Target: request.Targets[0].Target, ConfigPath: request.Targets[0].ConfigPath, Registry: request.Registry})
			assertSkillFile(t, existing, vendoredSkill, 0o644)
			if _, err := os.Lstat(filepath.Join(skills, "review")); !os.IsNotExist(err) {
				t.Fatalf("created skill folder survived undo: %v", err)
			}
		})
	}
}

// TestSkillPlaceholdersRenderPerTarget renders {{route.…}} in a skill's text files from
// each target's own route, copies non-UTF-8 files verbatim, and reports the rendered
// install in-sync.
func TestSkillPlaceholdersRenderPerTarget(t *testing.T) {
	binary := "\xff\xfe{{route.primary.provider}}"
	tests := map[string]string{"claude-code": "anthropic", "codex": "openai"}
	for name, provider := range tests {
		t.Run(name, func(t *testing.T) {
			target := skillTargets()[name]
			request, root, skills := skillRequest(t, target)
			writeInstallTestFile(t, filepath.Join(root, "skills", "review", "SKILL.md"), reviewSkill+"Route: {{route.primary.provider}}\n")
			writeInstallTestFile(t, filepath.Join(root, "skills", "review", "data.bin"), binary)
			applySkillPlan(t, request)
			assertSkillFile(t, filepath.Join(skills, "review", "SKILL.md"), reviewSkill+"Route: "+provider+"\n", 0o644)
			assertSkillFile(t, filepath.Join(skills, "review", "data.bin"), binary, 0o644)
			status := skillStatus(t, request)
			for _, file := range status.Files {
				if file.State != FileInSync {
					t.Fatalf("status %s = %s after install, want in-sync", file.Path, file.State)
				}
			}
			if status.Source != SourceCurrent {
				t.Fatalf("status source = %s (%s)", status.Source, status.SourceReason)
			}
		})
	}
}

func TestSkillInvalidPlaceholderNamesFile(t *testing.T) {
	request, root, _ := skillRequest(t, skillTargets()["codex"])
	writeInstallTestFile(t, filepath.Join(root, "skills", "review", "SKILL.md"), reviewSkill+"\n{{route.nope.model}}\n")
	_, err := BuildPlan(request)
	if err == nil || !strings.Contains(err.Error(), "skills/review/SKILL.md") || !strings.Contains(err.Error(), "line 7") {
		t.Fatalf("invalid placeholder error = %v", err)
	}
}

func TestSkillSwitchRestoresAdoptedFolder(t *testing.T) {
	for name, target := range skillTargets() {
		t.Run(name, func(t *testing.T) {
			request, root, skills := skillRequest(t, target)
			existing := filepath.Join(skills, "review", "SKILL.md")
			writeInstallTestFile(t, existing, "user skill\n")
			setSkillTestTime(t, existing, -time.Hour)
			writeInstallTestFile(t, filepath.Join(skills, "review", "notes.md"), "user notes\n")
			plan := buildSkillPlan(t, request, StatusReady)
			if !hasDiagnostic(plan.Targets[0].Diagnostics, "install.skill_unmanaged_files") || !hasFileAction(plan.Targets[0], ActionAdopt) {
				t.Fatalf("adoption plan = %#v", plan.Targets[0])
			}
			if _, err := ApplyPlan(plan, ApplyOptions{ExpectedPlanID: plan.PlanID}); err != nil {
				t.Fatal(err)
			}
			assertInstallTestFile(t, existing, reviewSkill)
			writeSkillPackage(t, root, false)
			applySkillPlan(t, request)
			assertInstallTestFile(t, existing, "user skill\n")
			assertInstallTestFile(t, filepath.Join(skills, "review", "notes.md"), "user notes\n")
			if _, err := os.Lstat(filepath.Join(skills, "review", "scripts")); !os.IsNotExist(err) {
				t.Fatalf("released scripts folder remains: %v", err)
			}
		})
	}
}

func TestSkillEditedInstalledFileConflicts(t *testing.T) {
	for name, target := range skillTargets() {
		t.Run(name, func(t *testing.T) {
			request, _, skills := skillRequest(t, target)
			applySkillPlan(t, request)
			script := filepath.Join(skills, "review", "scripts", "check.sh")
			writeInstallTestFile(t, script, "edited by user\n")
			plan := buildSkillPlan(t, request, StatusBlocked)
			if plan.Targets[0].Status != StatusConflict {
				t.Fatalf("edited skill file target = %#v", plan.Targets[0])
			}
			assertInstallTestFile(t, script, "edited by user\n")
		})
	}
}

func TestSkillDestinationSymlinkConflicts(t *testing.T) {
	for name, target := range skillTargets() {
		t.Run(name, func(t *testing.T) {
			request, root, skills := skillRequest(t, target)
			elsewhere := filepath.Join(root, "elsewhere")
			if err := os.MkdirAll(elsewhere, 0o755); err != nil {
				t.Fatal(err)
			}
			if err := os.MkdirAll(skills, 0o755); err != nil {
				t.Fatal(err)
			}
			if err := os.Symlink(elsewhere, filepath.Join(skills, "review")); err != nil {
				t.Fatal(err)
			}
			plan := buildSkillPlan(t, request, StatusBlocked)
			var found bool
			for _, diagnostic := range plan.Targets[0].Diagnostics {
				found = found || (diagnostic.Code == "install.skill_destination_conflict" && strings.Contains(diagnostic.Message, "review is a symlink"))
			}
			if !found {
				t.Fatalf("symlink conflict not named: %v", plan.Targets[0].Diagnostics)
			}
			if entries, _ := os.ReadDir(elsewhere); len(entries) != 0 {
				t.Fatalf("plan wrote through the symlink: %v", entries)
			}
		})
	}
}

func TestSkillBundleRejections(t *testing.T) {
	tests := map[string]struct {
		mutate func(t *testing.T, root string)
		want   string
	}{
		"symlinked bundle file": {
			mutate: func(t *testing.T, root string) {
				if err := os.Symlink(filepath.Join(root, "profiles"), filepath.Join(root, "skills", "review", "link")); err != nil {
					t.Fatal(err)
				}
			},
			want: "link is a symlink",
		},
		"name differs from folder": {
			mutate: func(t *testing.T, root string) {
				writeInstallTestFile(t, filepath.Join(root, "skills", "review", "SKILL.md"), strings.Replace(reviewSkill, "name: review", "name: other", 1))
			},
			want: `must match its folder name "review"`,
		},
		"missing description": {
			mutate: func(t *testing.T, root string) {
				writeInstallTestFile(t, filepath.Join(root, "skills", "review", "SKILL.md"), "---\nname: review\n---\nbody\n")
			},
			want: "description",
		},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			request, root, _ := skillRequest(t, skillTargets()["claude-code"])
			test.mutate(t, root)
			_, err := BuildPlan(request)
			if err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("error = %v, want %q", err, test.want)
			}
		})
	}
}

func TestSkillProvenanceMismatchNamesActualDigest(t *testing.T) {
	request, root, _ := skillRequest(t, skillTargets()["claude-code"])
	edited := vendoredSkill + "drift\n"
	writeInstallTestFile(t, filepath.Join(root, "skills", "vendored", "SKILL.md"), edited)
	actual := profilemango.SkillTreeDigest([]profilemango.SkillTreeFile{{Path: "SKILL.md", Mode: 0o644, SHA256: installfs.Hash([]byte(edited))}})
	if _, err := BuildPlan(request); err == nil || !strings.Contains(err.Error(), "tree digest is "+actual) {
		t.Fatalf("error = %v, want actual digest %s", err, actual)
	}
}

func TestSkillsSkippedForNamedProfiles(t *testing.T) {
	for name, target := range skillTargets() {
		if target.namedWrites {
			continue
		}
		t.Run(name, func(t *testing.T) {
			request, _, _ := skillRequest(t, target)
			request.Default = false
			plan, err := BuildPlan(request)
			if err != nil {
				t.Fatal(err)
			}
			want := SkippedRequirement{Requirement: RequirementSkills, Count: 2, Reason: skillsNamedOnlyReason}
			if skipped := plan.Targets[0].SkippedRequirements; len(skipped) == 0 || skipped[len(skipped)-1] != want || len(plan.Targets[0].Skills) != 0 {
				t.Fatalf("named plan = %#v", plan.Targets[0])
			}
		})
	}
}
