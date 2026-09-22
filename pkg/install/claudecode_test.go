package install

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gitlab.com/ariel-frischer/profile-mango/internal/installfs"
	"gitlab.com/ariel-frischer/profile-mango/pkg/adapters/claudecode"
	"gitlab.com/ariel-frischer/profile-mango/pkg/profilemango"
)

func TestClaudeCodeInstallMetadata(t *testing.T) {
	metadata := (claudeCodeAdapter{}).Metadata()
	if !metadata.Installable || metadata.Target != claudecode.TargetName || metadata.Version != claudecode.TargetVersion {
		t.Fatalf("metadata = %#v", metadata)
	}
	if !strings.Contains(metadata.Reason, "model") || !strings.Contains(metadata.Reason, "blocked") {
		t.Fatalf("metadata reason does not describe the narrow boundary: %q", metadata.Reason)
	}
}

func TestClaudeCodeInstallPlanPreservesTargetState(t *testing.T) {
	source := []byte("{\r\n  \"unknown\": true,\r\n  \"apiKey\": \"SYNTHETIC-CREDENTIAL\"\r\n}\r\n")
	patch, err := (claudeCodeAdapter{}).Plan(AdapterInput{
		Target:  Target{Name: claudecode.TargetName, Version: claudecode.TargetVersion},
		Profile: profilemango.ResolvedProfile{Metadata: profilemango.Metadata{Name: "route-only"}},
		Route:   claudeInstallRoute(),
		Config:  Snapshot{Content: source},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(patch.Files) != 1 || string(patch.Files[0].Content) != "{\"model\":\"claude-sonnet-4-5\",\r\n  \"unknown\": true,\r\n  \"apiKey\": \"SYNTHETIC-CREDENTIAL\"\r\n}\r\n" {
		t.Fatalf("patch files = %#v", patch.Files)
	}
	if len(patch.Fields) != 1 || patch.Fields[0].Path != "config.model" || patch.Fields[0].After != "claude-sonnet-4-5" {
		t.Fatalf("patch fields = %#v", patch.Fields)
	}
	if !patch.OverrideAllowed || len(patch.Diagnostics) != 1 || patch.Diagnostics[0].Severity != profilemango.SeverityWarning {
		t.Fatalf("patch safety metadata = %#v", patch)
	}
}

func TestClaudeCodeInstallBlocksUnsupportedProfileEffects(t *testing.T) {
	mode := "read-only"
	tests := map[string]struct {
		profile profilemango.ResolvedProfile
		want    string
	}{
		"permissions": {
			profile: profilemango.ResolvedProfile{Permissions: &profilemango.PermissionPolicy{Mode: &mode}},
			want:    "permission requirements",
		},
		"tools": {
			profile: profilemango.ResolvedProfile{Tools: &profilemango.ResolvedRules{Managed: true}},
			want:    "tool requirements",
		},
		"instructions": {
			profile: profilemango.ResolvedProfile{Instructions: []string{"CLAUDE.md"}},
			want:    "instruction and skill delivery",
		},
		"skills": {
			profile: profilemango.ResolvedProfile{Skills: []string{"skills/research/SKILL.md"}},
			want:    "instruction and skill delivery",
		},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			input := AdapterInput{
				Target:  Target{Name: claudecode.TargetName, Version: claudecode.TargetVersion},
				Profile: test.profile,
				Route:   claudeInstallRoute(),
			}
			_, err := (claudeCodeAdapter{}).Plan(input)
			if err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("error = %v, want %q", err, test.want)
			}
		})
	}
}

