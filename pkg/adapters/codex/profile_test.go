package codex

import (
	"strings"
	"testing"
)

func TestProfileFileName(t *testing.T) {
	if got := ProfileFileName("coding"); got != "coding.config.toml" {
		t.Fatalf("profile file = %q", got)
	}
}

func TestValidProfileName(t *testing.T) {
	tests := map[string]bool{
		"coding":        true,
		"code_Review-2": true,
		"":              false,
		"a.b":           false,
		"a/b":           false,
		"has space":     false,
		"ünïcode":       false,
	}
	for name, want := range tests {
		if got := ValidProfileName(name); got != want {
			t.Fatalf("ValidProfileName(%q) = %v, want %v", name, got, want)
		}
	}
}

func TestCheckBaseConfigAllowsUnrelatedState(t *testing.T) {
	sources := map[string]string{
		"missing":          "",
		"root model":       "model = \"root\"\n",
		"unrelated legacy": "[profiles.dev]\nmodel = \"legacy\"\n",
		"other provider":   "[model_providers.other]\nname = \"x\"\n",
	}
	for name, source := range sources {
		t.Run(name, func(t *testing.T) {
			if err := CheckBaseConfig([]byte(source), "coding"); err != nil {
				t.Fatalf("CheckBaseConfig = %v", err)
			}
		})
	}
}

func TestCheckBaseConfigRejectsStateThatBreaksTheProfile(t *testing.T) {
	tests := map[string]struct {
		source string
		want   string
	}{
		"legacy selector":      {source: "profile = \"dev\"\n", want: "legacy profile = setting"},
		"legacy same table":    {source: "[profiles.coding]\nmodel = \"legacy\"\n", want: "legacy [profiles.coding] table"},
		"legacy inline table":  {source: "profiles = { coding = { model = \"legacy\" } }\n", want: "legacy [profiles.coding] table"},
		"provider shadow":      {source: "[model_providers.openai]\nname = \"shadow\"\n", want: "provider override state"},
		"endpoint override":    {source: "openai_base_url = \"https://shadow.invalid/v1\"\n", want: "provider override state"},
		"malformed":            {source: "[features\n", want: "invalid codex TOML"},
		"invalid profile name": {source: "", want: "profile name"},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			profile := "coding"
			if name == "invalid profile name" {
				profile = "a.b"
			}
			err := CheckBaseConfig([]byte(test.source), profile)
			if err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("CheckBaseConfig = %v, want %q", err, test.want)
			}
		})
	}
}
