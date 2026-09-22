package claudecode

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"gitlab.com/ariel-frischer/profile-mango/pkg/profilemango"
)

func TestPatchModelJSONTable(t *testing.T) {
	tests := map[string]struct {
		source   string
		want     string
		before   string
		inserted bool
	}{
		"missing file": {
			want:     "{\n  \"model\": \"claude-sonnet-4-5\"\n}\n",
			inserted: true,
		},
		"replace preserves unrelated bytes": {
			source: "{\r\n\t\"model\" : \"old/model\",\r\n\t\"unknown\": true,\r\n\t\"credentials\": \"SYNTHETIC-CREDENTIAL\"\r\n}\r\n",
			want:   "{\r\n\t\"model\" : \"claude-sonnet-4-5\",\r\n\t\"unknown\": true,\r\n\t\"credentials\": \"SYNTHETIC-CREDENTIAL\"\r\n}\r\n",
			before: "old/model",
		},
		"insert before unknown state": {
			source:   "{\n  \"unknown\": {\"nested\": true},\n  \"apiKey\": \"SYNTHETIC-CREDENTIAL\"\n}\n",
			want:     "{\"model\":\"claude-sonnet-4-5\",\n  \"unknown\": {\"nested\": true},\n  \"apiKey\": \"SYNTHETIC-CREDENTIAL\"\n}\n",
			inserted: true,
		},
		"replace escaped value": {
			source: "{\"model\":\"old\\u002fmodel\",\"nested\":{\"model\":42}}",
			want:   "{\"model\":\"claude-sonnet-4-5\",\"nested\":{\"model\":42}}",
			before: "old/model",
		},
		"same value is byte stable": {
			source: "{\n  \"model\": \"claude-sonnet-4-5\",\n  \"unknown\": false\n}\n",
			want:   "{\n  \"model\": \"claude-sonnet-4-5\",\n  \"unknown\": false\n}\n",
			before: "claude-sonnet-4-5",
		},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			patch, err := PatchModelJSON([]byte(test.source), "claude-sonnet-4-5")
			if err != nil {
				t.Fatal(err)
			}
			if string(patch.Content) != test.want || patch.Before != test.before || patch.Inserted != test.inserted {
				t.Fatalf("patch = %#v\ncontent:\n%s", patch, patch.Content)
			}
			if !json.Valid(patch.Content) {
				t.Fatalf("patched settings are invalid JSON: %s", patch.Content)
			}
		})
	}
}

func TestPatchModelJSONRejectsAmbiguousOrUnsafeInput(t *testing.T) {
	tests := map[string]struct {
		source string
		want   string
	}{
		"malformed":            {source: `{ "model": `, want: "valid JSON"},
		"duplicate top key":    {source: `{ "model": "a", "model": "b" }`, want: "duplicate"},
		"duplicate nested key": {source: `{ "unknown": { "x": 1, "x": 2 } }`, want: "duplicate"},
		"non-string model":     {source: `{ "model": false }`, want: "must be a string"},
		"comment":              {source: `{ /* no JSON comments */ }`, want: "valid JSON"},
		"trailing comma":       {source: `{ "unknown": true, }`, want: "valid JSON"},
		"byte order mark":      {source: "\ufeff{}", want: "UTF-8 without"},
		"array root":           {source: `[]`, want: "top-level JSON object"},
		"trailing content":     {source: `{} {}`, want: "valid JSON"},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			_, err := PatchModelJSON([]byte(test.source), "claude-sonnet-4-5")
			if err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("error = %v, want %q", err, test.want)
			}
		})
	}
}

func TestPatchModelRouteSafetyTable(t *testing.T) {
	tests := map[string]struct {
		mutate func(*profilemango.RouteBinding)
		want   string
	}{
		"provider is not encoded": {
			mutate: func(route *profilemango.RouteBinding) { route.Provider = "openai" },
			want:   "provider anthropic",
		},
		"transport is not encoded": {
			mutate: func(route *profilemango.RouteBinding) { route.Transport = "proxy" },
			want:   "native transport",
		},
		"authentication is required": {
			mutate: func(route *profilemango.RouteBinding) { route.Authentication = "" },
			want:   "authentication mode",
		},
		"effort must be known": {
			mutate: func(route *profilemango.RouteBinding) { route.Effort = "unsupported" },
			want:   "unsupported effort",
		},
		"model cannot contain surrounding whitespace": {
			mutate: func(route *profilemango.RouteBinding) { route.Model = " claude-sonnet-4-5" },
			want:   "surrounding whitespace",
		},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			route := claudeRoute()
			test.mutate(&route)
			if _, err := PatchModel(nil, route); err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("error = %v, want %q", err, test.want)
			}
		})
	}
}

func TestPatchModelDoesNotCopyCredentialFields(t *testing.T) {
	source := []byte(`{"model":"old","apiKey":"SYNTHETIC-CREDENTIAL","oauthToken":"SYNTHETIC-TOKEN"}`)
	patch, err := PatchModel(source, claudeRoute())
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(patch.Content, []byte(`"apiKey":"SYNTHETIC-CREDENTIAL"`)) || !bytes.Contains(patch.Content, []byte(`"oauthToken":"SYNTHETIC-TOKEN"`)) {
		t.Fatalf("credential-shaped unrelated fields were not preserved: %s", patch.Content)
	}
	if patch.Before != "old" || patch.After != "claude-sonnet-4-5" {
		t.Fatalf("model field change = %#v", patch)
	}
}

func claudeRoute() profilemango.RouteBinding {
	return profilemango.RouteBinding{
		Provider:       "anthropic",
		Transport:      "native",
		Authentication: "oauth",
		Model:          "claude-sonnet-4-5",
		Effort:         "high",
	}
}
