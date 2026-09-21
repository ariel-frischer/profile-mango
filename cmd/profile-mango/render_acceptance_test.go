package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gitlab.com/ariel-frischer/profile-mango/pkg/adapters/arieljcode"
	"gitlab.com/ariel-frischer/profile-mango/pkg/adapters/claudecode"
	"gitlab.com/ariel-frischer/profile-mango/pkg/adapters/codex"
	"gitlab.com/ariel-frischer/profile-mango/pkg/adapters/hermes"
	"gitlab.com/ariel-frischer/profile-mango/pkg/adapters/ohmypi"
	"gitlab.com/ariel-frischer/profile-mango/pkg/adapters/openclaw"
	"gitlab.com/ariel-frischer/profile-mango/pkg/adapters/opencode"
	"gitlab.com/ariel-frischer/profile-mango/pkg/adapters/pi"
	"gitlab.com/ariel-frischer/profile-mango/pkg/profilemango"
	"gitlab.com/ariel-frischer/profile-mango/pkg/render"
)

type acceptanceTarget struct {
	name             string
	version          string
	evidence         string
	candidatePath    string
	codePrefix       string
	previewBlocker   string
	render           func(render.Input) render.Result
	experimentalOnly bool
}

func acceptanceTargets() map[string]acceptanceTarget {
	return map[string]acceptanceTarget{
		claudecode.TargetName: {
			name: claudecode.TargetName, version: claudecode.TargetVersion, evidence: claudecode.EvidenceSHA256,
			candidatePath: "preview/route-only.settings.json.preview", codePrefix: "claudecode",
			previewBlocker: "claudecode.config.acceptance_unverified", render: claudecode.Render,
		},
		codex.TargetName: {
			name: codex.TargetName, version: codex.TargetVersion, evidence: codex.EvidenceSHA256,
			candidatePath: "preview/route-only.config.toml.preview", codePrefix: "codex",
			previewBlocker: "codex.route.authentication_unverified", render: codex.Render,
		},
		pi.TargetName: {
			name: pi.TargetName, version: pi.TargetVersion, evidence: pi.EvidenceSHA256,
			candidatePath: "preview/route-only.settings.json.preview", codePrefix: "pi",
			previewBlocker: "pi.config.acceptance_unverified", render: pi.Render,
		},
		ohmypi.TargetName: {
			name: ohmypi.TargetName, version: ohmypi.TargetVersion, evidence: ohmypi.EvidenceSHA256,
			candidatePath: "preview/route-only.config.yml.preview", codePrefix: "ohmypi",
			previewBlocker: "ohmypi.config.inspector_unsafe", render: ohmypi.Render,
		},
		openclaw.TargetName: {
			name: openclaw.TargetName, version: openclaw.TargetVersion, evidence: openclaw.EvidenceSHA256,
			candidatePath: "preview/route-only.config.json5.preview", codePrefix: "openclaw",
			previewBlocker: "openclaw.config.inspector_unsafe", render: openclaw.Render,
		},
		opencode.TargetName: {
			name: opencode.TargetName, version: opencode.TargetVersion, evidence: opencode.EvidenceSHA256,
			candidatePath: "preview/route-only.opencode.jsonc.preview", codePrefix: "opencode",
			previewBlocker: "opencode.install.validation_deferred", render: opencode.Render,
		},
		hermes.TargetName: {
			name: hermes.TargetName, version: hermes.TargetVersion, evidence: hermes.EvidenceSHA256,
			candidatePath: "preview/route-only.config.yaml.preview", codePrefix: "hermes",
			previewBlocker: "hermes.config.inspector_unsafe", render: hermes.Render,
		},
		arieljcode.TargetName: {
			name: arieljcode.TargetName, version: arieljcode.TargetVersion, evidence: arieljcode.EvidenceSHA256,
			candidatePath: "preview/route-only.config.toml.preview", codePrefix: "arieljcode",
			previewBlocker: "arieljcode.experimental_only", render: arieljcode.Render, experimentalOnly: true,
		},
	}
}

func TestRenderTargetAcceptanceMatrix(t *testing.T) {
	for name, target := range acceptanceTargets() {
		t.Run(name, func(t *testing.T) {
			assertStableAcceptanceReports(t, target)
			assertAcceptanceEvidenceRejection(t, target)
			assertAcceptanceCLIBehavior(t, target)
		})
	}
}

