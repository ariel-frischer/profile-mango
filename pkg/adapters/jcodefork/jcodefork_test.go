package jcodefork

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ariel-frischer/profile-mango/pkg/profilemango"
)

func TestRenderTable(t *testing.T) {
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
				"jcodefork.config.acceptance_unverified",
				"jcodefork.config.discovery_unverified",
				"jcodefork.config.precedence_unverified",
				"jcodefork.child.overrides_unverified",
				"jcodefork.hooks.extensions.mcp_unverified",
				"jcodefork.experimental_only",
				"jcodefork.instructions.delivery_unverified",
				"jcodefork.route.authentication_unverified",
				"jcodefork.runtime.enforcement_unverified",
			},
			paths: []string{
				"preview/route-only.config.toml.preview",
				"resources/instructions/system.md",
			},
		},
		"constrained": {
			profile: profilemango.ResolvedProfile{
				Metadata:     profilemango.Metadata{Name: "read-only"},
				Permissions:  &profilemango.PermissionPolicy{Mode: &mode, Network: &network, Shell: &shell},
				Tools:        &profilemango.ResolvedRules{Managed: true, Closed: true, Allow: []string{"read"}, Deny: []string{"write"}},
				Instructions: []string{"instructions/research.md"},
				Skills:       []string{"skills/research/SKILL.md"},
			},
			resources: []Resource{
				ResourceFromContent("instructions/research.md", "instruction", []byte("research\n")),
				ResourceFromContent("skills/research/SKILL.md", "skill", []byte("skill\n")),
			},
			codes: []string{
				"jcodefork.permissions.mode_unverified",
				"jcodefork.permissions.network_unverified",
				"jcodefork.permissions.shell_unverified",
				"jcodefork.skills.delivery_unverified",
				"jcodefork.tools.delivery_unverified",
			},
			paths: []string{
				"preview/read-only.config.toml.preview",
				"resources/instructions/research.md",
				"resources/skills/research/SKILL.md",
			},
		},
		"unsupported-version": {
			profile: profilemango.ResolvedProfile{Metadata: profilemango.Metadata{Name: "unsupported"}},
			codes:   []string{"jcodefork.version.unsupported", "jcodefork.version.evidence_mismatch"},
			paths:   []string{"preview/unsupported.config.toml.preview"},
		},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			target := DefaultTarget()
			if name == "unsupported-version" {
				target.Version = "0.83.909-dev (wrong)"
				target.EvidenceSHA256 = "bad"
			}
			result := Render(Input{Profile: test.profile, Route: testRoute(), Target: target, Resources: test.resources})
			if result.Applicable {
				t.Fatal("experimental result was marked applicable")
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
	if string(firstReport) != string(secondReport) || !strings.Contains(string(firstReport), EvidenceSHA256) || !strings.Contains(string(firstReport), ExperimentalLabel) {
		t.Fatal("report is not deterministic or omitted exact experimental evidence")
	}
}

func TestRenderRejectsWrongEvidence(t *testing.T) {
	target := DefaultTarget()
	target.EvidenceSHA256 = "bad"
	result := Render(Input{Profile: testProfile(), Route: testRoute(), Target: target})
	if result.Applicable || !hasCode(result.Diagnostics, "jcodefork.version.evidence_mismatch") {
		t.Fatalf("evidence mismatch was not blocking: %#v", result)
	}
}

func TestRenderStrictlyRejectsUnsupportedCanonicalPolicy(t *testing.T) {
	profile := testProfile()
	profile.Tools = &profilemango.ResolvedRules{Managed: true, Closed: false, Allow: []string{"read"}}
	result := Render(Input{Profile: profile, Route: testRoute(), Target: DefaultTarget()})
	if result.Applicable || !hasCode(result.Diagnostics, "jcodefork.tools.policy_unsupported") {
		t.Fatalf("unsupported tool policy was not rejected: %#v", result)
	}
	if artifactByPath(result.Artifacts, "preview/route-only.config.toml.preview").Path != "" {
		t.Fatal("unsupported tool policy emitted a candidate")
	}
}

