package hermes

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
				Metadata: profilemango.Metadata{Name: "route-only"}, Instructions: []string{"instructions/system.md"},
			},
			resources: []Resource{ResourceFromContent("instructions/system.md", "instruction", []byte("system\n"))},
			codes: []string{
				"hermes.config.acceptance_unverified", "hermes.config.effective_state_unverified",
				"hermes.config.inspector_unsafe", "hermes.config.precedence_unverified",
				"hermes.context.discovery_unverified", "hermes.instructions.delivery_unverified",
				"hermes.route.authentication_unverified", "hermes.runtime.enforcement_unverified",
			},
			paths: []string{"preview/route-only.config.yaml.preview", "resources/instructions/system.md"},
		},
		"constrained": {
			profile: profilemango.ResolvedProfile{
				Metadata:     profilemango.Metadata{Name: "read-only"},
				Permissions:  &profilemango.PermissionPolicy{Mode: &mode, Network: &network, Shell: &shell},
				Tools:        &profilemango.ResolvedRules{Managed: true, Closed: true, Allow: []string{"read"}},
				Instructions: []string{"instructions/research.md"}, Skills: []string{"skills/research/SKILL.md"},
			},
			resources: []Resource{
				ResourceFromContent("instructions/research.md", "instruction", []byte("research\n")),
				ResourceFromContent("skills/research/SKILL.md", "skill", []byte("skill\n")),
			},
			codes: []string{
				"hermes.permissions.mode_unverified", "hermes.permissions.network_unverified", "hermes.permissions.shell_unverified",
				"hermes.security.permissions_unverified", "hermes.security.tools_unverified", "hermes.skills.delivery_unverified",
				"hermes.tools.allowlist_unverified",
			},
			paths: []string{"preview/read-only.config.yaml.preview", "resources/instructions/research.md", "resources/skills/research/SKILL.md"},
		},
		"unsupported-version": {
			profile: profilemango.ResolvedProfile{Metadata: profilemango.Metadata{Name: "unsupported"}},
			codes:   []string{"hermes.version.unsupported", "hermes.version.evidence_mismatch"},
			paths:   []string{"preview/unsupported.config.yaml.preview"},
		},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			target := DefaultTarget()
			if name == "unsupported-version" {
				target.Version, target.EvidenceSHA256 = "0.21.4", "bad"
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
		Profile: profilemango.ResolvedProfile{Metadata: profilemango.Metadata{Name: "route-only"}, Instructions: []string{"instructions/system.md"}},
		Route:   testRoute(), Target: DefaultTarget(), Resources: []Resource{ResourceFromContent("instructions/system.md", "instruction", []byte("system\n"))},
	}
	first, second := Render(input), Render(input)
	golden, err := os.ReadFile(filepath.Join("testdata", "golden", "route-only.config.yaml.preview"))
	if err != nil {
		t.Fatal(err)
	}
	if got := artifactByPath(first.Artifacts, "preview/route-only.config.yaml.preview").Content; string(got) != string(golden) {
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
		t.Fatal("Hermes report is not deterministic or omitted exact evidence")
	}
}

func TestRenderRejectsEvidenceAndEffort(t *testing.T) {
	target := DefaultTarget()
	target.EvidenceSHA256 = "bad"
	result := Render(Input{Profile: testProfile(), Route: testRoute(), Target: target})
	if result.Applicable || !hasCode(result.Diagnostics, "hermes.version.evidence_mismatch") {
		t.Fatalf("evidence mismatch was not blocking: %#v", result)
	}
	route := testRoute()
	route.Effort = "turbo"
	result = Render(Input{Profile: testProfile(), Route: route, Target: DefaultTarget()})
	if result.Applicable || !hasCode(result.Diagnostics, "hermes.route.effort_unsupported") {
		t.Fatalf("unsupported effort was not blocking: %#v", result)
	}
	if artifactByPath(result.Artifacts, "preview/route-only.config.yaml.preview").Path != "" {
		t.Fatal("unsupported effort emitted a candidate config")
	}
}

func TestRenderRejectsResourceDigestAndAuthenticationSentinel(t *testing.T) {
	resource := ResourceFromContent("instructions/system.md", "instruction", []byte("system\n"))
	resource.Content = []byte("changed\n")
	result := Render(Input{
		Profile: profilemango.ResolvedProfile{Metadata: profilemango.Metadata{Name: "route-only"}, Instructions: []string{"instructions/system.md"}},
		Route:   testRoute(), Target: DefaultTarget(), Resources: []Resource{resource},
	})
	if result.Applicable || !hasCode(result.Diagnostics, "hermes.resource.digest_mismatch") || len(result.Artifacts) != 0 {
		t.Fatalf("resource mismatch was not fail-closed: %#v", result)
	}
	result = Render(Input{
		Profile: testProfile(),
		Route:   profilemango.RouteBinding{Provider: "openai", Transport: "native", Authentication: "synthetic-secret-sentinel", Model: "gpt-5.6", Effort: "high"},
		Target:  DefaultTarget(),
	})
	artifact := artifactByPath(result.Artifacts, "preview/route-only.config.yaml.preview")
	if strings.Contains(string(artifact.Content), "synthetic-secret-sentinel") || strings.Contains(string(artifact.Content), "credential") {
		t.Fatal("candidate config copied authentication details")
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
		if artifact.Path == "config.yaml" || artifact.Path == "AGENTS.md" || artifact.Path == "plan.json" || artifact.Path == "manifest.json" || strings.HasPrefix(artifact.Path, "skills/") {
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
