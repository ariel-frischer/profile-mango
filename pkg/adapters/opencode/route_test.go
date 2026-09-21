package opencode

import (
	"strings"
	"testing"

	"gitlab.com/ariel-frischer/profile-mango/pkg/profilemango"
)

func TestPatchModelRejectsUnsupportedRouteRequirements(t *testing.T) {
	tests := map[string]struct {
		route profilemango.RouteBinding
		want  string
	}{
		"unsupported effort": {
			route: profilemango.RouteBinding{Provider: "openai", Transport: "native", Authentication: "oauth", Model: "gpt-5.6", Effort: "turbo"},
			want:  "unsupported effort",
		},
		"missing authentication": {
			route: profilemango.RouteBinding{Provider: "openai", Transport: "native", Model: "gpt-5.6", Effort: "high"},
			want:  "authentication mode",
		},
		"model whitespace": {
			route: profilemango.RouteBinding{Provider: "openai", Transport: "native", Authentication: "oauth", Model: "gpt 5.6", Effort: "high"},
			want:  "unsupported characters",
		},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			_, err := PatchModel([]byte(`{}`), test.route)
			if err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("error = %v, want %q", err, test.want)
			}
		})
	}
}
