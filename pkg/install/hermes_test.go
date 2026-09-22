package install

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gitlab.com/ariel-frischer/profile-mango/internal/installfs"
	"gitlab.com/ariel-frischer/profile-mango/pkg/adapters/hermes"
	"gitlab.com/ariel-frischer/profile-mango/pkg/profilemango"
	"gitlab.com/ariel-frischer/profile-mango/pkg/render"
)

func TestHermesAdapterPlansActualConfigPatch(t *testing.T) {
	adapter := hermesAdapter{}
	patch, err := adapter.Plan(AdapterInput{
		Target:  Target{Name: hermes.TargetName, Version: hermes.TargetVersion},
		Config:  Snapshot{Content: []byte("# keep\nmodel:\n  provider: old\nagent:\n  reasoning_effort: low\n")},
		Route:   hermesInstallRoute(),
		Profile: profilemango.ResolvedProfile{Metadata: profilemango.Metadata{Name: "route-only"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	want := "# keep\nmodel:\n  provider: \"openai\"\n  default: \"gpt-5.6\"\nagent:\n  reasoning_effort: \"high\"\n"
	if string(patch.Files[0].Content) != want {
		t.Fatalf("content = %q, want %q", patch.Files[0].Content, want)
	}
	if !strings.Contains(patch.Diagnostics.Error(), "only model.provider") {
		t.Fatalf("missing bounded-install diagnostic: %#v", patch.Diagnostics)
	}
	if !patch.OverrideAllowed || len(patch.Fields) != 3 {
		t.Fatalf("patch contract = %#v", patch)
	}
}

func TestHermesAdapterBlocksUnqualifiedProfileRequirements(t *testing.T) {
	mode := "read-only"
	_, err := (hermesAdapter{}).Plan(AdapterInput{
		Target: hermesTarget(),
		Route:  hermesInstallRoute(),
		Profile: profilemango.ResolvedProfile{
			Metadata:    profilemango.Metadata{Name: "route-only"},
			Permissions: &profilemango.PermissionPolicy{Mode: &mode},
		},
	})
	if err == nil || !strings.Contains(err.Error(), "permission requirements") {
		t.Fatalf("error = %v", err)
	}
}

func TestHermesAdapterBlocksUnqualifiedDeliveryRequirements(t *testing.T) {
	tests := map[string]struct {
		profile   profilemango.ResolvedProfile
		resources []render.Resource
		want      string
	}{
		"tools": {
			profile: profilemango.ResolvedProfile{Tools: &profilemango.ResolvedRules{Allow: []string{"shell"}}},
			want:    "tool requirements",
		},
		"instructions": {
			profile: profilemango.ResolvedProfile{Instructions: []string{"instructions.md"}},
			want:    "instruction, skill, and resource delivery",
		},
		"skills": {
			profile: profilemango.ResolvedProfile{Skills: []string{"skill.md"}},
			want:    "instruction, skill, and resource delivery",
		},
		"resources": {
			resources: []render.Resource{{Content: []byte("synthetic")}},
			want:      "instruction, skill, and resource delivery",
		},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			_, err := (hermesAdapter{}).Plan(AdapterInput{
				Target: hermesTarget(), Route: hermesInstallRoute(),
				Profile: test.profile, Resources: test.resources,
			})
			if err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("error = %v, want %q", err, test.want)
			}
		})
	}
}

func TestHermesMetadataIsInstallableButBounded(t *testing.T) {
	metadata := (hermesAdapter{}).Metadata()
	if !metadata.Installable || metadata.Status != StatusReady || metadata.EvidenceSHA256 != hermes.EvidenceSHA256 {
		t.Fatalf("metadata = %#v", metadata)
	}
	if !strings.Contains(metadata.Reason, "runtime enforcement") {
		t.Fatalf("metadata reason omitted limitation: %q", metadata.Reason)
	}
}

func TestHermesInstallPreservesStateAndCreatesDefaultBackup(t *testing.T) {
	request, config := hermesInstallRequest(t)
	before := hermesExistingConfig()
	writeInstallTestFile(t, config, before)
	if err := os.Chmod(config, 0o600); err != nil {
		t.Fatal(err)
	}
	plan := buildHermesInstallPlan(t, request)
	if plan.Status != StatusReady || plan.Targets[0].Status != StatusReady {
		t.Fatalf("plan = %#v", plan)
	}
	assertHermesPlanDiff(t, plan.Targets[0])
	if _, err := ApplyPlan(plan, ApplyOptions{ExpectedPlanID: plan.PlanID}); err != nil {
		t.Fatal(err)
	}
	assertInstallTestFile(t, config, hermesPatchedConfig())
	assertInstallTestFile(t, installfs.BackupPath(config, plan.PlanID), before)
	info, err := os.Stat(config)
	if err != nil || info.Mode().Perm() != 0o600 {
		t.Fatalf("config mode = %v, err = %v", info.Mode().Perm(), err)
	}
}

func TestHermesInstallReapplyIsNoop(t *testing.T) {
	request, config := hermesInstallRequest(t)
	writeInstallTestFile(t, config, hermesExistingConfig())
	first := buildHermesInstallPlan(t, request)
	if _, err := ApplyPlan(first, ApplyOptions{ExpectedPlanID: first.PlanID}); err != nil {
		t.Fatal(err)
	}
	second := buildHermesInstallPlan(t, request)
	if second.Status != StatusNoop || second.Targets[0].Files[0].Action != ActionNoop {
		t.Fatalf("reapply plan = %#v", second)
	}
	if _, err := ApplyPlan(second, ApplyOptions{ExpectedPlanID: second.PlanID}); err != nil {
		t.Fatal(err)
	}
	backups, err := filepath.Glob(config + ".profile-mango.bak.*")
	if err != nil || len(backups) != 1 {
		t.Fatalf("backups = %v, err = %v", backups, err)
	}
}

func TestHermesInstallRejectsStaleConfigBeforeWriting(t *testing.T) {
	request, config := hermesInstallRequest(t)
	writeInstallTestFile(t, config, hermesExistingConfig())
	plan := buildHermesInstallPlan(t, request)
	writeInstallTestFile(t, config, "third-party edit\n")
	if _, err := ApplyPlan(plan, ApplyOptions{ExpectedPlanID: plan.PlanID}); !errors.Is(err, installfs.ErrStale) {
		t.Fatalf("stale apply error = %v", err)
	}
	assertInstallTestFile(t, config, "third-party edit\n")
	if _, err := os.Stat(config + ".profile-mango.manifest.json"); !os.IsNotExist(err) {
		t.Fatalf("stale apply created manifest: %v", err)
	}
}

func TestHermesInstallPlanShowsOnlyBoundedDiff(t *testing.T) {
	request, config := hermesInstallRequest(t)
	writeInstallTestFile(t, config, hermesExistingConfig())
	plan := buildHermesInstallPlan(t, request)
	data, err := plan.JSON()
	if err != nil {
		t.Fatal(err)
	}
	text := string(data)
	for _, field := range []string{"model.provider", "model.default", "agent.reasoning_effort"} {
		if !strings.Contains(text, `"path": "`+field+`"`) {
			t.Fatalf("plan omitted bounded diff %q: %s", field, text)
		}
	}
	if strings.Contains(text, "SYNTHETIC_SECRET_SENTINEL") || strings.Contains(text, "third-party edit") {
		t.Fatalf("plan leaked config bytes: %s", text)
	}
}

func hermesInstallRequest(t *testing.T) (Request, string) {
	t.Helper()
	request, root := testRequest(t, NewRegistry(hermesAdapter{}))
	writeInstallTestFile(t, request.BindingsPath, hermesBindings())
	config := filepath.Join(root, "target", "config.yaml")
	if err := os.MkdirAll(filepath.Dir(config), 0o755); err != nil {
		t.Fatal(err)
	}
	request.Override = true
	request.Targets = []TargetRequest{{Target: hermesTarget(), ConfigPath: config}}
	return request, config
}

func buildHermesInstallPlan(t *testing.T, request Request) Plan {
	t.Helper()
	plan, err := BuildPlan(request)
	if err != nil {
		t.Fatal(err)
	}
	return plan
}

func assertHermesPlanDiff(t *testing.T, target TargetPlan) {
	t.Helper()
	got := make(map[string]FieldChange, len(target.Fields))
	for _, field := range target.Fields {
		got[field.Path] = field
	}
	want := map[string]FieldChange{
		"model.provider":         {Path: "model.provider", Before: "old-provider", After: "openai"},
		"model.default":          {Path: "model.default", Before: "old-model", After: "gpt-5.6"},
		"agent.reasoning_effort": {Path: "agent.reasoning_effort", Before: "low", After: "high"},
	}
	for path, field := range want {
		if got[path] != field {
			t.Fatalf("field %q = %#v, want %#v", path, got[path], field)
		}
	}
}

func hermesBindings() string {
	return `routes:
  primary:
    provider: openai
    transport: native
    authentication: oauth
    model: gpt-5.6
    effort: high
`
}

func hermesExistingConfig() string {
	return `# retain header
model:
  provider: old-provider # retain comment
  default: old-model
  extra: keep
agent:
  reasoning_effort: low # retain comment
unknown_sentinel:
  value: SYNTHETIC_SECRET_SENTINEL
`
}

func hermesPatchedConfig() string {
	return `# retain header
model:
  provider: "openai" # retain comment
  default: "gpt-5.6"
  extra: keep
agent:
  reasoning_effort: "high" # retain comment
unknown_sentinel:
  value: SYNTHETIC_SECRET_SENTINEL
`
}

func hermesTarget() Target {
	return Target{Name: hermes.TargetName, Version: hermes.TargetVersion}
}

func hermesInstallRoute() profilemango.RouteBinding {
	return profilemango.RouteBinding{Provider: "openai", Transport: "native", Authentication: "oauth", Model: "gpt-5.6", Effort: "high"}
}
