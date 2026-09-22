package install

import (
	"errors"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"gitlab.com/ariel-frischer/profile-mango/internal/installfs"
	"gitlab.com/ariel-frischer/profile-mango/pkg/profilemango"
)

func TestOhMyPiInstallPreservesStateAndReapplies(t *testing.T) {
	request, _ := ohMyPiTestRequest(t)
	config := request.Targets[0].ConfigPath
	before := "# keep\nmodelRoles:\n  reviewer: other/model\n  default: old/model # owned\ndefaultThinkingLevel: low # owned\nunknown:\n  apiKey: SYNTHETIC\n"
	writeInstallTestFile(t, config, before)
	if err := os.Chmod(config, 0o600); err != nil {
		t.Fatal(err)
	}

	plan, err := BuildPlan(request)
	if err != nil {
		t.Fatal(err)
	}
	if plan.Status != StatusReady || plan.Targets[0].Files[0].Action != ActionOverride {
		t.Fatalf("plan = %#v", plan)
	}
	if got := fieldPaths(plan.Targets[0].Fields); got != "config.defaultThinkingLevel,config.modelRoles.default" {
		t.Fatalf("fields = %s", got)
	}
	if len(plan.Targets[0].Files) != 2 {
		t.Fatalf("files = %#v", plan.Targets[0].Files)
	}

	if _, err := ApplyPlan(plan, ApplyOptions{ExpectedPlanID: plan.PlanID}); err != nil {
		t.Fatal(err)
	}
	want := "# keep\nmodelRoles:\n  reviewer: other/model\n  default: \"openai/gpt-5.6\" # owned\ndefaultThinkingLevel: \"high\" # owned\nunknown:\n  apiKey: SYNTHETIC\n"
	assertInstallTestFile(t, config, want)
	assertInstallTestFile(t, installfs.BackupPath(config, plan.PlanID), before)
	info, err := os.Stat(config)
	if err != nil || info.Mode().Perm() != 0o600 {
		t.Fatalf("config mode = %v, err = %v", info.Mode().Perm(), err)
	}

	reapply, err := BuildPlan(request)
	if err != nil {
		t.Fatal(err)
	}
	if reapply.Status != StatusNoop || reapply.Targets[0].Files[0].Action != ActionNoop {
		t.Fatalf("reapply = %#v", reapply)
	}
}

func TestOhMyPiInstallBindsDestinationAndRejectsStalePlan(t *testing.T) {
	request, root := ohMyPiTestRequest(t)
	config := request.Targets[0].ConfigPath
	writeInstallTestFile(t, config, "modelRoles:\n  default: old/model\ndefaultThinkingLevel: low\n")
	first, err := BuildPlan(request)
	if err != nil {
		t.Fatal(err)
	}
	other := filepath.Join(root, "other", "config.yml")
	if err := os.MkdirAll(filepath.Dir(other), 0o755); err != nil {
		t.Fatal(err)
	}
	writeInstallTestFile(t, other, "modelRoles:\n  default: old/model\ndefaultThinkingLevel: low\n")
	secondRequest := request
	secondRequest.Targets = []TargetRequest{{Target: request.Targets[0].Target, ConfigPath: other}}
	second, err := BuildPlan(secondRequest)
	if err != nil {
		t.Fatal(err)
	}
	if first.PlanID == second.PlanID || first.Targets[0].DestinationSHA256 == second.Targets[0].DestinationSHA256 {
		t.Fatal("plan was not bound to the explicit destination")
	}

	writeInstallTestFile(t, config, "# third-party edit\nmodelRoles:\n  default: edited/model\n")
	if _, err := ApplyPlan(first, ApplyOptions{ExpectedPlanID: first.PlanID}); !errors.Is(err, installfs.ErrStale) {
		t.Fatalf("stale apply error = %v", err)
	}
	assertInstallTestFile(t, config, "# third-party edit\nmodelRoles:\n  default: edited/model\n")
}

func TestOhMyPiAdapterBlocksUnverifiedRequirements(t *testing.T) {
	mode := "read-only"
	tests := map[string]struct {
		profile profilemango.ResolvedProfile
		route   profilemango.RouteBinding
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
		"resources": {
			profile: profilemango.ResolvedProfile{Instructions: []string{"instructions/system.md"}},
			want:    "delivery remains install-blocking",
		},
		"proxy route": {
			route: profilemango.RouteBinding{Provider: "openai", Model: "gpt-5.6", Transport: "proxy", Authentication: "oauth", Effort: "high"},
			want:  "native transport",
		},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			route := test.route
			if route.Provider == "" {
				route = ohMyPiRoute()
			}
			_, err := (ohMyPiAdapter{}).Plan(AdapterInput{Target: Target{Name: "oh-my-pi", Version: "18.2.6"}, Profile: test.profile, Route: route})
			if err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("error = %v, want substring %q", err, test.want)
			}
		})
	}
}

func TestOhMyPiMetadataRemainsBlockedWithoutNativeApplicability(t *testing.T) {
	metadata := (ohMyPiAdapter{}).Metadata()
	if metadata.Installable || metadata.Status != StatusBlocked {
		t.Fatalf("metadata = %#v, want blocked and not installable", metadata)
	}
	for _, want := range []string{"pi_natives", "source-native settings probe"} {
		if !strings.Contains(metadata.Reason, want) {
			t.Fatalf("reason = %q, want substring %q", metadata.Reason, want)
		}
	}
}

func ohMyPiTestRequest(t *testing.T) (Request, string) {
	t.Helper()
	request, root := testRequest(t, NewRegistry(ohMyPiTestAdapter{}))
	writeInstallTestFile(t, request.BindingsPath, `routes:
  primary:
    provider: openai
    transport: native
    authentication: oauth
    model: gpt-5.6
    effort: high
`)
	request.Override = true
	request.Targets = []TargetRequest{{Target: Target{Name: "oh-my-pi", Version: "18.2.6"}, ConfigPath: filepath.Join(root, "target", "config.yml")}}
	if err := os.MkdirAll(filepath.Dir(request.Targets[0].ConfigPath), 0o755); err != nil {
		t.Fatal(err)
	}
	return request, root
}

type ohMyPiTestAdapter struct{}

func (ohMyPiTestAdapter) Metadata() AdapterMetadata {
	metadata := (ohMyPiAdapter{}).Metadata()
	metadata.Installable = true
	metadata.Status = StatusReady
	metadata.Reason = "synthetic contract test override; production promotion remains blocked"
	return metadata
}

func (ohMyPiTestAdapter) Plan(input AdapterInput) (Patch, error) {
	return (ohMyPiAdapter{}).Plan(input)
}

func ohMyPiRoute() profilemango.RouteBinding {
	return profilemango.RouteBinding{Provider: "openai", Transport: "native", Authentication: "oauth", Model: "gpt-5.6", Effort: "high"}
}

func fieldPaths(fields []FieldChange) string {
	paths := make([]string, 0, len(fields))
	for _, field := range fields {
		paths = append(paths, field.Path)
	}
	sort.Strings(paths)
	return strings.Join(paths, ",")
}
