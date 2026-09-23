package install

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const openCodeTestSkill = `---
name: profile-mango-synthetic
description: Use for synthetic OpenCode qualification tests.
---

Synthetic skill body.
`

func TestOpenCodeSkillInstallAppliesAndReapplies(t *testing.T) {
	request, root := openCodeTestRequest(t)
	profile := filepath.Join(root, "profiles", "route-only", "profile.yaml")
	writeInstallTestFile(t, profile, `apiVersion: profilemango.dev/v1alpha1
kind: PolicyProfile
metadata:
  name: route-only
spec:
  routeRef: primary
  skills:
    - skills/research/SKILL.md
`)
	resource := filepath.Join(root, "skills", "research", "SKILL.md")
	if err := os.MkdirAll(filepath.Dir(resource), 0o755); err != nil {
		t.Fatal(err)
	}
	writeInstallTestFile(t, resource, openCodeTestSkill)

	plan, err := BuildPlan(request)
	if err != nil {
		t.Fatal(err)
	}
	if plan.Status != StatusReady || len(plan.Targets[0].Files) != 3 {
		t.Fatalf("plan = %#v", plan)
	}
	data, err := plan.JSON()
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(data), root) || !strings.Contains(string(data), "config.skills.paths") {
		t.Fatalf("plan leaked target path or omitted skill field: %s", data)
	}
	if _, err := ApplyPlan(plan, ApplyOptions{ExpectedPlanID: plan.PlanID}); err != nil {
		t.Fatal(err)
	}

	config := request.Targets[0].ConfigPath
	configData, err := os.ReadFile(config)
	if err != nil {
		t.Fatal(err)
	}
	configText := string(configData)
	if !strings.Contains(configText, `"skills":{"paths":[`) || !strings.Contains(configText, skillConfigDir(config)) || !strings.Contains(configText, `"model": "openai/gpt-5.6"`) {
		t.Fatalf("config omitted qualified fields: %s", configData)
	}
	assertInstallTestFile(t, filepath.Join(filepath.Dir(config), "SKILL.md"), openCodeTestSkill)

	reapply, err := BuildPlan(request)
	if err != nil {
		t.Fatal(err)
	}
	if reapply.Status != StatusNoop || len(reapply.Targets[0].Files) != 2 {
		t.Fatalf("reapply = %#v", reapply)
	}
}

func TestOpenCodeSkillInstallRejectsStaleConfigBeforeResourceWrite(t *testing.T) {
	request, root := openCodeTestRequest(t)
	profile := filepath.Join(root, "profiles", "route-only", "profile.yaml")
	writeInstallTestFile(t, profile, `apiVersion: profilemango.dev/v1alpha1
kind: PolicyProfile
metadata:
  name: route-only
spec:
  routeRef: primary
  skills:
    - skills/research/SKILL.md
`)
	resource := filepath.Join(root, "skills", "research", "SKILL.md")
	if err := os.MkdirAll(filepath.Dir(resource), 0o755); err != nil {
		t.Fatal(err)
	}
	writeInstallTestFile(t, resource, openCodeTestSkill)
	plan, err := BuildPlan(request)
	if err != nil {
		t.Fatal(err)
	}
	writeInstallTestFile(t, request.Targets[0].ConfigPath, `{ "model": "third-party/edit" }`)
	if _, err := ApplyPlan(plan, ApplyOptions{ExpectedPlanID: plan.PlanID}); err == nil {
		t.Fatal("stale skill plan applied")
	}
	if _, err := os.Stat(filepath.Join(filepath.Dir(request.Targets[0].ConfigPath), "SKILL.md")); !os.IsNotExist(err) {
		t.Fatalf("stale apply wrote skill resource: %v", err)
	}
}

func TestOpenCodeSkillInstallKeepsUnsupportedRequirementsBlocked(t *testing.T) {
	tests := map[string]string{
		"multiple skills": `apiVersion: profilemango.dev/v1alpha1
kind: PolicyProfile
metadata:
  name: route-only
spec:
  routeRef: primary
  skills:
    - skills/one/SKILL.md
    - skills/two/SKILL.md
`,
		"instructions": `apiVersion: profilemango.dev/v1alpha1
kind: PolicyProfile
metadata:
  name: route-only
spec:
  routeRef: primary
  instructions:
    append:
      - instructions/system.md
`,
	}
	for name, profileData := range tests {
		t.Run(name, func(t *testing.T) {
			request, root := openCodeTestRequest(t)
			request.Strict = true
			writeInstallTestFile(t, filepath.Join(root, "profiles", "route-only", "profile.yaml"), profileData)
			if name == "multiple skills" {
				for _, resourcePath := range []string{"skills/one/SKILL.md", "skills/two/SKILL.md"} {
					path := filepath.Join(root, resourcePath)
					if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
						t.Fatal(err)
					}
					writeInstallTestFile(t, path, openCodeTestSkill)
				}
			} else {
				path := filepath.Join(root, "instructions", "system.md")
				if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
					t.Fatal(err)
				}
				writeInstallTestFile(t, path, "synthetic instruction\n")
			}
			plan, err := BuildPlan(request)
			if err != nil {
				t.Fatal(err)
			}
			if plan.Status != StatusBlocked || !strings.Contains(plan.Targets[0].Reason, "install") {
				t.Fatalf("plan = %#v", plan)
			}
		})
	}
}
