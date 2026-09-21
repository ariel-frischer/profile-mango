package codex

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gitlab.com/ariel-frischer/profile-mango/pkg/profilemango"
)

func TestRenderTable(t *testing.T) {
	t.Parallel()
	mode := "read-only"
	network := "allow"
	shell := "deny"
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
				"codex.route.authentication_unverified",
				"codex.delivery.unverified",
			},
			paths: []string{
				"preview/route-only.config.toml.preview",
				"resources/instructions/system.md",
			},
		},
		"constrained": {
			profile: profilemango.ResolvedProfile{
				Metadata: profilemango.Metadata{Name: "read-only"},
				Permissions: &profilemango.PermissionPolicy{
					Mode: &mode, Network: &network, Shell: &shell,
				},
				Tools:        &profilemango.ResolvedRules{Managed: true, Closed: true, Allow: []string{"read"}},
				Instructions: []string{"instructions/research.md"},
				Skills:       []string{"skills/research/SKILL.md"},
			},
			resources: []Resource{
				ResourceFromContent("instructions/research.md", "instruction", []byte("research\n")),
				ResourceFromContent("skills/research/SKILL.md", "skill", []byte("skill\n")),
			},
			codes: []string{
				"codex.route.authentication_unverified",
				"codex.delivery.unverified",
				"codex.security.permissions_unverified",
				"codex.security.tools_unverified",
			},
			paths: []string{
				"preview/read-only.config.toml.preview",
				"resources/instructions/research.md",
				"resources/skills/research/SKILL.md",
			},
		},
		"unsupported-version": {
			profile: profilemango.ResolvedProfile{Metadata: profilemango.Metadata{Name: "unsupported"}},
			codes:   []string{"codex.version.unsupported", "codex.version.evidence_mismatch"},
			paths:   []string{"preview/unsupported.config.toml.preview"},
		},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			target := DefaultTarget()
			if name == "unsupported-version" {
				target.Version = "0.155.0"
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
	profile := profilemango.ResolvedProfile{
		Metadata:     profilemango.Metadata{Name: "route-only"},
		Instructions: []string{"instructions/system.md"},
	}
	input := Input{
		Profile:   profile,
		Route:     testRoute(),
		Target:    DefaultTarget(),
		Resources: []Resource{ResourceFromContent("instructions/system.md", "instruction", []byte("system\n"))},
	}
	first := Render(input)
	second := Render(input)
	golden, err := os.ReadFile(filepath.Join("testdata", "golden", "route-only.config.toml.preview"))
	if err != nil {
		t.Fatal(err)
	}
	if got := artifactByPath(first.Artifacts, "preview/route-only.config.toml.preview").Content; string(got) != string(golden) {
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
	if string(firstReport) != string(secondReport) {
		t.Fatal("render report is not deterministic")
	}
	if strings.Contains(string(golden), "oauth") || strings.Contains(string(golden), "credential") {
		t.Fatal("candidate config leaked authentication details")
	}
	if !strings.Contains(string(firstReport), EvidenceSHA256) {
		t.Fatal("report omitted exact evidence hash")
	}
}

func TestRenderRejectsEvidenceHash(t *testing.T) {
	target := DefaultTarget()
	target.EvidenceSHA256 = "bad"
	result := Render(Input{Profile: testProfile(), Route: testRoute(), Target: target})
	if result.Applicable || !hasCode(result.Diagnostics, "codex.version.evidence_mismatch") {
		t.Fatalf("evidence mismatch was not blocking: %#v", result)
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
		if artifact.Path == "config.toml" || artifact.Path == "AGENTS.md" || strings.HasPrefix(artifact.Path, "skills/") {
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
