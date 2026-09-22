package opencode

import (
	"bytes"
	"strings"
	"testing"
)

func TestPatchModelJSONCLosslessReplacement(t *testing.T) {
	source := "{\r\n  // preserve comments\r\n  \"model\" : \"old\\u002fmodel\",\r\n  \"provider\": {\"options\": {\"apiKey\": \"SYNTHETIC\"}},\r\n  \"unknown\": [true, {\"nested\": 1,}],\r\n}\r\n"
	want := "{\r\n  // preserve comments\r\n  \"model\" : \"openai/gpt-5.6\",\r\n  \"provider\": {\"options\": {\"apiKey\": \"SYNTHETIC\"}},\r\n  \"unknown\": [true, {\"nested\": 1,}],\r\n}\r\n"
	patch, err := PatchModelJSONC([]byte(source), "openai/gpt-5.6")
	if err != nil {
		t.Fatal(err)
	}
	if patch.Before != "old/model" || !patch.Found || patch.Inserted {
		t.Fatalf("patch metadata = %#v", patch)
	}
	if string(patch.Content) != want {
		t.Fatalf("content = %q, want %q", patch.Content, want)
	}
}

func TestPatchModelJSONCInsertionIsDeterministic(t *testing.T) {
	tests := map[string]struct {
		source string
		want   string
	}{
		"multiline CRLF": {
			source: "{\r\n  // keep\r\n  \"provider\": {\"options\": {\"apiKey\": \"SYNTHETIC\"}},\r\n}\r\n",
			want:   "{\"model\":\"openai/gpt-5.6\",\r\n  // keep\r\n  \"provider\": {\"options\": {\"apiKey\": \"SYNTHETIC\"}},\r\n}\r\n",
		},
		"comment-only": {
			source: "{/* keep */}",
			want:   "{\"model\":\"openai/gpt-5.6\"/* keep */}",
		},
		"empty": {
			source: "{}",
			want:   "{\"model\":\"openai/gpt-5.6\"}",
		},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			first, err := PatchModelJSONC([]byte(test.source), "openai/gpt-5.6")
			if err != nil {
				t.Fatal(err)
			}
			second, err := PatchModelJSONC([]byte(test.source), "openai/gpt-5.6")
			if err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(first.Content, second.Content) || string(first.Content) != test.want || !first.Inserted {
				t.Fatalf("first = %#v, second = %#v", first, second)
			}
		})
	}
}

func TestPatchModelJSONCNoopPreservesEscapes(t *testing.T) {
	source := []byte(`{"model":"openai\/gpt-5.6","unknown":true}`)
	patch, err := PatchModelJSONC(source, "openai/gpt-5.6")
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(patch.Content, source) || patch.Before != "openai/gpt-5.6" || !patch.Found {
		t.Fatalf("no-op patch = %#v", patch)
	}
}

func TestPatchModelJSONCRejectsAmbiguousInput(t *testing.T) {
	tests := map[string]string{
		"duplicate model":      `{ "model": "a/b", "model": "c/d" }`,
		"duplicate unknown":    `{ "provider": 1, "provider": 2 }`,
		"non-string model":     `{ "model": false }`,
		"top-level array":      `[]`,
		"trailing content":     `{}` + ` {}`,
		"unterminated comment": `{ /*` + "\n",
		"invalid escape":       `{ "model": "bad\q" }`,
		"malformed array":      `{ "unknown": [1,,2] }`,
		"byte order mark":      "\ufeff{}",
		"unpaired surrogate":   `{ "model": "bad\ud800" }`,
		"unsafe desired value": `{}`,
	}
	for name, source := range tests {
		t.Run(name, func(t *testing.T) {
			model := "openai/gpt-5.6"
			if name == "unsafe desired value" {
				model = "openai/gpt\n5.6"
			}
			_, err := PatchModelJSONC([]byte(source), model)
			if err == nil {
				t.Fatal("ambiguous or unsafe input was accepted")
			}
		})
	}
}

func TestPatchModelJSONCRejectsInvalidEncoding(t *testing.T) {
	_, err := PatchModelJSONC([]byte{'{', '}', 0xff}, "openai/gpt-5.6")
	if err == nil || !strings.Contains(err.Error(), "UTF-8") {
		t.Fatalf("error = %v", err)
	}
}

