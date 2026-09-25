package claudecode

import (
	"strings"
	"testing"
)

func TestPatchSettingsEffortLevel(t *testing.T) {
	tests := map[string]struct {
		source, effort, want, before string
	}{
		"absent file":        {effort: "medium", want: "{\n  \"model\": \"claude-sonnet-4-5\",\n  \"effortLevel\": \"medium\"\n}\n"},
		"replace in place":   {source: "{\"effortLevel\": \"low\", \"model\": \"claude-sonnet-4-5\", \"keep\": 1}", effort: "xhigh", want: "{\"effortLevel\": \"xhigh\", \"model\": \"claude-sonnet-4-5\", \"keep\": 1}", before: "low"},
		"unsupported kept":   {source: "{\"effortLevel\": \"low\"}", effort: "max", want: "{\"model\":\"claude-sonnet-4-5\",\"effortLevel\": \"low\"}"},
		"unsupported absent": {effort: "ultra", want: "{\n  \"model\": \"claude-sonnet-4-5\"\n}\n"},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			route := claudeRoute()
			route.Effort = test.effort
			patch, err := PatchSettings([]byte(test.source), route)
			if err != nil {
				t.Fatal(err)
			}
			if string(patch.Content) != test.want || patch.EffortBefore != test.before {
				t.Fatalf("patch = %q (before %q), want %q (before %q)", patch.Content, patch.EffortBefore, test.want, test.before)
			}
		})
	}
}

func TestPatchSettingsRejectsAmbiguousEffortLevel(t *testing.T) {
	tests := map[string]string{
		"non-string": `{"effortLevel": 3}`,
		"duplicate":  `{"effortLevel": "low", "effortLevel": "high"}`,
	}
	for name, source := range tests {
		t.Run(name, func(t *testing.T) {
			_, err := PatchSettings([]byte(source), claudeRoute())
			if err == nil || !strings.Contains(err.Error(), "effortLevel") {
				t.Fatalf("error = %v, want effortLevel rejection", err)
			}
		})
	}
}
