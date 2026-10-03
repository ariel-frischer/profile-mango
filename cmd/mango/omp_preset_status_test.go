package main

import (
	"bytes"
	"strings"
	"testing"

	"github.com/ariel-frischer/profile-mango/pkg/install"
)

func TestHumanModelPresetStatus(t *testing.T) {
	var output bytes.Buffer
	preset := &install.ModelPresetStatus{Name: "mango-sol", Route: "sol", TaskMaxEffort: "high", RouteSubagentMaxEffort: "medium", MaxEffortDiffers: true}
	if err := writeHumanModelPreset(&output, preset); err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"roles match mango preset sol (mango-sol; switched via omp)", "task.maxEffort stays high from the applied profile; preset route requests medium (not switched)"} {
		if !strings.Contains(output.String(), want) {
			t.Fatalf("missing %q: %s", want, output.String())
		}
	}
}
