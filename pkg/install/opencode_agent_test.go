package install

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func namedOpenCodeRequest(t *testing.T, mode string) (Request, string) {
	t.Helper()
	request, root := openCodeTestRequest(t)
	request.Targets[0].Agent = AgentDestination{Mode: mode, Name: "mango-review"}
	request.Targets[0].ConfigPath = filepath.Join(root, "target", "agents", "mango-review.md")
	if err := os.MkdirAll(filepath.Dir(request.Targets[0].ConfigPath), 0o755); err != nil {
		t.Fatal(err)
	}
	writeInstallTestFile(t, filepath.Join(root, "profiles", "route-only", "profile.yaml"), `apiVersion: profilemango.dev/v1alpha1
kind: PolicyProfile
metadata:
  name: route-only
spec:
  routeRef: primary
  instructions:
    append:
      - instructions/first.md
      - instructions/second.md
`)
	for name, body := range map[string]string{"first": "First directive.\n", "second": "Second directive.\n"} {
		path := filepath.Join(root, "instructions", name+".md")
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		writeInstallTestFile(t, path, body)
	}
	return request, root
}

func TestNamedOpenCodeAgentPlanApplyAndReapply(t *testing.T) {
	for _, mode := range []string{"primary", "subagent"} {
		t.Run(mode, func(t *testing.T) {
			request, root := namedOpenCodeRequest(t, mode)
			plan, err := BuildPlan(request)
			if err != nil {
				t.Fatal(err)
			}
			if plan.Status != StatusReady {
				t.Fatalf("plan: %#v", plan.Targets[0])
			}
			data, err := plan.JSON()
			if err != nil {
				t.Fatal(err)
			}
			if !strings.Contains(string(data), `"mode": "`+mode+`"`) || strings.Contains(string(data), root) {
				t.Fatalf("public plan: %s", data)
			}
			if _, err := ApplyPlan(plan, ApplyOptions{ExpectedPlanID: plan.PlanID}); err != nil {
				t.Fatal(err)
			}
			content, err := os.ReadFile(request.Targets[0].ConfigPath)
			if err != nil {
				t.Fatal(err)
			}
			text := string(content)
			if !strings.Contains(text, "mode: "+mode) || !strings.Contains(text, `model: "openai/gpt-5.6"`) || strings.Index(text, "First directive.") > strings.Index(text, "Second directive.") {
				t.Fatalf("definition: %s", text)
			}
			if _, err := os.Stat(filepath.Join(root, "target", "opencode.jsonc")); !os.IsNotExist(err) {
				t.Fatalf("main config touched: %v", err)
			}
			reapply, err := BuildPlan(request)
			if err != nil || reapply.Status != StatusNoop {
				t.Fatalf("reapply: %v, %#v", err, reapply)
			}
		})
	}
}

func TestNamedOpenCodeAgentConflictsAndConsent(t *testing.T) {
	request, _ := namedOpenCodeRequest(t, "primary")
	plan, err := BuildPlan(request)
	if err != nil {
		t.Fatal(err)
	}
	request.Targets[0].Agent.Mode = "subagent"
	changed, err := BuildPlan(request)
	if err != nil || plan.PlanID == changed.PlanID {
		t.Fatalf("mode did not bind consent: %v", err)
	}
	request.Targets[0].Agent.Mode = "primary"
	writeInstallTestFile(t, request.Targets[0].ConfigPath, "unowned\n")
	blocked, err := BuildPlan(request)
	if err != nil || blocked.Status != StatusBlocked {
		t.Fatalf("unowned collision: %v, %#v", err, blocked)
	}
	if _, err := ApplyPlan(blocked, ApplyOptions{ExpectedPlanID: blocked.PlanID}); err == nil {
		t.Fatal("collision applied")
	}
}

