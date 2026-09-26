package install

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ariel-frischer/profile-mango/internal/installfs"
	"github.com/ariel-frischer/profile-mango/pkg/profilemango"
)

func TestCodexInstallPreservesStateBacksUpAndReapplies(t *testing.T) {
	request, _ := codexTestRequest(t)
	config := request.Targets[0].ConfigPath
	before := "# keep\nunknown = true\n[features]\napps = false\n"
	writeInstallTestFile(t, config, before)
	request.Override = true
	if err := os.Chmod(config, 0o600); err != nil {
		t.Fatal(err)
	}
	plan, err := BuildPlan(request)
	if err != nil || plan.Status != StatusReady {
		t.Fatalf("plan = %#v, err=%v", plan, err)
	}
	if !plan.Backup || !hasFileAction(plan.Targets[0], ActionOverride) {
		t.Fatalf("plan backup/action = %v/%#v", plan.Backup, plan.Targets[0].Files)
	}
	if len(plan.Targets[0].Fields) != 6 {
		t.Fatalf("plan fields = %#v", plan.Targets[0].Fields)
	}
	if _, err := ApplyPlan(plan, ApplyOptions{ExpectedPlanID: plan.PlanID}); err != nil {
		t.Fatal(err)
	}
	content, err := os.ReadFile(config)
	if err != nil {
		t.Fatal(err)
	}
	want := "# keep\nunknown = true\nmodel_provider = \"openai\"\nmodel = \"gpt-5.6\"\nmodel_reasoning_effort = \"high\"\n[features]\napps = false\n"
	if string(content) != want {
		t.Fatalf("installed config = %q, want %q", content, want)
	}
	info, err := os.Stat(config)
	if err != nil || info.Mode().Perm() != 0o600 {
		t.Fatalf("config mode = %v, err=%v", info.Mode().Perm(), err)
	}
	backup, err := os.ReadFile(installfs.BackupPath(config, plan.PlanID))
	if err != nil || string(backup) != before {
		t.Fatalf("backup = %q, err=%v", backup, err)
	}
	reapply, err := BuildPlan(request)
	if err != nil || reapply.Status != StatusNoop || !hasFileAction(reapply.Targets[0], ActionNoop) {
		t.Fatalf("reapply = %#v, err=%v", reapply, err)
	}
}

func TestCodexSettingsInstallLeavesAuthenticationUntouched(t *testing.T) {
	tests := map[string]string{
		"forced api":     "forced_login_method = \"api\"\n",
		"forced chatgpt": "forced_login_method = \"chatgpt\"\n",
	}
	for name, authSetting := range tests {
		t.Run(name, func(t *testing.T) {
			request, _ := codexTestRequest(t)
			request.Registry = DefaultRegistry()
			request.Override = true
			config := request.Targets[0].ConfigPath
			authPath := filepath.Join(filepath.Dir(config), "auth.json")
			writeInstallTestFile(t, authPath, "synthetic credential sentinel")
			writeInstallTestFile(t, config, authSetting+"unknown = true\n")
			plan, err := BuildPlan(request)
			if err != nil || plan.Status != StatusReady {
				t.Fatalf("plan = %#v, err=%v", plan, err)
			}
			if _, err := ApplyPlan(plan, ApplyOptions{ExpectedPlanID: plan.PlanID}); err != nil {
				t.Fatal(err)
			}
			content, err := os.ReadFile(config)
			if err != nil || !strings.Contains(string(content), authSetting) {
				t.Fatalf("auth setting changed: content=%q err=%v", content, err)
			}
			assertInstallTestFile(t, authPath, "synthetic credential sentinel")
		})
	}
}

func TestCodexInstallAdoptsUnownedConfigAndAllowsOverride(t *testing.T) {
	request, _ := codexTestRequest(t)
	config := request.Targets[0].ConfigPath
	writeInstallTestFile(t, config, "unknown = true\n")
	adopt, err := BuildPlan(request)
	if err != nil || adopt.Status != StatusReady || !hasFileAction(adopt.Targets[0], ActionAdopt) {
		t.Fatalf("adopt = %#v, err=%v", adopt, err)
	}
	request.Override = true
	override, err := BuildPlan(request)
	if err != nil || override.Status != StatusReady || !hasFileAction(override.Targets[0], ActionOverride) {
		t.Fatalf("override = %#v, err=%v", override, err)
	}
}

