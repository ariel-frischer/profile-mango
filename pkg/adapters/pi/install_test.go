package pi

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/ariel-frischer/profile-mango/pkg/profilemango"
)

func TestPatchSettingsTable(t *testing.T) {
	route := installTestRoute()
	tests := map[string]struct {
		data   []byte
		exists bool
		want   string
		fields []SettingChange
	}{
		"missing file": {
			want: "{\n  \"defaultProvider\": \"openai\",\n  \"defaultModel\": \"gpt-5.6\",\n  \"defaultThinkingLevel\": \"high\"\n}\n",
			fields: []SettingChange{
				{Path: "defaultProvider", After: "openai"},
				{Path: "defaultModel", After: "gpt-5.6"},
				{Path: "defaultThinkingLevel", After: "high"},
			},
		},
		"replace target values losslessly": {
			exists: true,
			data:   []byte("{\r\n  \"defaultProvider\" : \"old/provider\",\r\n  \"defaultModel\":\"old-model\",\r\n  \"defaultThinkingLevel\": \"low\",\r\n  \"unknown\": {\"keep\": true},\r\n  \"provider\": {\"sentinel\": {\"options\": {\"apiKey\": \"SYNTHETIC\"}}}\r\n}\r\n"),
			want:   "{\r\n  \"defaultProvider\" : \"openai\",\r\n  \"defaultModel\":\"gpt-5.6\",\r\n  \"defaultThinkingLevel\": \"high\",\r\n  \"unknown\": {\"keep\": true},\r\n  \"provider\": {\"sentinel\": {\"options\": {\"apiKey\": \"SYNTHETIC\"}}}\r\n}\r\n",
			fields: []SettingChange{
				{Path: "defaultProvider", Before: "old/provider", After: "openai"},
				{Path: "defaultModel", Before: "old-model", After: "gpt-5.6"},
				{Path: "defaultThinkingLevel", Before: "low", After: "high"},
			},
		},
		"insert missing keys without rewriting existing bytes": {
			exists: true,
			data:   []byte("{\n  \"unknown\": true\n}\n"),
			want:   "{\"defaultProvider\":\"openai\",\"defaultModel\":\"gpt-5.6\",\"defaultThinkingLevel\":\"high\",\n  \"unknown\": true\n}\n",
			fields: []SettingChange{
				{Path: "defaultProvider", After: "openai"},
				{Path: "defaultModel", After: "gpt-5.6"},
				{Path: "defaultThinkingLevel", After: "high"},
			},
		},
		"preserve BOM": {
			exists: true,
			data:   []byte("\xef\xbb\xbf{\"defaultModel\":\"old\"}"),
			want:   "\xef\xbb\xbf{\"defaultProvider\":\"openai\",\"defaultThinkingLevel\":\"high\",\"defaultModel\":\"gpt-5.6\"}",
		},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			patch, err := PatchSettings(test.data, test.exists, route)
			if err != nil {
				t.Fatal(err)
			}
			if string(patch.Content) != test.want {
				t.Fatalf("content = %q, want %q", patch.Content, test.want)
			}
			if len(test.fields) > 0 && !equalSettingChanges(patch.Fields, test.fields) {
				t.Fatalf("fields = %#v, want %#v", patch.Fields, test.fields)
			}
			if _, err := parseSettingsJSON(patch.Content); err != nil {
				t.Fatalf("patched content is not JSON: %v", err)
			}
			second, err := PatchSettings(test.data, test.exists, route)
			if err != nil || !bytes.Equal(patch.Content, second.Content) {
				t.Fatalf("patch is not deterministic: %v", err)
			}
		})
	}
}

func TestPatchSettingsRejectsMalformedOrAmbiguousInput(t *testing.T) {
	route := installTestRoute()
	tests := map[string]struct {
		data string
		want string
	}{
		"empty existing file":  {data: "", want: "valid UTF-8 JSON"},
		"array root":           {data: "[]", want: "top-level JSON object"},
		"duplicate key":        {data: `{"defaultModel":"a","defaultModel":"b"}`, want: "duplicate key"},
		"nested duplicate key": {data: `{"unknown":{"keep":true,"keep":false}}`, want: "duplicate key"},
		"null target value":    {data: `{"defaultModel":null}`, want: "must be a string"},
		"number target value":  {data: `{"defaultModel":42}`, want: "must be a string"},
		"trailing comma":       {data: `{"defaultModel":"a",}`, want: "valid UTF-8 JSON"},
		"comment":              {data: `{/* keep */}`, want: "valid UTF-8 JSON"},
		"trailing content":     {data: `{} {}`, want: "valid UTF-8 JSON"},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			_, err := PatchSettings([]byte(test.data), true, route)
			if err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("error = %v, want %q", err, test.want)
			}
		})
	}
}

func TestValidateInstallRouteTable(t *testing.T) {
	tests := map[string]struct {
		route profilemango.RouteBinding
		want  string
	}{
		"wrong transport": {route: installTestRouteWith(func(route *profilemango.RouteBinding) { route.Transport = "proxy" }), want: "native transport"},
		"missing auth":    {route: installTestRouteWith(func(route *profilemango.RouteBinding) { route.Authentication = "" }), want: "authentication"},
		"bad thinking":    {route: installTestRouteWith(func(route *profilemango.RouteBinding) { route.Effort = "ultra" }), want: "thinking level"},
		"unsafe model":    {route: installTestRouteWith(func(route *profilemango.RouteBinding) { route.Model = "gpt\n5" }), want: "unsupported characters"},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			if err := ValidateInstallRoute(test.route); err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("error = %v, want %q", err, test.want)
			}
		})
	}
}

func installTestRoute() profilemango.RouteBinding {
	return profilemango.RouteBinding{Provider: "openai", Transport: "native", Authentication: "oauth", Model: "gpt-5.6", Effort: "high"}
}

func installTestRouteWith(update func(*profilemango.RouteBinding)) profilemango.RouteBinding {
	route := installTestRoute()
	update(&route)
	return route
}

func equalSettingChanges(left, right []SettingChange) bool {
	if len(left) != len(right) {
		return false
	}
	for index := range left {
		if left[index] != right[index] {
			return false
		}
	}
	return true
}

func parseSettingsJSON(data []byte) (map[string]any, error) {
	var value map[string]any
	if err := json.Unmarshal(bytes.TrimPrefix(data, []byte{0xef, 0xbb, 0xbf}), &value); err != nil {
		return nil, err
	}
	return value, nil
}