func TestNamedOpenCodeAgentSameBytesUnownedStillConflict(t *testing.T) {
	request, _ := namedOpenCodeRequest(t, "primary")
	plan, err := BuildPlan(request)
	if err != nil || plan.Status != StatusReady {
		t.Fatalf("initial plan: %v, %#v", err, plan)
	}
	writeInstallTestFile(t, request.Targets[0].ConfigPath, string(plan.Targets[0].changes[0].Content))
	blocked, err := BuildPlan(request)
	if err != nil || blocked.Status != StatusBlocked || !strings.Contains(blocked.Targets[0].Reason, "unowned") {
		t.Fatalf("same-byte collision: %v, %#v", err, blocked)
	}
}

func TestNamedOpenCodeAgentRejectsInvalidDestinationBeforeRead(t *testing.T) {
	tests := map[string]AgentDestination{
		"built-in":  {Mode: "primary", Name: "build"},
		"traversal": {Mode: "primary", Name: "../escape"},
		"uppercase": {Mode: "subagent", Name: "Review"},
		"bad-mode":  {Mode: "other", Name: "mango-review"},
	}
	for name, agent := range tests {
		t.Run(name, func(t *testing.T) {
			request, _ := namedOpenCodeRequest(t, "primary")
			request.Targets[0].Agent = agent
			plan, err := BuildPlan(request)
			if err != nil || plan.Status != StatusBlocked {
				t.Fatalf("invalid agent: %v, %#v", err, plan)
			}
		})
	}
}

func TestNamedOpenCodeAgentRejectsWrongPathAndTarget(t *testing.T) {
	for name, change := range map[string]func(*Request){
		"wrong directory": func(r *Request) {
			r.Targets[0].ConfigPath = filepath.Join(filepath.Dir(filepath.Dir(r.Targets[0].ConfigPath)), "mango-review.md")
		},
		"wrong basename": func(r *Request) {
			r.Targets[0].ConfigPath = filepath.Join(filepath.Dir(r.Targets[0].ConfigPath), "other.md")
		},
		"unsupported target": func(r *Request) { r.Targets[0].Target = Target{Name: "opencode", Version: "1.18.30"} },
	} {
		t.Run(name, func(t *testing.T) {
			request, _ := namedOpenCodeRequest(t, "primary")
			change(&request)
			plan, err := BuildPlan(request)
			if err != nil || plan.Status != StatusBlocked {
				t.Fatalf("unsafe destination: %v, %#v", err, plan)
			}
		})
	}
}

func TestNamedOpenCodeAgentRequiredPolicyBlocksUnderStrict(t *testing.T) {
	for name, test := range map[string]struct{ extra, want string }{
		"permissions": {extra: "  permissions:\n    mode: read-only\n    network: deny\n    shell: deny\n", want: "unqualified"},
		"tools":       {extra: "  tools:\n    allow: [read]\n    deny: [write]\n", want: "unqualified"},
		"skills":      {extra: "  skills:\n    - skills/one/SKILL.md\n", want: skillsAgentReason},
	} {
		t.Run(name, func(t *testing.T) {
			request, root := namedOpenCodeRequest(t, "primary")
			request.Strict = true
			profile := filepath.Join(root, "profiles", "route-only", "profile.yaml")
			writeInstallTestFile(t, profile, string(mustReadNamedTestFile(t, profile))+test.extra)
			if name == "skills" {
				path := filepath.Join(root, "skills", "one", "SKILL.md")
				if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
					t.Fatal(err)
				}
				writeInstallTestFile(t, path, strings.Replace(openCodeTestSkill, "profile-mango-synthetic", "one", 1))
			}
			plan, err := BuildPlan(request)
			if err != nil || plan.Status != StatusBlocked || !strings.Contains(plan.Targets[0].Reason, test.want) {
				t.Fatalf("required %s accepted: %v, %#v", name, err, plan)
			}
		})
	}
}

func mustReadNamedTestFile(t *testing.T, path string) []byte {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return data
}