func TestRenderRejectsMalformedSelectors(t *testing.T) {
	tests := map[string]struct {
		allow []string
		deny  []string
		code  string
	}{
		"duplicate": {allow: []string{"read", "read"}, code: "jcodefork.selector.duplicate"},
		"newline":   {deny: []string{"write\nclose"}, code: "jcodefork.selector.invalid"},
		"empty":     {allow: []string{""}, code: "jcodefork.selector.invalid"},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			profile := testProfile()
			profile.Tools = &profilemango.ResolvedRules{Managed: true, Closed: true, Allow: test.allow, Deny: test.deny}
			result := Render(Input{Profile: profile, Route: testRoute(), Target: DefaultTarget()})
			if result.Applicable || !hasCode(result.Diagnostics, test.code) {
				t.Fatalf("selector failure was not rejected: %#v", result)
			}
			if artifactByPath(result.Artifacts, "preview/route-only.config.toml.preview").Path != "" {
				t.Fatal("malformed selector emitted a candidate")
			}
		})
	}
}

func TestRenderRejectsInvalidResources(t *testing.T) {
	profile := testProfile()
	profile.Instructions = []string{"instructions/system.md"}
	resource := ResourceFromContent("instructions/system.md", "instruction", []byte("system\n"))
	resource.Content = []byte("changed\n")
	result := Render(Input{Profile: profile, Route: testRoute(), Target: DefaultTarget(), Resources: []Resource{resource}})
	if result.Applicable || !hasCode(result.Diagnostics, "jcodefork.resource.digest_mismatch") {
		t.Fatalf("digest mismatch was not rejected: %#v", result)
	}
	if artifactByPath(result.Artifacts, "preview/route-only.config.toml.preview").Path != "" {
		t.Fatal("digest mismatch emitted a candidate")
	}

	profile.Instructions = []string{"../outside.md"}
	pathResult := Render(Input{Profile: profile, Route: testRoute(), Target: DefaultTarget()})
	if !hasCode(pathResult.Diagnostics, "jcodefork.resource.path_unsupported") {
		t.Fatalf("escaping profile resource was not rejected: %#v", pathResult.Diagnostics)
	}
}

func TestRenderDoesNotProjectUnknownCanonicalOrTargetFields(t *testing.T) {
	profile := testProfile()
	profile.Metadata.Description = "SENTINEL-DESCRIPTION"
	profile.Metadata.Labels = map[string]string{"unknown_key": "SENTINEL-UNKNOWN-KEY"}
	profile.Skills = []string{"sentinel-skill"}
	result := Render(Input{Profile: profile, Route: testRoute(), Target: DefaultTarget(), Resources: []Resource{ResourceFromContent("sentinel-skill", "skill", []byte("skill\n"))}})
	config := string(artifactByPath(result.Artifacts, "preview/route-only.config.toml.preview").Content)
	for _, forbidden := range []string{
		"SENTINEL-DESCRIPTION", "unknown_key", "SENTINEL-UNKNOWN-KEY",
		"authentication", "credential", "secret", "provider_profile", "agents_md_path", "disabled_skills",
	} {
		if strings.Contains(strings.ToLower(config), strings.ToLower(forbidden)) {
			t.Fatalf("candidate projected forbidden field %q: %s", forbidden, config)
		}
	}
	if !strings.Contains(config, `skills = ["sentinel-skill"]`) {
		t.Fatalf("known canonical skill selector missing: %s", config)
	}
}

func TestRenderEscapesCandidateValues(t *testing.T) {
	route := testRoute()
	route.Provider = "openai-api"
	route.Model = "model\"quoted"
	route.Effort = "low"
	result := Render(Input{Profile: testProfile(), Route: route, Target: DefaultTarget()})
	config := string(artifactByPath(result.Artifacts, "preview/route-only.config.toml.preview").Content)
	if !strings.Contains(config, `model = "model\"quoted"`) {
		t.Fatalf("candidate value was not TOML escaped: %s", config)
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
		if artifact.Path == "config.toml" || artifact.Path == "AGENTS.md" || artifact.Path == "plan.json" || artifact.Path == "manifest.json" || strings.HasPrefix(artifact.Path, "skills/") {
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