func TestRenderArielIdentityOnEarlyFailures(t *testing.T) {
	target := acceptanceTargets()[arieljcode.TargetName]
	profiles := t.TempDir()
	resources := t.TempDir()
	bindings := writeBindingsFixture(t)
	writeFile(t, filepath.Join(profiles, "unsupported", "profile.yaml"), "apiVersion: profilemango.dev/v1alpha1\nkind: PolicyProfile\nmetadata:\n  name: unsupported\nspec:\n  routeRef: codex-oauth\nroles: [reviewer]\n")
	out := filepath.Join(t.TempDir(), "invalid-profile")
	stdout, stderr, err := executeRenderForTest(t, acceptanceCLIArgs("unsupported", profiles, resources, bindings, out, true, true, target))
	if err == nil || stderr == "" {
		t.Fatal("invalid Ariel profile did not fail with diagnostics")
	}
	assertArielEarlyIdentity(t, stdout, stderr, "invalid profile")

	validProfiles, validResources, _ := writeRenderFixture(t, false)
	malformedBindings := filepath.Join(t.TempDir(), "malformed.yaml")
	writeFile(t, malformedBindings, "routes: [\n")
	textOut := filepath.Join(t.TempDir(), "malformed-bindings")
	_, stderr, err = executeRenderForTest(t, acceptanceCLIArgs("route-only", validProfiles, validResources, malformedBindings, textOut, true, false, target))
	if err == nil {
		t.Fatal("malformed Ariel bindings unexpectedly succeeded")
	}
	if !strings.Contains(stderr, arieljcode.ExperimentalLabel) || !strings.Contains(stderr, "yaml.invalid") {
		t.Fatalf("malformed binding text lost Ariel identity: %s", stderr)
	}

	missingBindings := filepath.Join(t.TempDir(), "missing.yaml")
	missingOut := filepath.Join(t.TempDir(), "missing-bindings")
	stdout, _, err = executeRenderForTest(t, acceptanceCLIArgs("route-only", validProfiles, validResources, missingBindings, missingOut, true, true, target))
	if err == nil || !strings.Contains(stdout, arieljcode.ExperimentalLabel) || !strings.Contains(stdout, "render.bindings.read") {
		t.Fatalf("missing binding JSON lost Ariel identity: %v\n%s", err, stdout)
	}
}

func assertArielEarlyIdentity(t *testing.T, stdout, stderr, caseName string) {
	t.Helper()
	var report render.Result
	if err := json.Unmarshal([]byte(stdout), &report); err != nil {
		t.Fatalf("%s did not return JSON: %v\n%s", caseName, err, stdout)
	}
	if report.Target != arieljcode.TargetName || !hasAcceptanceCode(report.Diagnostics, "arieljcode.experimental_only") || !strings.Contains(stdout, arieljcode.ExperimentalLabel) {
		t.Fatalf("%s lost explicit Ariel identity: %#v\n%s", caseName, report, stderr)
	}
}

func assertStableAcceptanceReports(t *testing.T, target acceptanceTarget) {
	t.Helper()
	for name, input := range map[string]render.Input{
		"route-only":  acceptanceRouteInput(target),
		"constrained": acceptanceConstrainedInput(target),
	} {
		first, second := target.render(input), target.render(input)
		firstReport, err := first.ReportJSON(true)
		if err != nil {
			t.Fatalf("%s first report: %v", name, err)
		}
		secondReport, err := second.ReportJSON(true)
		if err != nil {
			t.Fatalf("%s second report: %v", name, err)
		}
		if !bytes.Equal(firstReport, secondReport) || len(first.Artifacts) != len(second.Artifacts) {
			t.Fatalf("%s report or artifact metadata is unstable", name)
		}
		for index := range first.Artifacts {
			if first.Artifacts[index].Path != second.Artifacts[index].Path || !bytes.Equal(first.Artifacts[index].Content, second.Artifacts[index].Content) {
				t.Fatalf("%s artifact %d is unstable", name, index)
			}
		}
	}
	result := target.render(acceptanceRouteInput(target))
	report, err := result.ReportJSON(true)
	if err != nil || !strings.Contains(string(report), target.evidence) || !strings.Contains(string(report), "no safe automated remedy is known") {
		t.Fatalf("%s report omitted evidence or actionable limitation: %v\n%s", target.name, err, report)
	}
	if result.Applicable || !hasAcceptanceCode(result.Diagnostics, target.previewBlocker) {
		t.Fatalf("%s was not blocked by its expected limitation: %#v", target.name, result)
	}
	if target.experimentalOnly != strings.Contains(string(report), arieljcode.ExperimentalLabel) {
		t.Fatalf("%s experimental identity mismatch: %s", target.name, report)
	}
}

