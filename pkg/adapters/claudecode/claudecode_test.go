package claudecode

import (
	"encoding/json"
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
			profile:   profilemango.ResolvedProfile{Metadata: profilemango.Metadata{Name: "route-only"}, Instructions: []string{"instructions/system.md"}},
			resources: []Resource{ResourceFromContent("instructions/system.md", "instruction", []byte("system\n"))},
			codes:     []string{"claudecode.config.acceptance_unverified", "claudecode.config.effective_state_unverified", "claudecode.config.inspector_unsafe", "claudecode.config.precedence_unverified", "claudecode.extensions.plugins_hooks_mcp_unverified", "claudecode.instructions.delivery_unverified", "claudecode.route.authentication_unverified", "claudecode.runtime.enforcement_unverified"},
			paths:     []string{"preview/route-only.settings.json.preview", "resources/instructions/system.md"},
		},
		"constrained": {
			profile:   profilemango.ResolvedProfile{Metadata: profilemango.Metadata{Name: "read-only"}, Permissions: &profilemango.PermissionPolicy{Mode: &mode, Network: &network, Shell: &shell}, Tools: &profilemango.ResolvedRules{Managed: true, Closed: true, Allow: []string{"read"}}, Instructions: []string{"instructions/research.md"}, Skills: []string{"skills/research/SKILL.md"}},
			resources: []Resource{ResourceFromContent("instructions/research.md", "instruction", []byte("research\n")), ResourceFromContent("skills/research/SKILL.md", "skill", []byte("skill\n"))},
			codes:     []string{"claudecode.permissions.mode_unverified", "claudecode.permissions.network_unverified", "claudecode.permissions.shell_unverified", "claudecode.security.permissions_unverified", "claudecode.security.tools_unverified", "claudecode.skills.delivery_unverified", "claudecode.tools.allowlist_unverified"},
			paths:     []string{"preview/read-only.settings.json.preview", "resources/instructions/research.md", "resources/skills/research/SKILL.md"},
		},
		"unsupported-version": {
			profile: profilemango.ResolvedProfile{Metadata: profilemango.Metadata{Name: "unsupported"}},
			codes:   []string{"claudecode.version.unsupported", "claudecode.version.evidence_mismatch"},
			paths:   []string{"preview/unsupported.settings.json.preview"},
		},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			target := DefaultTarget()
			if name == "unsupported-version" {
				target.Version, target.EvidenceSHA256 = "2.1.279", "bad"
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
	input := Input{Profile: profilemango.ResolvedProfile{Metadata: profilemango.Metadata{Name: "route-only"}, Instructions: []string{"instructions/system.md"}}, Route: testRoute(), Target: DefaultTarget(), Resources: []Resource{ResourceFromContent("instructions/system.md", "instruction", []byte("system\n"))}}
	first, second := Render(input), Render(input)
	golden, err := os.ReadFile(filepath.Join("testdata", "golden", "route-only.settings.json.preview"))
	if err != nil {
		t.Fatal(err)
	}
	if got := artifactByPath(first.Artifacts, "preview/route-only.settings.json.preview").Content; string(got) != string(golden) {
		t.Fatalf("config golden mismatch\n%s", got)
	}
	firstReport, err := first.ReportJSON(true)
	if err != nil {
		t.Fatal(err)
	}
	secondReport, err := second.ReportJSON(true)
	if err != nil {
		t.Fatal(err)
	}
	if string(firstReport) != string(secondReport) || !strings.Contains(string(firstReport), EvidenceSHA256) {
		t.Fatal("render report is not deterministic or omitted exact evidence")
	}
	if strings.Contains(string(golden), "oauth") || strings.Contains(string(golden), "secret") || strings.Contains(string(golden), "openai") {
		t.Fatal("candidate config leaked route or authentication details")
	}
}

func TestRenderRejectsResourceDigestAndPath(t *testing.T) {
	profile := profilemango.ResolvedProfile{Metadata: profilemango.Metadata{Name: "route-only"}, Instructions: []string{"instructions/system.md"}}
	badDigest := ResourceFromContent("instructions/system.md", "instruction", []byte("system\n"))
	badDigest.Digest.SHA256 = "bad"
	result := Render(Input{Profile: profile, Route: testRoute(), Target: DefaultTarget(), Resources: []Resource{badDigest}})
	if !hasCode(result.Diagnostics, "claudecode.resource.digest_mismatch") || len(result.Artifacts) != 0 {
		t.Fatalf("digest mismatch was not fail-closed: %#v", result)
	}
	badPath := ResourceFromContent("../secret", "instruction", []byte("secret\n"))
	result = Render(Input{Profile: profile, Route: testRoute(), Target: DefaultTarget(), Resources: []Resource{badPath}})
	if !hasCode(result.Diagnostics, "claudecode.resource.path_invalid") || len(result.Artifacts) != 0 {
		t.Fatalf("escaping resource was not fail-closed: %#v", result)
	}
}

