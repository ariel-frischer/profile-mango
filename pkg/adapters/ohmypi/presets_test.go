package ohmypi

import (
	"strings"
	"testing"

	"github.com/ariel-frischer/profile-mango/pkg/profilemango"
)

func TestPatchPresetsPreservesAndReleases(t *testing.T) {
	route := profilemango.RouteBinding{Provider: "openai", Model: "gpt-6", Transport: "native", Authentication: "oauth", Effort: "high"}
	routes := map[string]profilemango.RouteBinding{"sol": route}
	cases := map[string]string{
		"missing":     "# keep\ntheme: dark\n",
		"user preset": "modelPresets:\n  personal: # keep preset\n    modelRoles:\n      default: mine/model # keep role\n    defaultThinkingLevel: low\ntheme: dark\n",
	}
	for name, source := range cases {
		t.Run(name, func(t *testing.T) {
			patch, err := PatchPresets([]byte(source), routes)
			if err != nil {
				t.Fatal(err)
			}
			if !strings.Contains(string(patch.Content), "mango-sol:") || !strings.Contains(string(patch.Content), "task: \"openai/gpt-6:high\"") {
				t.Fatalf("missing complete preset: %s", patch.Content)
			}
			if name == "user preset" && !strings.Contains(string(patch.Content), "default: mine/model # keep role") {
				t.Fatal("changed user preset")
			}
			again, err := PatchPresets(patch.Content, routes)
			if err != nil || string(again.Content) != string(patch.Content) {
				t.Fatalf("not idempotent: %v", err)
			}
			released, _, err := ReleasePresets(patch.Content, map[string]string{"mango-sol": ""})
			if err != nil || string(released) != source {
				t.Fatalf("release = %s, %v; want %s", released, err, source)
			}
		})
	}
}

func TestPatchPresetsRestoresPriorEntry(t *testing.T) {
	source := "modelPresets:\n  mango-sol: # prior\n    modelRoles: {default: old/model}\n    defaultThinkingLevel: low\n  personal:\n    modelRoles: {default: mine/model}\n"
	route := profilemango.RouteBinding{Provider: "openai", Model: "gpt-6", Transport: "native", Authentication: "oauth", Effort: "high"}
	patch, err := PatchPresets([]byte(source), map[string]profilemango.RouteBinding{"sol": route})
	if err != nil {
		t.Fatal(err)
	}
	released, _, err := ReleasePresets(patch.Content, map[string]string{"mango-sol": patch.Presets[0].Prior})
	if err != nil || string(released) != source {
		t.Fatalf("restore = %s, %v", released, err)
	}
}
