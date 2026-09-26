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
	request.Default = true
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
	if plan.Status != StatusReady || len(plan.Targets[0].Files) != 4 {
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
	if reapply.Status != StatusNoop || len(reapply.Targets[0].Files) != 3 {
		t.Fatalf("reapply = %#v", reapply)
	}
}

func TestOpenCodeSkillInstallRejectsStaleConfigBeforeResourceWrite(t *testing.T) {
	request, root := openCodeTestRequest(t)
	request.Default = true
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
	tests := map[string]struct {
		profile string
		want    string
	}{
		"multiple skills": {want: "exactly one skill", profile: `apiVersion: profilemango.dev/v1alpha1
kind: PolicyProfile
metadata:
  name: route-only
spec:
  routeRef: primary
  skills:
    - skills/one/SKILL.md
    - skills/two/SKILL.md
`},
		"permissions": {want: "unqualified", profile: `apiVersion: profilemango.dev/v1alpha1
kind: PolicyProfile
metadata:
  name: route-only
spec:
  routeRef: primary
  permissions:
    mode: read-only
`},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			request, root := openCodeTestRequest(t)
			request.Strict, request.Default = true, true
			writeInstallTestFile(t, filepath.Join(root, "profiles", "route-only", "profile.yaml"), test.profile)
			if name == "multiple skills" {
				for _, resourcePath := range []string{"skills/one/SKILL.md", "skills/two/SKILL.md"} {
					path := filepath.Join(root, resourcePath)
					if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
						t.Fatal(err)
					}
					writeInstallTestFile(t, path, openCodeTestSkill)
				}
			}
			plan, err := BuildPlan(request)
			if err != nil {
				t.Fatal(err)
			}
			if plan.Status != StatusBlocked || !strings.Contains(plan.Targets[0].Reason, test.want) {
				t.Fatalf("plan = %#v", plan)
			}
		})
	}
}

func TestOpenCodeDefaultInstallSkipsMultipleSkills(t *testing.T) {
	request, root := openCodeTestRequest(t)
	request.Default = true
	writeInstallTestFile(t, filepath.Join(root, "profiles", "route-only", "profile.yaml"), `apiVersion: profilemango.dev/v1alpha1
kind: PolicyProfile
metadata:
  name: route-only
spec:
  routeRef: primary
  skills:
    - skills/one/SKILL.md
    - skills/two/SKILL.md
`)
	for _, resourcePath := range []string{"skills/one/SKILL.md", "skills/two/SKILL.md"} {
		path := filepath.Join(root, resourcePath)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		writeInstallTestFile(t, path, openCodeTestSkill)
	}
	plan, err := BuildPlan(request)
	if err != nil {
		t.Fatal(err)
	}
	want := SkippedRequirement{Requirement: RequirementSkills, Count: 2, Reason: "installs at most 1 skill; this profile has 2"}
	target := plan.Targets[0]
	if plan.Status != StatusReady || len(target.SkippedRequirements) != 1 || target.SkippedRequirements[0] != want {
		t.Fatalf("plan = %#v", plan)
	}
	for _, file := range target.Files {
		if strings.HasSuffix(file.Path, "SKILL.md") {
			t.Fatalf("skipped skill still written: %s", file.Path)
		}
	}
}