func assertAcceptanceEvidenceRejection(t *testing.T, target acceptanceTarget) {
	t.Helper()
	wrongVersion := acceptanceRouteInput(target)
	wrongVersion.Target.Version = target.version + ".wrong"
	wrongVersionResult := target.render(wrongVersion)
	if !hasAcceptanceCode(wrongVersionResult.Diagnostics, target.codePrefix+".version.unsupported") {
		t.Fatalf("%s accepted an unsupported version: %#v", target.name, wrongVersionResult.Diagnostics)
	}
	wrongHash := acceptanceRouteInput(target)
	wrongHash.Target.EvidenceSHA256 = strings.Repeat("0", 64)
	wrongHashResult := target.render(wrongHash)
	if !hasAcceptanceCode(wrongHashResult.Diagnostics, target.codePrefix+".version.evidence_mismatch") {
		t.Fatalf("%s accepted an evidence mismatch: %#v", target.name, wrongHashResult.Diagnostics)
	}
	invalid := acceptanceInvalidInput(target)
	invalidResult := target.render(invalid)
	if len(invalidResult.Artifacts) != 0 || !hasAcceptanceCode(invalidResult.Diagnostics, target.codePrefix+".resource.digest_mismatch") {
		t.Fatalf("%s invalid input was not artifact-free: %#v", target.name, invalidResult)
	}
}

func assertAcceptanceCLIBehavior(t *testing.T, target acceptanceTarget) {
	t.Helper()
	assertNoPreviewWrite(t, target)
	assertPreviewStage(t, target)
	assertExistingOutputPreserved(t, target)
	assertInvalidInputNoArtifact(t, target)
}

func assertNoPreviewWrite(t *testing.T, target acceptanceTarget) {
	profiles, resources, bindings := writeRenderFixture(t, false)
	out := filepath.Join(t.TempDir(), "candidate")
	stdout, stderr, err := runAcceptanceCLI(t, target, "route-only", profiles, resources, bindings, out, false)
	if err == nil || reportHasPreview(stdout) || !strings.Contains(stderr, target.previewBlocker) {
		t.Fatalf("%s no-preview behavior was not blocked: %v\n%s\n%s", target.name, err, stdout, stderr)
	}
	if _, statErr := os.Stat(out); !os.IsNotExist(statErr) {
		t.Fatalf("%s no-preview created output: %v", target.name, statErr)
	}
}

func assertPreviewStage(t *testing.T, target acceptanceTarget) {
	profiles, resources, bindings := writeRenderFixture(t, false)
	out := filepath.Join(t.TempDir(), "candidate")
	stdout, stderr, err := runAcceptanceCLI(t, target, "route-only", profiles, resources, bindings, out, true)
	if err == nil || !reportHasPreview(stdout) || !strings.Contains(stderr, target.previewBlocker) {
		t.Fatalf("%s preview behavior was not blocked and staged: %v\n%s\n%s", target.name, err, stdout, stderr)
	}
	assertRenderFile(t, out, "render.json")
	assertRenderFile(t, out, target.candidatePath)
	assertRenderFile(t, out, "resources/instructions/system.md")
}

func assertExistingOutputPreserved(t *testing.T, target acceptanceTarget) {
	profiles, resources, bindings := writeRenderFixture(t, false)
	out := filepath.Join(t.TempDir(), "candidate")
	if err := os.Mkdir(out, 0o755); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(out, "sentinel"), "preserve")
	_, _, err := runAcceptanceCLI(t, target, "route-only", profiles, resources, bindings, out, true)
	if err == nil {
		t.Fatalf("%s overwrote an existing staging directory", target.name)
	}
	data, readErr := os.ReadFile(filepath.Join(out, "sentinel"))
	if readErr != nil || string(data) != "preserve" {
		t.Fatalf("%s changed unrelated sentinel: %v %q", target.name, readErr, data)
	}
}

