package pi

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ariel-frischer/profile-mango/pkg/profilemango"
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
				"pi.config.acceptance_unverified",
				"pi.config.effective_state_unverified",
				"pi.config.inspector_unsafe",
				"pi.config.precedence_unverified",
				"pi.delivery.unverified",
				"pi.extensions.discovery_unverified",
				"pi.route.authentication_unverified",
				"pi.runtime.enforcement_unverified",
			},
			paths: []string{
				"preview/route-only.settings.json.preview",
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
				"pi.permissions.mode_unverified",
				"pi.permissions.network_unverified",
				"pi.permissions.shell_unverified",
				"pi.security.permissions_unverified",
				"pi.security.tools_unverified",
				"pi.skills.delivery_unverified",
				"pi.tools.allowlist_unverified",
			},
			paths: []string{
				"preview/read-only.settings.json.preview",
				"resources/instructions/research.md",
				"resources/skills/research/SKILL.md",
			},
		},
		"unsupported-version": {
			profile: profilemango.ResolvedProfile{Metadata: profilemango.Metadata{Name: "unsupported"}},
			codes:   []string{"pi.version.unsupported", "pi.version.evidence_mismatch"},
			paths:   []string{"preview/unsupported.settings.json.preview"},
		},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			target := DefaultTarget()
			if name == "unsupported-version" {
				target.Version, target.EvidenceSHA256 = "0.86.2", "bad"
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
		Profile:   profilemango.ResolvedProfile{Metadata: profilemango.Metadata{Name: "route-only"}, Instructions: []string{"instructions/system.md"}},
		Route:     testRoute(),
		Target:    DefaultTarget(),
		Resources: []Resource{ResourceFromContent("instructions/system.md", "instruction", []byte("system\n"))},
	}
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
	if strings.Contains(string(golden), "authentication") || strings.Contains(string(golden), "oauth") || strings.Contains(string(golden), "secret") {
		t.Fatal("candidate config leaked authentication data")
	}
}

func TestRenderCandidateContainsOnlyExactPiSettingsKeys(t *testing.T) {
	artifact := artifactByPath(Render(Input{Profile: testProfile(), Route: testRoute(), Target: DefaultTarget()}).Artifacts, "preview/route-only.settings.json.preview")
	var settings map[string]any
	if err := json.Unmarshal(artifact.Content, &settings); err != nil {
		t.Fatal(err)
	}
	for key := range settings {
		if key != "defaultProvider" && key != "defaultModel" && key != "defaultThinkingLevel" {
			t.Fatalf("candidate emitted unsupported key %q", key)
		}
	}
	if len(settings) != 3 || settings["defaultProvider"] != "openai" || settings["defaultModel"] != "gpt-5.6" || settings["defaultThinkingLevel"] != "high" {
		t.Fatalf("unexpected candidate settings: %#v", settings)
	}
}

func TestRenderRejectsEvidenceAndVersion(t *testing.T) {
	target := DefaultTarget()
	target.Version, target.EvidenceSHA256 = "0.86.2", "bad"
	result := Render(Input{Profile: testProfile(), Route: testRoute(), Target: target})
	if result.Applicable || !hasCode(result.Diagnostics, "pi.target.version_unsupported") || !hasCode(result.Diagnostics, "pi.version.evidence_mismatch") {
		t.Fatalf("version/hash mismatch was not blocking: %#v", result)
	}
}

func TestRenderRejectsWrongTargetName(t *testing.T) {
	target := DefaultTarget()
	target.Name = "oh-my-pi"
	result := Render(Input{Profile: testProfile(), Route: testRoute(), Target: target})
	if result.Applicable || !hasCode(result.Diagnostics, "pi.target.unsupported") {
		t.Fatalf("wrong target name was not blocking: %#v", result)
	}
}

func TestRenderRejectsUnsupportedThinkingAndTransport(t *testing.T) {
	route := testRoute()
	route.Effort = "turbo"
	result := Render(Input{Profile: testProfile(), Route: route, Target: DefaultTarget()})
	if result.Applicable || !hasCode(result.Diagnostics, "pi.route.effort_unsupported") || artifactByPath(result.Artifacts, "preview/route-only.settings.json.preview").Path != "" {
		t.Fatalf("unsupported thinking level was not fail-closed: %#v", result)
	}

	route = testRoute()
	route.Transport = "proxy"
	result = Render(Input{Profile: testProfile(), Route: route, Target: DefaultTarget()})
	if result.Applicable || !hasCode(result.Diagnostics, "pi.route.transport_unsupported") || artifactByPath(result.Artifacts, "preview/route-only.settings.json.preview").Path != "" {
		t.Fatalf("unsupported transport was not fail-closed: %#v", result)
	}
}

func TestRenderRejectsIncompleteRoute(t *testing.T) {
	route := testRoute()
	route.Authentication = ""
	result := Render(Input{Profile: testProfile(), Route: route, Target: DefaultTarget()})
	if result.Applicable || !hasCode(result.Diagnostics, "pi.route.incomplete") || artifactByPath(result.Artifacts, "preview/route-only.settings.json.preview").Path != "" {
		t.Fatalf("incomplete route was not fail-closed: %#v", result)
	}
}

func TestRenderRejectsResourceDigestAndPath(t *testing.T) {
	profile := profilemango.ResolvedProfile{Metadata: profilemango.Metadata{Name: "route-only"}, Instructions: []string{"instructions/system.md"}}
	badDigest := ResourceFromContent("instructions/system.md", "instruction", []byte("system\n"))
	badDigest.Digest.SHA256 = "bad"
	result := Render(Input{Profile: profile, Route: testRoute(), Target: DefaultTarget(), Resources: []Resource{badDigest}})
	if !hasCode(result.Diagnostics, "pi.resource.digest_mismatch") || len(result.Artifacts) != 0 {
		t.Fatalf("digest mismatch was not fail-closed: %#v", result)
	}

	badPath := ResourceFromContent("../secret", "instruction", []byte("secret\n"))
	result = Render(Input{Profile: profile, Route: testRoute(), Target: DefaultTarget(), Resources: []Resource{badPath}})
	if !hasCode(result.Diagnostics, "pi.resource.path_invalid") || len(result.Artifacts) != 0 {
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
	artifact := artifactByPath(result.Artifacts, "preview/route-only.settings.json.preview")
	if strings.Contains(string(artifact.Content), "synthetic-secret-sentinel") || strings.Contains(string(artifact.Content), "authentication") {
		t.Fatal("candidate config copied authentication details")
	}
}

func TestRenderEscapesSettingsJSON(t *testing.T) {
	route := testRoute()
	route.Provider = "provider-\"newline\n"
	route.Model = "model-\"newline\n"
	result := Render(Input{Profile: testProfile(), Route: route, Target: DefaultTarget()})
	artifact := artifactByPath(result.Artifacts, "preview/route-only.settings.json.preview")
	if artifact.Path == "" || !json.Valid(artifact.Content) || strings.Contains(string(artifact.Content), "synthetic-secret") {
		t.Fatalf("candidate was not valid JSON or leaked a secret: %s", artifact.Content)
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
		if artifact.Path == "settings.json" || artifact.Path == "AGENTS.md" || artifact.Path == "plan.json" || artifact.Path == "manifest.json" || strings.HasPrefix(artifact.Path, "skills/") {
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
