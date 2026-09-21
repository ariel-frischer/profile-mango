package opencode

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gitlab.com/ariel-frischer/profile-mango/pkg/profilemango"
)

func TestRenderTable(t *testing.T) {
	t.Parallel()
	mode, network, shell := "read-only", "allow", "deny"
	tests := map[string]struct {
		profile   profilemango.ResolvedProfile
		resources []Resource
		codes     []string
		paths     []string
	}{
		"route-only": {
			profile: profilemango.ResolvedProfile{
				Metadata:     profilemango.Metadata{Name: "route-only"},
				Instructions: []string{"instructions/system.md"},
			},
			resources: []Resource{ResourceFromContent("instructions/system.md", "instruction", []byte("system\n"))},
			codes: []string{
				"opencode.config.effective_state_partial",
				"opencode.config.inspector_writeful",
				"opencode.config.precedence_partial",
				"opencode.config.unknown_keys_ignored",
				"opencode.extensions.plugins_mcp_unverified",
				"opencode.install.validation_deferred",
				"opencode.instructions.delivery_unverified",
				"opencode.route.authentication_unverified",
				"opencode.route.effort_unverified",
				"opencode.route.transport_unverified",
				"opencode.runtime.enforcement_unverified",
			},
			paths: []string{
				"preview/route-only.opencode.jsonc.preview",
				"resources/instructions/system.md",
			},
		},
		"constrained": {
			profile: profilemango.ResolvedProfile{
				Metadata:     profilemango.Metadata{Name: "read-only"},
				Permissions:  &profilemango.PermissionPolicy{Mode: &mode, Network: &network, Shell: &shell},
				Tools:        &profilemango.ResolvedRules{Managed: true, Closed: true, Allow: []string{"read"}},
				Instructions: []string{"instructions/research.md"},
				Skills:       []string{"skills/research/SKILL.md"},
			},
			resources: []Resource{
				ResourceFromContent("instructions/research.md", "instruction", []byte("research\n")),
				ResourceFromContent("skills/research/SKILL.md", "skill", []byte("skill\n")),
			},
			codes: []string{
				"opencode.permissions.mode_unverified",
				"opencode.permissions.network_unverified",
				"opencode.permissions.shell_unverified",
				"opencode.security.permissions_unverified",
				"opencode.security.tools_unverified",
				"opencode.skills.delivery_unverified",
				"opencode.tools.allowlist_unverified",
			},
			paths: []string{
				"preview/read-only.opencode.jsonc.preview",
				"resources/instructions/research.md",
				"resources/skills/research/SKILL.md",
			},
		},
		"unsupported-version": {
			profile: profilemango.ResolvedProfile{Metadata: profilemango.Metadata{Name: "unsupported"}},
			codes:   []string{"opencode.target.version_unsupported", "opencode.version.unsupported", "opencode.version.evidence_mismatch"},
			paths:   []string{"preview/unsupported.opencode.jsonc.preview"},
		},
	}
	for name, test := range tests {
		name, test := name, test
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			target := DefaultTarget()
			if name == "unsupported-version" {
				target.Version = "1.18.32"
				target.EvidenceSHA256 = "bad"
			}
			result := Render(Input{Profile: test.profile, Route: testRoute(), Target: target, Resources: test.resources})
			if result.Applicable {
				t.Fatal("blocked result was marked applicable")
			}
			for _, code := range test.codes {
				if !hasCode(result.Diagnostics, code) {
					t.Fatalf("missing diagnostic %s: %#v", code, result.Diagnostics)
				}
			}
			for _, path := range test.paths {
				if artifactByPath(result.Artifacts, path).Path == "" {
					t.Fatalf("missing artifact %s: %#v", path, result.Artifacts)
				}
			}
			assertNoActiveArtifacts(t, result.Artifacts)
		})
	}
}

