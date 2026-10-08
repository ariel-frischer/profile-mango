package install

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ariel-frischer/profile-mango/internal/installfs"
)

type adoptCase struct {
	request  func(*testing.T) (Request, string)
	existing string
	sentinel string
	skill    bool
	// openCodeDefault requests --default so an opencode case's main config is adopted;
	// without it, an agent-empty opencode install no longer touches the main config.
	openCodeDefault bool
}

func adoptCases() map[string]adoptCase {
	json := `{"theme":"dark"}`
	return map[string]adoptCase{
		"claude-code":     {request: claudeCodeTestRequest, existing: json, sentinel: "theme"},
		"codex":           {request: codexTestRequest, existing: "theme = \"dark\"\n", sentinel: "theme"},
		"pi":              {request: piInstallTestRequest, existing: json, sentinel: "theme"},
		"oh-my-pi":        {request: ohMyPiTestRequest, existing: "theme: dark\n", sentinel: "theme"},
		"openclaw":        {request: openClawAdoptRequest, existing: json, sentinel: "theme"},
		"hermes":          {request: hermesInstallRequest, existing: "theme: dark\n", sentinel: "theme"},
		"opencode":        {request: openCodeTestRequest, existing: json, sentinel: "theme", openCodeDefault: true},
		"opencode skills": {request: openCodeTestRequest, existing: json, sentinel: "theme", skill: true, openCodeDefault: true},
	}
}

func adoptRequest(t *testing.T, test adoptCase) (Request, string) {
	t.Helper()
	request, rootOrConfig := test.request(t)
	if test.skill {
		addOpenCodeTestSkill(t, rootOrConfig)
	}
	request.Backup, request.Override = true, false
	if test.openCodeDefault {
		request.Default = true
	}
	config := request.Targets[0].ConfigPath
	writeInstallTestFile(t, config, test.existing)
	return request, config
}

func TestInstallAdoptsUnownedFieldPatchConfigWithBackup(t *testing.T) {
	for name, test := range adoptCases() {
		t.Run(name, func(t *testing.T) {
			request, config := adoptRequest(t, test)
			plan, err := BuildPlan(request)
			if err != nil {
				t.Fatal(err)
			}
			target := plan.Targets[0]
			if plan.Status != StatusReady || target.Status != StatusReady {
				t.Fatalf("plan status=%s target=%s diagnostics=%v", plan.Status, target.Status, target.Diagnostics)
			}
			if file := piFilePlan(target, filepath.Base(config)); file.Action != ActionAdopt || file.Owned {
				t.Fatalf("config file plan = %#v", file)
			}
			if !hasDiagnostic(target.Diagnostics, "install.adopt_backup") {
				t.Fatalf("plan does not note the adoption backup: %v", target.Diagnostics)
			}
			if _, err := ApplyPlan(plan, ApplyOptions{ExpectedPlanID: plan.PlanID}); err != nil {
				t.Fatal(err)
			}
			assertInstallTestFile(t, installfs.BackupPath(config, plan.PlanID), test.existing)
			data, err := os.ReadFile(config)
			if err != nil || !strings.Contains(string(data), test.sentinel) {
				t.Fatalf("adopted config lost unrelated state: %q err=%v", data, err)
			}
			reapply, err := BuildPlan(request)
			if err != nil || reapply.Status != StatusNoop {
				t.Fatalf("reapply after adoption = %s err=%v", reapply.Status, err)
			}
		})
	}
}

func TestInstallAdoptionRequiresBackup(t *testing.T) {
	for name, test := range adoptCases() {
		t.Run(name, func(t *testing.T) {
			request, config := adoptRequest(t, test)
			request.Backup = false
			plan, err := BuildPlan(request)
			if err != nil {
				t.Fatal(err)
			}
			if plan.Status != StatusBlocked || plan.Targets[0].Status != StatusConflict || !hasDiagnostic(plan.Targets[0].Diagnostics, "install.adopt_requires_backup") || !strings.Contains(plan.Targets[0].Reason, "--no-backup") {
				t.Fatalf("no-backup adoption was not blocked: %#v", plan.Targets[0])
			}
			assertInstallTestFile(t, config, test.existing)
		})
	}
}

func TestInstallAdoptionKeepsWholeFileAndEditedConflicts(t *testing.T) {
	tests := map[string]func(*testing.T) Request{
		"unowned named agent": func(t *testing.T) Request {
			request, _ := namedOpenCodeRequest(t, "primary")
			writeInstallTestFile(t, request.Targets[0].ConfigPath, "user-owned agent\n")
			return request
		},
		"edited owned config": func(t *testing.T) Request {
			request, _ := claudeCodeTestRequest(t)
			plan, err := BuildPlan(request)
			if err != nil {
				t.Fatal(err)
			}
			if _, err := ApplyPlan(plan, ApplyOptions{ExpectedPlanID: plan.PlanID}); err != nil {
				t.Fatal(err)
			}
			writeInstallTestFile(t, request.Targets[0].ConfigPath, `{"model":"edited-by-user"}`)
			return request
		},
	}
	for name, setup := range tests {
		t.Run(name, func(t *testing.T) {
			request := setup(t)
			request.Backup = true
			plan, err := BuildPlan(request)
			if err != nil {
				t.Fatal(err)
			}
			target := plan.Targets[0]
			if plan.Status != StatusBlocked || target.Status == StatusReady || hasFileAction(target, ActionAdopt) {
				t.Fatalf("conflict was weakened: status=%s target=%#v", plan.Status, target)
			}
		})
	}
}