func assertInvalidInputNoArtifact(t *testing.T, target acceptanceTarget) {
	profiles := t.TempDir()
	resources := t.TempDir()
	bindings := writeBindingsFixture(t)
	writeFile(t, filepath.Join(profiles, "unsupported", "profile.yaml"), "apiVersion: profilemango.dev/v1alpha1\nkind: PolicyProfile\nmetadata:\n  name: unsupported\nspec:\n  routeRef: codex-oauth\nroles: [reviewer]\n")
	out := filepath.Join(t.TempDir(), "candidate")
	stdout, _, err := runAcceptanceCLI(t, target, "unsupported", profiles, resources, bindings, out, true)
	if err == nil {
		t.Fatalf("%s invalid input unexpectedly succeeded", target.name)
	}
	var report render.Result
	if decodeErr := json.Unmarshal([]byte(stdout), &report); decodeErr != nil {
		t.Fatalf("%s invalid input did not return JSON: %v", target.name, decodeErr)
	}
	if len(report.Artifacts) != 0 {
		t.Fatalf("%s invalid input returned artifacts: %#v", target.name, report.Artifacts)
	}
	if _, statErr := os.Stat(out); !os.IsNotExist(statErr) {
		t.Fatalf("%s invalid input created output: %v", target.name, statErr)
	}
}

func runAcceptanceCLI(t *testing.T, target acceptanceTarget, profile, profiles, resources, bindings, out string, preview bool) (string, string, error) {
	t.Helper()
	return executeRenderForTest(t, acceptanceCLIArgs(profile, profiles, resources, bindings, out, preview, true, target))
}

func acceptanceCLIArgs(profile, profiles, resources, bindings, out string, preview, jsonOutput bool, target acceptanceTarget) []string {
	args := []string{profile, "--profiles", profiles, "--resource-root", resources, "--bindings", bindings, "--target", target.name, "--target-version", target.version, "--out", out}
	if preview {
		args = append(args, "--preview")
	}
	if jsonOutput {
		args = append(args, "--json")
	}
	return args
}

func acceptanceRouteInput(target acceptanceTarget) render.Input {
	return render.Input{
		Profile: profilemango.ResolvedProfile{Metadata: profilemango.Metadata{Name: "route-only"}, Instructions: []string{"instructions/system.md"}},
		Route:   acceptanceRoute(), Target: render.TargetBuild{Name: target.name, Version: target.version, EvidenceSHA256: target.evidence},
		Resources: []render.Resource{render.ResourceFromContent("instructions/system.md", "instruction", []byte("system\n"))},
	}
}

func acceptanceConstrainedInput(target acceptanceTarget) render.Input {
	mode, network, shell := "read-only", "allow", "deny"
	return render.Input{
		Profile: profilemango.ResolvedProfile{
			Metadata:     profilemango.Metadata{Name: "read-only"},
			Permissions:  &profilemango.PermissionPolicy{Mode: &mode, Network: &network, Shell: &shell},
			Tools:        &profilemango.ResolvedRules{Managed: true, Closed: true, Allow: []string{"read"}},
			Instructions: []string{"instructions/research.md"}, Skills: []string{"skills/research/SKILL.md"},
		},
		Route: acceptanceRoute(), Target: render.TargetBuild{Name: target.name, Version: target.version, EvidenceSHA256: target.evidence},
		Resources: []render.Resource{
			render.ResourceFromContent("instructions/research.md", "instruction", []byte("research\n")),
			render.ResourceFromContent("skills/research/SKILL.md", "skill", []byte("skill\n")),
		},
	}
}

func acceptanceInvalidInput(target acceptanceTarget) render.Input {
	input := acceptanceRouteInput(target)
	input.Profile.Metadata.Name = "unsupported"
	input.Resources[0].Content = []byte("changed\n")
	return input
}

func acceptanceRoute() profilemango.RouteBinding {
	return profilemango.RouteBinding{Provider: "openai", Transport: "native", Authentication: "oauth", Model: "gpt-5.6", Effort: "high"}
}

func hasAcceptanceCode(diagnostics profilemango.Diagnostics, code string) bool {
	for _, diagnostic := range diagnostics {
		if diagnostic.Code == code {
			return true
		}
	}
	return false
}

func reportHasPreview(data string) bool {
	return strings.Contains(data, `"preview": true`)
}