func TestNamedOpenCodeAgentSwitchAndOmitInstructions(t *testing.T) {
	request, root := namedOpenCodeRequest(t, "subagent")
	applyNamedTestPlan(t, request)
	first, err := os.ReadFile(request.Targets[0].ConfigPath)
	if err != nil {
		t.Fatal(err)
	}
	other := filepath.Join(root, "profiles", "other", "profile.yaml")
	if err := os.MkdirAll(filepath.Dir(other), 0o755); err != nil {
		t.Fatal(err)
	}
	writeInstallTestFile(t, other, `apiVersion: profilemango.dev/v1alpha1
kind: PolicyProfile
metadata:
  name: other
spec:
  routeRef: primary
  instructions:
    append:
      - instructions/second.md
`)
	request.ProfileName = "other"
	applyNamedTestPlan(t, request)
	second, err := os.ReadFile(request.Targets[0].ConfigPath)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(second), "First directive.") || !strings.Contains(string(second), "Second directive.") {
		t.Fatalf("switch: %s", second)
	}
	writeInstallTestFile(t, other, `apiVersion: profilemango.dev/v1alpha1
kind: PolicyProfile
metadata:
  name: other
spec:
  routeRef: primary
`)
	applyNamedTestPlan(t, request)
	third, err := os.ReadFile(request.Targets[0].ConfigPath)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(third), "Second directive.") {
		t.Fatalf("omission: %s", third)
	}
	request.ProfileName = "route-only"
	applyNamedTestPlan(t, request)
	last, err := os.ReadFile(request.Targets[0].ConfigPath)
	if err != nil || string(last) != string(first) {
		t.Fatalf("switch back: %v, %s", err, last)
	}
}

func applyNamedTestPlan(t *testing.T, request Request) {
	t.Helper()
	plan, err := BuildPlan(request)
	if err != nil || plan.Status != StatusReady {
		t.Fatalf("plan: %v, %#v", err, plan)
	}
	if _, err := ApplyPlan(plan, ApplyOptions{ExpectedPlanID: plan.PlanID}); err != nil {
		t.Fatal(err)
	}
}

func TestNamedOpenCodeAgentEditedAndModeSwitchBlocked(t *testing.T) {
	request, _ := namedOpenCodeRequest(t, "primary")
	applyNamedTestPlan(t, request)
	request.Targets[0].Agent.Mode = "subagent"
	plan, err := BuildPlan(request)
	if err != nil || plan.Status != StatusBlocked || !strings.Contains(plan.Targets[0].Reason, "different agent destination") {
		t.Fatalf("mode switch: %v, %#v", err, plan)
	}
	request.Targets[0].Agent.Mode = "primary"
	writeInstallTestFile(t, request.Targets[0].ConfigPath, "user edit\n")
	plan, err = BuildPlan(request)
	if err != nil || plan.Status != StatusBlocked || !strings.Contains(plan.Targets[0].Reason, "edited") {
		t.Fatalf("edited: %v, %#v", err, plan)
	}
}

func TestNamedOpenCodeAgentStalePlanAndSymlink(t *testing.T) {
	request, root := namedOpenCodeRequest(t, "primary")
	plan, err := BuildPlan(request)
	if err != nil || plan.Status != StatusReady {
		t.Fatalf("plan: %v, %#v", err, plan)
	}
	writeInstallTestFile(t, request.Targets[0].ConfigPath, "racing user\n")
	if _, err := ApplyPlan(plan, ApplyOptions{ExpectedPlanID: plan.PlanID}); err == nil {
		t.Fatal("stale plan applied")
	}
	if _, err := os.Stat(request.Targets[0].ConfigPath + ".profile-mango.manifest.json"); !os.IsNotExist(err) {
		t.Fatalf("manifest written: %v", err)
	}
	if err := os.Remove(request.Targets[0].ConfigPath); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(filepath.Join(root, "instructions", "first.md"), request.Targets[0].ConfigPath); err != nil {
		t.Fatal(err)
	}
	blocked, err := BuildPlan(request)
	if err != nil || blocked.Status != StatusBlocked {
		t.Fatalf("symlink: %v, %#v", err, blocked)
	}
}