func TestCodexInstallRejectsStalePlanWithoutWriting(t *testing.T) {
	request, _ := codexTestRequest(t)
	request.Override = true
	config := request.Targets[0].ConfigPath
	writeInstallTestFile(t, config, "unknown = true\n")
	plan, err := BuildPlan(request)
	if err != nil || plan.Status != StatusReady {
		t.Fatalf("plan = %#v, err=%v", plan, err)
	}
	writeInstallTestFile(t, config, "third-party = true\n")
	if _, err := ApplyPlan(plan, ApplyOptions{ExpectedPlanID: plan.PlanID}); !errors.Is(err, installfs.ErrStale) {
		t.Fatalf("stale apply error = %v", err)
	}
	assertInstallTestFile(t, config, "third-party = true\n")
}

func TestCodexStrictInstallBlocksPermissionRequirements(t *testing.T) {
	request, _ := codexTestRequest(t)
	request.Strict = true
	profile := filepath.Join(request.ProfilesRoot, "route-only", "profile.yaml")
	writeInstallTestFile(t, profile, `apiVersion: profilemango.dev/v1alpha1
kind: PolicyProfile
metadata:
  name: route-only
spec:
  routeRef: primary
  permissions:
    mode: read-only
`)
	plan, err := BuildPlan(request)
	if err != nil || plan.Status != StatusBlocked || !strings.Contains(plan.Targets[0].Reason, "permission requirements") {
		t.Fatalf("plan = %#v, err=%v", plan, err)
	}
}

func TestCodexInstallRejectsPrecedenceAndProviderShadowState(t *testing.T) {
	tests := map[string]struct {
		config string
		want   string
	}{
		"legacy profile selector": {
			config: "profile = \"work\"\n",
			want:   "legacy profile = setting",
		},
		"legacy table for the same profile": {
			config: "[profiles.route-only]\nmodel = \"profile-model\"\n",
			want:   "legacy [profiles.route-only] table",
		},
		"provider shadow": {
			config: "[model_providers.openai]\nname = \"shadow\"\n",
			want:   "provider override state",
		},
		"provider endpoint override": {
			config: "openai_base_url = \"https://shadow.invalid/v1\"\n",
			want:   "provider override state",
		},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			request, _ := codexTestRequest(t)
			config := request.Targets[0].ConfigPath
			writeInstallTestFile(t, config, test.config)
			plan, err := BuildPlan(request)
			if err != nil {
				t.Fatal(err)
			}
			if plan.Status != StatusBlocked || plan.Targets[0].Status != StatusBlocked || !strings.Contains(plan.Targets[0].Reason, test.want) {
				t.Fatalf("plan = %#v, want blocked reason containing %q", plan, test.want)
			}
			assertInstallTestFile(t, config, test.config)
		})
	}
}

func TestCodexInstallRejectsMalformedTOMLWithoutWriting(t *testing.T) {
	request, _ := codexTestRequest(t)
	request.Override = true
	config := request.Targets[0].ConfigPath
	source := "private_key = credential_canary_12345\n"
	writeInstallTestFile(t, config, source)
	plan, err := BuildPlan(request)
	if err != nil {
		t.Fatal(err)
	}
	if plan.Status != StatusBlocked || !hasDiagnostic(plan.Targets[0].Diagnostics, "install.adapter_plan_failed") {
		t.Fatalf("malformed config plan status = %q, diagnostics = %#v", plan.Status, plan.Targets[0].Diagnostics)
	}
	if !strings.Contains(plan.Targets[0].Reason, "invalid codex TOML at line 1") || strings.Contains(plan.Targets[0].Reason, "credential_canary") {
		t.Fatalf("malformed config reason = %q", plan.Targets[0].Reason)
	}
	assertInstallTestFile(t, config, source)
}

func TestCodexInstallReportsNativeEvidenceAndBoundedPrecedence(t *testing.T) {
	request, _ := codexTestRequest(t)
	writeInstallTestFile(t, request.Targets[0].ConfigPath, "unknown = true\n")
	plan, err := BuildPlan(request)
	if err != nil {
		t.Fatal(err)
	}
	if !hasDiagnostic(plan.Targets[0].Diagnostics, "codex.install.route_fields_source_qualified") || !hasDiagnostic(plan.Targets[0].Diagnostics, "codex.install.precedence_bounded") {
		t.Fatalf("target diagnostics = %#v", plan.Targets[0].Diagnostics)
	}
	for _, diagnostic := range plan.Targets[0].Diagnostics {
		if diagnostic.Code != "codex.install.route_fields_source_qualified" {
			continue
		}
		if !strings.Contains(diagnostic.Message, "installed Codex 0.154.0 binary") || strings.Contains(diagnostic.Message, "installed-binary equivalence") {
			t.Fatalf("native evidence warning = %q", diagnostic.Message)
		}
	}
}

