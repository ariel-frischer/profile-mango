package main

import (
	"bytes"
	"strings"
	"testing"

	"github.com/ariel-frischer/profile-mango/pkg/install"
	"github.com/ariel-frischer/profile-mango/pkg/profilemango"
)

func TestCompactPlanNamesPresetEffects(t *testing.T) {
	effects := compactFieldEffects([]install.FieldChange{{Path: "config.modelPresets.mango-sol", After: `{"modelRoles":{"default":"openai/gpt-6"}}`}}, map[string]struct{}{}, true)
	if len(effects) != 1 || effects[0] != "preset mango-sol create" {
		t.Fatalf("effects = %v", effects)
	}
	var output bytes.Buffer
	target := install.TargetPlan{}
	target.Diagnostics.Add(profilemango.SeverityWarning, "ohmypi.install.overwrite_preset", "config.modelRoles", "overwrites mango preset sol (mango-sol)", 0, 0)
	if err := writeCompactWarnings(&output, target, stylesFor(&output, false)); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(output.String(), "overwrites mango preset sol") {
		t.Fatalf("missing preset overwrite note: %s", output.String())
	}
}