func TestRenderRouteGoldenAndStableReport(t *testing.T) {
	input := Input{
		Profile: profilemango.ResolvedProfile{
			Metadata:     profilemango.Metadata{Name: "route-only"},
			Instructions: []string{"instructions/system.md"},
		},
		Route:     testRoute(),
		Target:    DefaultTarget(),
		Resources: []Resource{ResourceFromContent("instructions/system.md", "instruction", []byte("system\n"))},
	}
	first, second := Render(input), Render(input)
	golden, err := os.ReadFile(filepath.Join("testdata", "golden", "route-only.opencode.jsonc.preview"))
	if err != nil {
		t.Fatal(err)
	}
	artifact := artifactByPath(first.Artifacts, "preview/route-only.opencode.jsonc.preview")
	if string(artifact.Content) != string(golden) {
		t.Fatalf("config golden mismatch\n%s", artifact.Content)
	}
	firstReport, err := first.ReportJSON(true)
	if err != nil {
		t.Fatal(err)
	}
	secondReport, err := second.ReportJSON(true)
	if err != nil {
		t.Fatal(err)
	}
	if string(firstReport) != string(secondReport) {
		t.Fatal("render report is not deterministic")
	}
	if !strings.Contains(string(firstReport), EvidenceSHA256) || !strings.Contains(string(firstReport), AdapterVersion) ||
		!strings.Contains(string(firstReport), EvidenceLevel) || !strings.Contains(string(firstReport), "opencode.install.validation_deferred") {
		t.Fatal("report omitted exact evidence, adapter metadata, native validation level, or install boundary")
	}
	if strings.Contains(string(golden), "oauth") || strings.Contains(string(golden), "credential") {
		t.Fatal("candidate config leaked authentication details")
	}
}

func TestRenderCandidateMapsOnlyProviderModel(t *testing.T) {
	artifact := artifactByPath(Render(Input{Profile: testProfile(), Route: testRoute(), Target: DefaultTarget()}).Artifacts, "preview/route-only.opencode.jsonc.preview")
	content := string(artifact.Content)
	if artifact.Path == "" || !strings.Contains(content, `"model": "openai/gpt-5.6",`) {
		t.Fatalf("candidate omitted provider/model route: %s", content)
	}
	for _, forbidden := range []string{"provider:", "effort:", "authentication:", "permissions:", "tools:", "options:", "auth.json"} {
		if strings.Contains(content, forbidden) {
			t.Fatalf("candidate emitted unsupported key %q: %s", forbidden, content)
		}
	}
}

func TestRenderReportsNativeValidationCapabilities(t *testing.T) {
	result := Render(Input{Profile: testProfile(), Route: testRoute(), Target: DefaultTarget()})
	want := map[string]string{
		"config.acceptance":      StatusSupported,
		"config.inspection":      StatusPartial,
		"config.effective-state": StatusPartial,
		"config.precedence":      StatusPartial,
		"install":                StatusBlocking,
	}
	for field, status := range want {
		if actual, ok := capabilityStatus(result, field); !ok || actual != status {
			t.Errorf("capability %q = %q, %v; want %q, true", field, actual, ok, status)
		}
	}
}

func TestRenderRejectsEvidenceAndVersion(t *testing.T) {
	target := DefaultTarget()
	target.Version, target.EvidenceSHA256 = "1.18.32", "bad"
	result := Render(Input{Profile: testProfile(), Route: testRoute(), Target: target})
	if result.Applicable || !hasCode(result.Diagnostics, "opencode.target.version_unsupported") || !hasCode(result.Diagnostics, "opencode.version.evidence_mismatch") {
		t.Fatalf("version/hash mismatch was not blocking: %#v", result)
	}
}

func TestRenderRejectsWrongTargetName(t *testing.T) {
	target := DefaultTarget()
	target.Name = "openclaw"
	result := Render(Input{Profile: testProfile(), Route: testRoute(), Target: target})
	if result.Applicable || !hasCode(result.Diagnostics, "opencode.target.unsupported") {
		t.Fatalf("wrong target name was not blocking: %#v", result)
	}
}

func TestRenderRejectsUnsupportedRoute(t *testing.T) {
	tests := map[string]struct {
		mutate func(*profilemango.RouteBinding)
		code   string
	}{
		"transport": {
			mutate: func(route *profilemango.RouteBinding) { route.Transport = "proxy" },
			code:   "opencode.route.transport_unsupported",
		},
		"effort": {
			mutate: func(route *profilemango.RouteBinding) { route.Effort = "turbo" },
			code:   "opencode.route.effort_unsupported",
		},
		"incomplete": {
			mutate: func(route *profilemango.RouteBinding) { route.Model = "" },
			code:   "opencode.route.incomplete",
		},
	}
	for name, test := range tests {
		name, test := name, test
		t.Run(name, func(t *testing.T) {
			route := testRoute()
			test.mutate(&route)
			result := Render(Input{Profile: testProfile(), Route: route, Target: DefaultTarget()})
			if result.Applicable || !hasCode(result.Diagnostics, test.code) || artifactByPath(result.Artifacts, "preview/route-only.opencode.jsonc.preview").Path != "" {
				t.Fatalf("unsupported route was not fail-closed: %#v", result)
			}
		})
	}
}

