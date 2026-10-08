package install

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ariel-frischer/profile-mango/internal/installfs"
	"github.com/ariel-frischer/profile-mango/pkg/adapters/openclaw"
	"github.com/ariel-frischer/profile-mango/pkg/profilemango"
	"github.com/ariel-frischer/profile-mango/pkg/render"
)

func TestOpenClawAdapterMetadata(t *testing.T) {
	metadata := (openClawAdapter{}).Metadata()
	if metadata.Target != openclaw.TargetName || metadata.Version != openclaw.TargetVersion {
		t.Fatalf("metadata target = %#v", metadata)
	}
	if !metadata.Installable || metadata.Status != StatusReady || metadata.EvidenceSHA256 != openclaw.EvidenceSHA256 {
		t.Fatalf("metadata did not expose qualified install subset: %#v", metadata)
	}
	for _, term := range []string{"model", "thinking", "resolver", "getter", "fallback", "override"} {
		if !strings.Contains(metadata.Reason, term) {
			t.Fatalf("metadata reason omitted evidence scope %q: %q", term, metadata.Reason)
		}
	}
}

func TestOpenClawAdapterPlanUsesOnlyQualifiedFields(t *testing.T) {
	input := AdapterInput{
		Target:  Target{Name: openclaw.TargetName, Version: openclaw.TargetVersion},
		Profile: profilemango.ResolvedProfile{Metadata: profilemango.Metadata{Name: "route-only"}},
		Route:   openClawTestRoute(),
		Config:  Snapshot{Content: []byte("{unknown:{secret:'SYNTHETIC'}}")},
	}
	patch, err := (openClawAdapter{}).Plan(input)
	if err != nil {
		t.Fatal(err)
	}
	if len(patch.Files) != 1 || !strings.Contains(string(patch.Files[0].Content), "SYNTHETIC") {
		t.Fatalf("patch did not preserve unrelated config: %#v", patch)
	}
	if !patch.OverrideAllowed || len(patch.Fields) != 2 {
		t.Fatalf("patch contract = %#v", patch)
	}
	if patch.Fields[0].Path != "config.agents.defaults.model.primary" || patch.Fields[1].Path != "config.agents.defaults.thinkingDefault" {
		t.Fatalf("patch fields = %#v", patch.Fields)
	}
	if len(patch.Diagnostics) != 1 || patch.Diagnostics[0].Code != "openclaw.install.model_thinking_only" {
		t.Fatalf("patch diagnostics = %#v", patch.Diagnostics)
	}
}

func TestOpenClawAdapterBlocksUnqualifiedRequirements(t *testing.T) {
	mode := "read-only"
	tests := map[string]struct {
		mutate func(*AdapterInput)
		want   string
	}{
		"permissions": {
			mutate: func(input *AdapterInput) {
				input.Profile.Permissions = &profilemango.PermissionPolicy{Mode: &mode}
			},
			want: "permission",
		},
		"tools": {
			mutate: func(input *AdapterInput) {
				input.Profile.Tools = &profilemango.ResolvedRules{Managed: true}
			},
			want: "tool",
		},
		"instructions": {
			mutate: func(input *AdapterInput) { input.Profile.Instructions = []string{"AGENTS.md"} },
			want:   "instruction",
		},
		"skills": {
			mutate: func(input *AdapterInput) { input.Profile.Skills = []profilemango.SkillRef{{Path: "skill/SKILL.md"}} },
			want:   "instruction",
		},
		"resources": {
			mutate: func(input *AdapterInput) {
				input.Resources = []render.Resource{{Digest: profilemango.ResourceDigest{Path: "AGENTS.md"}}}
			},
			want: "instruction",
		},
		"wrong target": {
			mutate: func(input *AdapterInput) { input.Target = Target{Name: "openclaw", Version: "2026.9.6"} },
			want:   "exact target",
		},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			input := openClawTestInput()
			test.mutate(&input)
			_, err := (openClawAdapter{}).Plan(input)
			if err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("error = %v, want %q", err, test.want)
			}
		})
	}
}