func TestClaudeCodeInstallPlanApplyReapplyAndStale(t *testing.T) {
	request, root := claudeCodeTestRequest(t)
	request.Override = true
	config := request.Targets[0].ConfigPath
	before := "{\n  \"model\": \"old/model\",\n  \"unknown\": true\n}\n"
	writeInstallTestFile(t, config, before)
	plan, err := BuildPlan(request)
	if err != nil {
		t.Fatal(err)
	}
	if plan.Status != StatusReady || plan.Targets[0].Status != StatusReady {
		t.Fatalf("plan = %#v", plan)
	}
	if _, err := ApplyPlan(plan, ApplyOptions{ExpectedPlanID: plan.PlanID}); err != nil {
		t.Fatal(err)
	}
	assertInstallTestFile(t, config, "{\n  \"model\": \"claude-sonnet-4-5\",\n  \"unknown\": true\n}\n")
	if _, err := os.Stat(installfs.BackupPath(config, plan.PlanID)); err != nil {
		t.Fatalf("backup missing: %v", err)
	}
	reapply, err := BuildPlan(request)
	if err != nil {
		t.Fatal(err)
	}
	if reapply.Targets[0].Files[0].Action != ActionNoop {
		t.Fatalf("reapply action = %s", reapply.Targets[0].Files[0].Action)
	}
	stale, err := BuildPlan(request)
	if err != nil {
		t.Fatal(err)
	}
	writeInstallTestFile(t, config, "{\n  \"model\": \"third-party/edit\"\n}\n")
	if _, err := ApplyPlan(stale, ApplyOptions{ExpectedPlanID: stale.PlanID}); !errors.Is(err, installfs.ErrStale) {
		t.Fatalf("stale apply error = %v", err)
	}
	if _, err := os.Stat(filepath.Join(root, "target")); err != nil {
		t.Fatalf("target directory disappeared: %v", err)
	}
}

func TestClaudeCodeInstallRejectsMalformedAndUnownedConflict(t *testing.T) {
	request, _ := claudeCodeTestRequest(t)
	config := request.Targets[0].ConfigPath
	writeInstallTestFile(t, config, `{ "model": `)
	malformed, err := BuildPlan(request)
	if err != nil {
		t.Fatal(err)
	}
	if malformed.Status != StatusBlocked || !strings.Contains(malformed.Targets[0].Reason, "valid JSON") {
		t.Fatalf("malformed plan = %#v", malformed)
	}
	writeInstallTestFile(t, config, `{ "model": "edited-by-user" }`)
	conflict, err := BuildPlan(request)
	if err != nil {
		t.Fatal(err)
	}
	if conflict.Status != StatusBlocked || conflict.Targets[0].Status != StatusConflict {
		t.Fatalf("conflict plan = %#v", conflict)
	}
}

func claudeCodeTestRequest(t *testing.T) (Request, string) {
	t.Helper()
	root := t.TempDir()
	profiles := filepath.Join(root, "profiles", "route-only")
	if err := os.MkdirAll(profiles, 0o755); err != nil {
		t.Fatal(err)
	}
	writeInstallTestFile(t, filepath.Join(profiles, "profile.yaml"), `apiVersion: profilemango.dev/v1alpha1
kind: PolicyProfile
metadata:
  name: route-only
spec:
  routeRef: primary
`)
	bindingsDir := filepath.Join(root, "bindings")
	if err := os.MkdirAll(bindingsDir, 0o755); err != nil {
		t.Fatal(err)
	}
	bindings := `routes:
  primary:
    provider: anthropic
    transport: native
    authentication: oauth
    model: claude-sonnet-4-5
    effort: high
`
	bindingsPath := filepath.Join(bindingsDir, "local.yaml")
	writeInstallTestFile(t, bindingsPath, bindings)
	config := filepath.Join(root, "target", "settings.json")
	if err := os.MkdirAll(filepath.Dir(config), 0o755); err != nil {
		t.Fatal(err)
	}
	return Request{
		ProfileName:  "route-only",
		ProfilesRoot: filepath.Join(root, "profiles"),
		ResourceRoot: root,
		BindingsPath: bindingsPath,
		Backup:       true,
		Override:     false,
		Registry:     NewRegistry(claudeCodeAdapter{}),
		Targets:      []TargetRequest{{Target: Target{Name: claudecode.TargetName, Version: claudecode.TargetVersion}, ConfigPath: config}},
	}, root
}

func claudeInstallRoute() profilemango.RouteBinding {
	return profilemango.RouteBinding{
		Provider:       "anthropic",
		Transport:      "native",
		Authentication: "oauth",
		Model:          "claude-sonnet-4-5",
		Effort:         "high",
	}
}