func TestPatchSkillsJSONCPreservesUnrelatedConfig(t *testing.T) {
	source := "{\r\n  // keep\r\n  \"skills\" : {\r\n    \"urls\": [\"https://example.invalid/skills\"],\r\n    \"paths\": [\"existing\" /* keep path comment */],\r\n  },\r\n  \"unknown\": true,\r\n}\r\n"
	patch, err := PatchSkillsJSONC([]byte(source), []string{"/state/config-dir"})
	if err != nil {
		t.Fatal(err)
	}
	want := "{\r\n  // keep\r\n  \"skills\" : {\r\n    \"urls\": [\"https://example.invalid/skills\"],\r\n    \"paths\": [\"existing\" /* keep path comment */,\"/state/config-dir\"],\r\n  },\r\n  \"unknown\": true,\r\n}\r\n"
	if string(patch.Content) != want {
		t.Fatalf("content = %q, want %q", patch.Content, want)
	}
	if strings.Join(patch.Before, ",") != "existing" || strings.Join(patch.After, ",") != "existing,/state/config-dir" {
		t.Fatalf("patch metadata = %#v", patch)
	}
	if len(patch.Added) != 1 || patch.Added[0] != "/state/config-dir" {
		t.Fatalf("added paths = %#v", patch.Added)
	}
	second, err := PatchSkillsJSONC([]byte(source), []string{"/state/config-dir"})
	if err != nil || !bytes.Equal(patch.Content, second.Content) {
		t.Fatalf("patch is not deterministic: %v", err)
	}
}

func TestRemoveSkillPathJSONCPreservesUnrelatedConfig(t *testing.T) {
	source := "{\r\n  // keep\r\n  \"skills\" : {\r\n    \"urls\": [\"https://example.invalid/skills\"],\r\n    \"paths\": [\"existing\" /* keep path comment */,\"/state/config-dir\"],\r\n  },\r\n  \"unknown\": true,\r\n}\r\n"
	patch, err := RemoveSkillPathJSONC([]byte(source), "/state/config-dir")
	if err != nil {
		t.Fatal(err)
	}
	want := "{\r\n  // keep\r\n  \"skills\" : {\r\n    \"urls\": [\"https://example.invalid/skills\"],\r\n    \"paths\": [\"existing\" /* keep path comment */],\r\n  },\r\n  \"unknown\": true,\r\n}\r\n"
	if string(patch.Content) != want || !patch.Removed {
		t.Fatalf("content=%q removed=%v, want %q and removed", patch.Content, patch.Removed, want)
	}
	if strings.Join(patch.Before, ",") != "existing,/state/config-dir" || strings.Join(patch.After, ",") != "existing" {
		t.Fatalf("patch metadata = %#v", patch)
	}
}

func TestRemoveSkillPathJSONCNoopAndRejectsDuplicate(t *testing.T) {
	tests := map[string]struct {
		source string
		want   string
		err    bool
	}{
		"missing": {
			source: `{ "skills": { "paths": ["existing"] } }`,
			want:   `{ "skills": { "paths": ["existing"] } }`,
		},
		"duplicate": {
			source: `{ "skills": { "paths": ["/state/config-dir", "/state/config-dir"] } }`,
			err:    true,
		},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			patch, err := RemoveSkillPathJSONC([]byte(test.source), "/state/config-dir")
			if test.err {
				if err == nil {
					t.Fatal("duplicate managed path was accepted")
				}
				return
			}
			if err != nil || string(patch.Content) != test.want || patch.Removed {
				t.Fatalf("patch=%#v err=%v", patch, err)
			}
		})
	}
}

func TestPatchSkillsJSONCInsertsAndNoops(t *testing.T) {
	tests := map[string]struct {
		source string
		want   string
	}{
		"missing skills": {
			source: `{ "model": "openai/gpt-5.6", "unknown": true }`,
			want:   `{"skills":{"paths":["/state/config-dir"]}, "model": "openai/gpt-5.6", "unknown": true }`,
		},
		"empty skills": {
			source: `{ "skills": {} }`,
			want:   `{ "skills": {"paths":["/state/config-dir"]} }`,
		},
		"existing path noop": {
			source: `{ "skills": { "paths": ["/state/config-dir"] } }`,
			want:   `{ "skills": { "paths": ["/state/config-dir"] } }`,
		},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			patch, err := PatchSkillsJSONC([]byte(test.source), []string{"/state/config-dir"})
			if err != nil {
				t.Fatal(err)
			}
			if string(patch.Content) != test.want {
				t.Fatalf("content = %q, want %q", patch.Content, test.want)
			}
		})
	}
}

func TestPatchSkillsJSONCRejectsAmbiguousInput(t *testing.T) {
	tests := map[string]string{
		"skills is not object": `{ "skills": false }`,
		"paths is not array":   `{ "skills": { "paths": false } }`,
		"path is not string":   `{ "skills": { "paths": [false] } }`,
		"duplicate paths":      `{ "skills": { "paths": [], "paths": [] } }`,
	}
	for name, source := range tests {
		t.Run(name, func(t *testing.T) {
			if _, err := PatchSkillsJSONC([]byte(source), []string{"/state/config-dir"}); err == nil {
				t.Fatal("ambiguous skills config was accepted")
			}
		})
	}
}