func TestCodexInstallRestoresSyntheticConfigThroughTransactionEngine(t *testing.T) {
	request, _ := codexTestRequest(t)
	request.Override = true
	config := request.Targets[0].ConfigPath
	before := "model = \"old/model\"\n"
	writeInstallTestFile(t, config, before)
	original, err := installfs.SnapshotFile(config)
	if err != nil {
		t.Fatal(err)
	}
	plan, err := BuildPlan(request)
	if err != nil || plan.Status != StatusReady {
		t.Fatalf("restore plan = %#v, err=%v", plan, err)
	}
	if _, err := ApplyPlan(plan, ApplyOptions{ExpectedPlanID: plan.PlanID}); err != nil {
		t.Fatal(err)
	}
	current, err := installfs.SnapshotFile(config)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := installfs.Apply([]installfs.Change{{Path: config, Before: current, Content: []byte(before)}}, installfs.ApplyOptions{PlanID: "codex-restore", Backup: false}); err != nil {
		t.Fatal(err)
	}
	restored, err := installfs.SnapshotFile(config)
	if err != nil || restored.SHA256 != original.SHA256 || restored.Mode != original.Mode || string(restored.Content) != before {
		t.Fatalf("restored = %#v, err=%v", restored, err)
	}
}

func TestCodexMetadataReportsSettingsOnlyQualification(t *testing.T) {
	metadata := (codexAdapter{}).Metadata()
	if !metadata.Installable || metadata.Status != StatusReady {
		t.Fatalf("metadata = %#v, want settings-only installable", metadata)
	}
	for _, required := range []string{"settings-only", "model_provider", "installed Codex 0.154.0 binary", "OAuth identity"} {
		if !strings.Contains(metadata.Reason, required) {
			t.Fatalf("metadata reason = %q, want %q", metadata.Reason, required)
		}
	}
	if strings.Contains(metadata.Reason, "installed-binary equivalence") || strings.Contains(metadata.Reason, "project/runtime precedence") {
		t.Fatalf("metadata reason retains obsolete uncertainty: %q", metadata.Reason)
	}
}

type codexTestAdapter struct{}

func (codexTestAdapter) Metadata() AdapterMetadata {
	metadata := (codexAdapter{}).Metadata()
	metadata.Installable = true
	metadata.Status = StatusReady
	metadata.Reason = "test-only contract coverage for the Codex patcher"
	return metadata
}

func (codexTestAdapter) Plan(input AdapterInput) (Patch, error) {
	return (codexAdapter{}).Plan(input)
}

func (codexTestAdapter) NamedProfileFile(name string) (string, error) {
	return (codexAdapter{}).NamedProfileFile(name)
}

func (codexTestAdapter) NamedProfileUse(name string) string {
	return (codexAdapter{}).NamedProfileUse(name)
}

func codexTestRequest(t *testing.T) (Request, string) {
	t.Helper()
	request, root := testRequest(t, NewRegistry(codexTestAdapter{}))
	writeInstallTestFile(t, request.BindingsPath, `routes:
  primary:
    provider: openai
    transport: native
    authentication: oauth
    model: gpt-5.6
    effort: high
`)
	config := filepath.Join(root, "target", "config.toml")
	if err := os.MkdirAll(filepath.Dir(config), 0o755); err != nil {
		t.Fatal(err)
	}
	request.Targets = []TargetRequest{{Target: Target{Name: "codex", Version: "0.154.0"}, ConfigPath: config}}
	// Most Codex tests cover the config.toml patch, which only --default writes.
	request.Default = true
	return request, root
}

func hasFileAction(target TargetPlan, action string) bool {
	for _, file := range target.Files {
		if file.Action == action {
			return true
		}
	}
	return false
}

func hasDiagnostic(diagnostics profilemango.Diagnostics, code string) bool {
	for _, diagnostic := range diagnostics {
		if diagnostic.Code == code {
			return true
		}
	}
	return false
}
