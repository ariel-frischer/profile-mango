package install

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// openClawAgentsBase defines two agents; coder already has an allowlist and the
// comments, quoting, and unrelated keys must survive every patch byte for byte.
const openClawAgentsBase = "// keep\n{\n  agents: {\n    defaults: { model: { primary: 'openai/gpt-5.6' }, thinkingDefault: 'high' },\n    entries: {\n      main: { name: 'Main' },\n      coder: { skills: [\"old\"], /* note */ tools: {} },\n    },\n  },\n  gateway: { port: 1234 },\n}\n"

// openClawSkillsRequest is a default install of the two-skill profile over
// openClawAgentsBase, setting agent's allowlist when agent is non-empty.
func openClawSkillsRequest(t *testing.T, agent string) (Request, string, string) {
	t.Helper()
	request, config := openClawInstallRequest(t)
	root := filepath.Dir(filepath.Dir(filepath.Dir(config)))
	request.Env = PathEnv{UserHome: func() (string, error) { return filepath.Join(root, "home"), nil }}
	request.Backup, request.Release = true, true
	request.Targets[0].SkillAgent = agent
	writeSkillPackage(t, root, true)
	writeInstallTestFile(t, config, openClawAgentsBase)
	return request, root, config
}

func fieldChange(target TargetPlan, path string) (FieldChange, bool) {
	for _, field := range target.Fields {
		if field.Path == path {
			return field, true
		}
	}
	return FieldChange{}, false
}

func TestOpenClawSkillAllowlist(t *testing.T) {
	tests := map[string]struct {
		agent, path, before string
	}{
		"replaces an existing allowlist": {agent: "coder", path: "config.agents.entries.coder.skills", before: `["old"]`},
		"adds an absent allowlist":       {agent: "main", path: "config.agents.entries.main.skills", before: ""},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			request, _, config := openClawSkillsRequest(t, test.agent)
			plan := applySkillPlan(t, request)
			field, found := fieldChange(plan.Targets[0], test.path)
			if !found || field.Before != test.before || field.After != `["review","vendored"]` {
				t.Fatalf("allowlist field = %#v found=%v in %v", field, found, plan.Targets[0].Fields)
			}
			data, err := os.ReadFile(config)
			if err != nil {
				t.Fatal(err)
			}
			if got := strings.Count(string(data), `skills:["review","vendored"]`) + strings.Count(string(data), `skills: ["review","vendored"]`); got != 1 {
				t.Fatalf("config = %s", data)
			}
			assertSkillFile(t, filepath.Join(filepath.Dir(config), "skills", "review", "SKILL.md"), reviewSkill, 0o644)
			if status := skillStatus(t, request); status.Source != SourceCurrent {
				t.Fatalf("status = %s (%s)", status.Source, status.SourceReason)
			}
			request.Targets[0].SkillAgent = ""
			again := buildSkillPlan(t, request, StatusNoop)
			if field, found := fieldChange(again.Targets[0], test.path); !found || field.Before != field.After {
				t.Fatalf("replan without --agent dropped or changed the allowlist: %#v found=%v", field, found)
			}
		})
	}
}

// The exact bytes written for coder: only the allowlist value changes.
func TestOpenClawSkillAllowlistPreservesUnrelatedBytes(t *testing.T) {
	request, _, config := openClawSkillsRequest(t, "coder")
	applySkillPlan(t, request)
	assertInstallTestFile(t, config, strings.Replace(openClawAgentsBase, `skills: ["old"]`, `skills: ["review","vendored"]`, 1))
}

// Switching to a profile without skills releases the folders and gives coder its prior
// allowlist back; undo of that switch restores both.
func TestOpenClawSkillSwitchReleasesAndUndoRestores(t *testing.T) {
	request, root, config := openClawSkillsRequest(t, "coder")
	applySkillPlan(t, request)
	installed, err := os.ReadFile(config)
	if err != nil {
		t.Fatal(err)
	}
	writeSkillPackage(t, root, false)
	request.Targets[0].SkillAgent = ""
	plan := applySkillPlan(t, request)
	if field, found := fieldChange(plan.Targets[0], "config.agents.entries.coder.skills"); !found || field.After != `["old"]` {
		t.Fatalf("release field = %#v found=%v", field, found)
	}
	assertInstallTestFile(t, config, openClawAgentsBase)
	skills := filepath.Join(filepath.Dir(config), "skills")
	if _, err := os.Lstat(filepath.Join(skills, "review")); !os.IsNotExist(err) {
		t.Fatalf("released skill folder remains: %v", err)
	}
	applyUndo(t, UndoRequest{Target: request.Targets[0].Target, ConfigPath: config, Registry: request.Registry})
	assertInstallTestFile(t, config, string(installed))
	assertSkillFile(t, filepath.Join(skills, "review", "scripts", "check.sh"), reviewScript, 0o755)
}

// Moving --agent to another agent gives the first agent its prior allowlist back.
func TestOpenClawSkillAgentSwitchReleasesPreviousAgent(t *testing.T) {
	request, _, config := openClawSkillsRequest(t, "coder")
	applySkillPlan(t, request)
	request.Targets[0].SkillAgent = "main"
	applySkillPlan(t, request)
	want := strings.Replace(openClawAgentsBase, `main: { name: 'Main' }`, `main: {skills:["review","vendored"], name: 'Main' }`, 1)
	assertInstallTestFile(t, config, want)
}

func TestOpenClawSkillAgentMustExist(t *testing.T) {
	tests := map[string]struct{ agent, want string }{
		"undefined agent": {agent: "ghost", want: "agents.entries.ghost"},
		"invalid id":      {agent: "Bad Id", want: "openclaw agent id"},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			request, _, _ := openClawSkillsRequest(t, test.agent)
			plan := buildSkillPlan(t, request, StatusBlocked)
			if target := plan.Targets[0]; !strings.Contains(target.Reason, test.want) {
				t.Fatalf("reason = %q, want %q", target.Reason, test.want)
			}
		})
	}
}

// A named profile writes its skills into its own state directory, not the default one.
func TestOpenClawNamedProfileWritesSkillsToProfileState(t *testing.T) {
	request, root, config := openClawSkillsRequest(t, "")
	request.Default, request.ProfileName = false, "route-only"
	applySkillPlan(t, request)
	named := filepath.Join(filepath.Dir(openClawProfileConfig(config, "route-only")), "skills")
	assertSkillFile(t, filepath.Join(named, "vendored", "SKILL.md"), vendoredSkill, 0o644)
	if _, err := os.Lstat(filepath.Join(filepath.Dir(config), "skills")); !os.IsNotExist(err) {
		t.Fatalf("default state received skills: %v", err)
	}
	writeSkillPackage(t, root, false)
	applySkillPlan(t, request)
	if _, err := os.Lstat(filepath.Join(named, "vendored")); !os.IsNotExist(err) {
		t.Fatalf("released named skill folder remains: %v", err)
	}
}