func TestRenderEscapesModelAsJSONWithoutSecrets(t *testing.T) {
	model := "model-\"newline\n"
	result := Render(Input{Profile: profilemango.ResolvedProfile{Metadata: profilemango.Metadata{Name: "quoted"}}, Route: profilemango.RouteBinding{Provider: "secret-provider", Transport: "native", Authentication: "oauth-secret", Model: model, Effort: "high"}, Target: DefaultTarget()})
	artifact := artifactByPath(result.Artifacts, "preview/quoted.settings.json.preview")
	if artifact.Path == "" || !json.Valid(artifact.Content) || strings.Contains(string(artifact.Content), "secret-provider") || strings.Contains(string(artifact.Content), "oauth-secret") {
		t.Fatalf("candidate was not valid or leaked route data: %s", artifact.Content)
	}
}

func TestRenderRejectsEvidenceAndEffort(t *testing.T) {
	target := DefaultTarget()
	target.EvidenceSHA256 = "bad"
	result := Render(Input{Profile: testProfile(), Route: testRoute(), Target: target})
	if result.Applicable || !hasCode(result.Diagnostics, "claudecode.version.evidence_mismatch") {
		t.Fatalf("evidence mismatch was not blocking: %#v", result)
	}

	route := testRoute()
	route.Effort = "turbo"
	result = Render(Input{Profile: testProfile(), Route: route, Target: DefaultTarget()})
	if result.Applicable || !hasCode(result.Diagnostics, "claudecode.route.effort_unsupported") {
		t.Fatalf("unsupported effort was not blocking: %#v", result)
	}
	if artifactByPath(result.Artifacts, "preview/route-only.settings.json.preview").Path != "" {
		t.Fatal("unsupported effort emitted a candidate config")
	}
}

func TestRenderRejectsMalformedProfileAndIncompleteRoute(t *testing.T) {
	malformed := testProfile()
	malformed.Metadata.Name = "../escape"
	result := Render(Input{Profile: malformed, Route: testRoute(), Target: DefaultTarget()})
	if len(result.Artifacts) != 0 {
		t.Fatal("malformed profile emitted an artifact")
	}

	route := testRoute()
	route.Model = ""
	result = Render(Input{Profile: testProfile(), Route: route, Target: DefaultTarget()})
	if !hasCode(result.Diagnostics, "claudecode.route.incomplete") || len(result.Artifacts) != 0 {
		t.Fatalf("incomplete route was not fail-closed: %#v", result)
	}
}

func TestRenderReportsSeparateClaudeBoundaries(t *testing.T) {
	result := Render(Input{Profile: testProfile(), Route: testRoute(), Target: DefaultTarget()})
	want := map[string]string{
		"target.artifact":              StatusSupported,
		"target.version":               StatusSupported,
		"config.fidelity":              StatusPartial,
		"config.acceptance":            StatusBlocking,
		"config.effective-state":       StatusBlocking,
		"config.precedence":            StatusBlocking,
		"route.model":                  StatusPartial,
		"route.effort":                 StatusBlocking,
		"route.provider":               StatusBlocking,
		"route.authentication":         StatusBlocking,
		"route.transport":              StatusBlocking,
		"extensions.plugins-hooks-mcp": StatusBlocking,
		"runtime.enforcement":          StatusBlocking,
	}
	for field, status := range want {
		if got, ok := capabilityStatus(result, field); !ok || got != status {
			t.Fatalf("capability %s = %q, %t, want %q", field, got, ok, status)
		}
	}
}

func testProfile() profilemango.ResolvedProfile {
	return profilemango.ResolvedProfile{Metadata: profilemango.Metadata{Name: "route-only"}}
}

func testRoute() profilemango.RouteBinding {
	return profilemango.RouteBinding{Provider: "openai", Transport: "native", Authentication: "oauth", Model: "gpt-5.6", Effort: "high"}
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

func artifactByPath(artifacts []Artifact, path string) Artifact {
	for _, artifact := range artifacts {
		if artifact.Path == path {
			return artifact
		}
	}
	return Artifact{}
}

func assertNoActiveArtifacts(t *testing.T, artifacts []Artifact) {
	t.Helper()
	for _, artifact := range artifacts {
		if strings.Contains(artifact.Path, "config.json") || strings.HasSuffix(artifact.Path, ".json") || artifact.Path == "CLAUDE.md" || strings.HasPrefix(artifact.Path, ".claude/") {
			t.Fatalf("active artifact path emitted: %s", artifact.Path)
		}
	}
}