func TestOpenClawInstallPlanApplyReapplyBackupAndStale(t *testing.T) {
	request, config := openClawInstallRequest(t)
	original := "{\n  agents: {\n    defaults: {\n      model: { primary: 'old/model', fallbacks: ['keep/model'], },\n      thinkingDefault: 'low',\n    },\n    entries: { main: {}, },\n  },\n  unknown: { secret: 'SYNTHETIC' },\n}\n"
	writeInstallTestFile(t, config, original)
	plan, err := BuildPlan(request)
	if err != nil {
		t.Fatal(err)
	}
	if plan.Status != StatusReady || plan.Targets[0].Status != StatusReady {
		t.Fatalf("plan = %#v", plan)
	}
	if !hasFileAction(plan.Targets[0], ActionOverride) || !hasFileAction(plan.Targets[0], ActionCreate) || len(plan.Targets[0].Fields) != 4 {
		t.Fatalf("plan diff = %#v", plan.Targets[0])
	}
	if _, err := ApplyPlan(plan, ApplyOptions{ExpectedPlanID: plan.PlanID}); err != nil {
		t.Fatal(err)
	}
	patched, err := os.ReadFile(config)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(patched), `primary: "openai/gpt-5.6"`) || !strings.Contains(string(patched), `thinkingDefault: "high"`) || !strings.Contains(string(patched), "SYNTHETIC") {
		t.Fatalf("applied config lost qualified or unrelated state: %s", patched)
	}
	if _, err := os.Stat(installedBackup(t, plan.PlanID, config)); err != nil {
		t.Fatalf("default backup missing: %v", err)
	}
	reapply, err := BuildPlan(request)
	if err != nil {
		t.Fatal(err)
	}
	if reapply.Status != StatusNoop || reapply.Targets[0].Status != StatusNoop {
		t.Fatalf("reapply plan = %#v", reapply)
	}
	stale, err := BuildPlan(request)
	if err != nil {
		t.Fatal(err)
	}
	writeInstallTestFile(t, config, "{unknown:true}\n")
	if _, err := ApplyPlan(stale, ApplyOptions{ExpectedPlanID: stale.PlanID}); !errors.Is(err, installfs.ErrStale) {
		t.Fatalf("stale apply error = %v", err)
	}
	assertInstallTestFile(t, config, "{unknown:true}\n")
}

func TestOpenClawInstallCreatesConfigFromMissingPath(t *testing.T) {
	request, config := openClawInstallRequest(t)
	plan, err := BuildPlan(request)
	if err != nil {
		t.Fatal(err)
	}
	if plan.Status != StatusReady || plan.Targets[0].Files[0].Action != ActionCreate || plan.Targets[0].Files[1].Action != ActionCreate {
		t.Fatalf("create plan = %#v", plan)
	}
	if _, err := ApplyPlan(plan, ApplyOptions{ExpectedPlanID: plan.PlanID}); err != nil {
		t.Fatal(err)
	}
	created := "{agents:{defaults:{model:{primary:\"openai/gpt-5.6\"},thinkingDefault:\"high\"}}}\n"
	assertInstallTestFile(t, config, created)
	assertInstallTestFile(t, openClawProfileConfig(config, request.ProfileName), created)
	if _, err := os.Stat(installedBackup(t, plan.PlanID, config)); !os.IsNotExist(err) {
		t.Fatalf("unexpected backup for newly created config: %v", err)
	}
}

func openClawInstallRequest(t *testing.T) (Request, string) {
	t.Helper()
	request, root := testRequest(t, NewRegistry(openClawAdapter{}))
	writeInstallTestFile(t, request.BindingsPath, `routes:
  primary:
    provider: openai
    transport: native
    authentication: oauth
    model: gpt-5.6
    effort: high
`)
	config := filepath.Join(root, "home", ".openclaw", "openclaw.json")
	if err := os.MkdirAll(filepath.Dir(config), 0o755); err != nil {
		t.Fatal(err)
	}
	request.Override = true
	request.Targets = []TargetRequest{{Target: Target{Name: openclaw.TargetName, Version: openclaw.TargetVersion}, ConfigPath: config}}
	// Most OpenClaw tests cover the default config patch, which only --default writes.
	request.Default = true
	return request, config
}

// openClawProfileConfig is where `openclaw --profile <name>` reads config beside config.
func openClawProfileConfig(config, name string) string {
	return filepath.Join(filepath.Dir(filepath.Dir(config)), ".openclaw-"+name, "openclaw.json")
}

// openClawAdoptRequest also seeds the profile config so --default adopts both files.
func openClawAdoptRequest(t *testing.T) (Request, string) {
	t.Helper()
	request, config := openClawInstallRequest(t)
	profile := openClawProfileConfig(config, request.ProfileName)
	if err := os.MkdirAll(filepath.Dir(profile), 0o700); err != nil {
		t.Fatal(err)
	}
	writeInstallTestFile(t, profile, `{"theme":"dark"}`)
	return request, config
}

func openClawTestInput() AdapterInput {
	return AdapterInput{
		Target:  Target{Name: openclaw.TargetName, Version: openclaw.TargetVersion},
		Profile: profilemango.ResolvedProfile{Metadata: profilemango.Metadata{Name: "route-only"}},
		Route:   openClawTestRoute(),
		Config:  Snapshot{Content: []byte("{}")},
	}
}

func openClawTestRoute() profilemango.RouteBinding {
	return profilemango.RouteBinding{Provider: "openai", Transport: "native", Authentication: "oauth", Model: "gpt-5.6", Effort: "high"}
}
