package render

import (
	"strings"
	"testing"

	"gitlab.com/ariel-frischer/profile-mango/pkg/profilemango"
)

func TestResourceArtifactsUsesCanonicalBytesAndStablePaths(t *testing.T) {
	profile := profilemango.ResolvedProfile{
		Metadata:     profilemango.Metadata{Name: "route-only"},
		Instructions: []string{"instructions/system.md"},
	}
	resources := []Resource{ResourceFromContent("instructions/system.md", "instruction", []byte("system\n"))}
	artifacts, diagnostics := ResourceArtifacts(profile, resources, ResourceArtifactOptions{
		CodePrefix:         "test",
		InstructionMessage: "instruction delivery is unverified",
	})
	if len(artifacts) != 1 || artifacts[0].Path != "resources/instructions/system.md" {
		t.Fatalf("unexpected artifacts: %#v", artifacts)
	}
	if !HasCodePrefix(diagnostics, "test.instructions.") || diagnostics.HasErrors() == false {
		t.Fatalf("delivery diagnostic missing: %#v", diagnostics)
	}
}

func TestResourceArtifactsRejectsDigestMismatch(t *testing.T) {
	profile := profilemango.ResolvedProfile{
		Metadata:     profilemango.Metadata{Name: "route-only"},
		Instructions: []string{"instructions/system.md"},
	}
	resource := ResourceFromContent("instructions/system.md", "instruction", []byte("system\n"))
	resource.Content = []byte("changed\n")
	artifacts, diagnostics := ResourceArtifacts(profile, []Resource{resource}, ResourceArtifactOptions{CodePrefix: "test"})
	if len(artifacts) != 0 || !HasCodePrefix(diagnostics, "test.resource.digest_mismatch") {
		t.Fatalf("digest mismatch not rejected: %#v %#v", artifacts, diagnostics)
	}
}

func TestUnknownTargetHasNoArtifacts(t *testing.T) {
	result := UnknownTarget("route-only", TargetBuild{Name: "unknown", Version: "1.0.0"})
	data, err := result.ReportJSON(true)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Artifacts) != 0 || !HasCodePrefix(result.Diagnostics, "render.target.unsupported") {
		t.Fatalf("unexpected unknown report: %#v", result)
	}
	if !strings.Contains(string(data), "render.target.unsupported") || strings.Contains(string(data), "candidate-config") {
		t.Fatalf("unknown report contains unexpected output: %s", data)
	}
}