func TestRenderRejectsResourceDigestAndPath(t *testing.T) {
	profile := profilemango.ResolvedProfile{Metadata: profilemango.Metadata{Name: "route-only"}, Instructions: []string{"instructions/system.md"}}
	badDigest := ResourceFromContent("instructions/system.md", "instruction", []byte("system\n"))
	badDigest.Digest.SHA256 = "bad"
	result := Render(Input{Profile: profile, Route: testRoute(), Target: DefaultTarget(), Resources: []Resource{badDigest}})
	if !hasCode(result.Diagnostics, "opencode.resource.digest_mismatch") || len(result.Artifacts) != 0 {
		t.Fatalf("digest mismatch was not fail-closed: %#v", result)
	}

	badPath := ResourceFromContent("../secret", "instruction", []byte("secret\n"))
	result = Render(Input{Profile: profile, Route: testRoute(), Target: DefaultTarget(), Resources: []Resource{badPath}})
	if !hasCode(result.Diagnostics, "opencode.resource.path_invalid") || len(result.Artifacts) != 0 {
		t.Fatalf("escaping resource was not fail-closed: %#v", result)
	}
}

func TestRenderDoesNotCopyAuthenticationSentinel(t *testing.T) {
	result := Render(Input{
		Profile: testProfile(),
		Route: profilemango.RouteBinding{
			Provider: "openai", Transport: "native", Authentication: "synthetic-secret-sentinel", Model: "gpt-5.6", Effort: "high",
		},
		Target: DefaultTarget(),
	})
	artifact := artifactByPath(result.Artifacts, "preview/route-only.opencode.jsonc.preview")
	report, err := result.ReportJSON(true)
	if err != nil {
		t.Fatal(err)
	}
	for _, output := range []string{string(artifact.Content), string(report)} {
		if strings.Contains(output, "synthetic-secret-sentinel") || strings.Contains(output, "provider-options") || strings.Contains(output, "apiKey") {
			t.Fatalf("output copied authentication or provider data: %s", output)
		}
	}
}

func TestRenderEscapesJSONCString(t *testing.T) {
	route := testRoute()
	route.Provider = "provider-\"newline"
	route.Model = "model-\n"
	result := Render(Input{Profile: testProfile(), Route: route, Target: DefaultTarget()})
	artifact := artifactByPath(result.Artifacts, "preview/route-only.opencode.jsonc.preview")
	if artifact.Path == "" || !strings.Contains(string(artifact.Content), `provider-\"newline/model-\n`) {
		t.Fatalf("candidate did not escape JSONC string: %s", artifact.Content)
	}
}

func testProfile() profilemango.ResolvedProfile {
	return profilemango.ResolvedProfile{Metadata: profilemango.Metadata{Name: "route-only"}}
}

func testRoute() profilemango.RouteBinding {
	return profilemango.RouteBinding{Provider: "openai", Transport: "native", Authentication: "oauth", Model: "gpt-5.6", Effort: "high"}
}

func artifactByPath(artifacts []Artifact, wanted string) Artifact {
	for _, artifact := range artifacts {
		if artifact.Path == wanted {
			return artifact
		}
	}
	return Artifact{}
}

func assertNoActiveArtifacts(t *testing.T, artifacts []Artifact) {
	t.Helper()
	for _, artifact := range artifacts {
		if artifact.Path == "opencode.json" || artifact.Path == "opencode.jsonc" || artifact.Path == "AGENTS.md" || artifact.Path == "plan.json" || artifact.Path == "manifest.json" || artifact.Path == "auth.json" || strings.HasPrefix(artifact.Path, "plugins/") || strings.HasPrefix(artifact.Path, "skills/") {
			t.Fatalf("active artifact emitted: %s", artifact.Path)
		}
	}
}

func hasCode(diagnostics profilemango.Diagnostics, code string) bool {
	for _, diagnostic := range diagnostics {
		if diagnostic.Code == code {
			return true
		}
	}
	return false
}

func capabilityStatus(result Result, field string) (string, bool) {
	for _, capability := range result.Capabilities {
		if capability.Field == field {
			return capability.Status, true
		}
	}
	return "", false
}
