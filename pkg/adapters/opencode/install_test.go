package opencode

import (
	"bytes"
	"strings"
	"testing"

	"gitlab.com/ariel-frischer/profile-mango/pkg/profilemango"
)

func TestPatchModelTable(t *testing.T) {
	tests := map[string]struct {
		source string
		want   string
		before string
	}{
		"missing file": {
			want: "{\n  \"model\": \"openai/gpt-5.6\"\n}\n",
		},
		"replace only value": {
			source: "{\r\n  // keep\r\n  \"model\" : \"old/model\",\r\n  \"unknown\": true,\r\n}\r\n",
			want:   "{\r\n  // keep\r\n  \"model\" : \"openai/gpt-5.6\",\r\n  \"unknown\": true,\r\n}\r\n",
			before: "old/model",
		},
		"insert before target owned bytes": {
			source: "{\n  // credential-shaped synthetic state\n  \"provider\": {\"openai\": {\"options\": {\"apiKey\": \"SYNTHETIC\"}}},\n}\n",
			want:   "{\"model\":\"openai/gpt-5.6\",\n  // credential-shaped synthetic state\n  \"provider\": {\"openai\": {\"options\": {\"apiKey\": \"SYNTHETIC\"}}},\n}\n",
		},
		"empty object with comment": {
			source: "{/* keep */}",
			want:   "{\"model\":\"openai/gpt-5.6\"/* keep */}",
		},
		"comment adjacent to primitive": {
			source: `{"unknown":true/* keep */}`,
			want:   `{"model":"openai/gpt-5.6","unknown":true/* keep */}`,
		},
		"escaped old value": {
			source: "{\"model\":\"old\\u002fmodel\",\"nested\":{\"model\":42}}",
			want:   "{\"model\":\"openai/gpt-5.6\",\"nested\":{\"model\":42}}",
			before: "old/model",
		},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			patch, err := PatchModel([]byte(test.source), installTestRoute())
			if err != nil {
				t.Fatal(err)
			}
			if string(patch.Content) != test.want || patch.Before != test.before || patch.After != "openai/gpt-5.6" {
				t.Fatalf("patch = %#v\ncontent:\n%s", patch, patch.Content)
			}
			second, err := PatchModel([]byte(test.source), installTestRoute())
			if err != nil || !bytes.Equal(patch.Content, second.Content) {
				t.Fatalf("patch is not deterministic: %v", err)
			}
		})
	}
}

func TestPatchModelRejectsAmbiguousOrUnsupportedInput(t *testing.T) {
	tests := map[string]struct {
		source string
		route  profilemango.RouteBinding
		want   string
	}{
		"duplicate model":      {source: `{ "model": "a/b", "model": "c/d" }`, route: installTestRoute(), want: "duplicate"},
		"non-string model":     {source: `{ "model": false }`, route: installTestRoute(), want: "must be a string"},
		"unquoted key":         {source: `{ model: "a/b" }`, route: installTestRoute(), want: "quoted string"},
		"trailing content":     {source: `{}` + ` {}`, route: installTestRoute(), want: "trailing content"},
		"unterminated comment": {source: `{ /*`, route: installTestRoute(), want: "unterminated"},
		"byte order mark":      {source: "\ufeff{}", route: installTestRoute(), want: "byte-order mark"},
		"wrong transport":      {source: `{}`, route: profilemango.RouteBinding{Provider: "openai", Model: "gpt-5.6", Transport: "proxy"}, want: "native transport"},
		"provider slash":       {source: `{}`, route: profilemango.RouteBinding{Provider: "open/ai", Model: "gpt-5.6", Transport: "native"}, want: "unsupported characters"},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			_, err := PatchModel([]byte(test.source), test.route)
			if err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("error = %v, want %q", err, test.want)
			}
		})
	}
}

func installTestRoute() profilemango.RouteBinding {
	return profilemango.RouteBinding{Provider: "openai", Transport: "native", Authentication: "oauth", Model: "gpt-5.6", Effort: "high"}
}
