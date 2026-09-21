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
