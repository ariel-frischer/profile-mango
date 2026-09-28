package openclaw

import (
	"strings"
	"testing"
)

func TestSetAgentSkillsRoundTripsLosslessly(t *testing.T) {
	tests := map[string]struct {
		config, value, want string
	}{
		"insert beside other keys": {
			config: "// keep\n{agents:{entries:{docs:{ model: 'x' /* c */ }}}, other: [1,]}\n",
			value:  `["a","b"]`,
			want:   "// keep\n{agents:{entries:{docs:{skills:[\"a\",\"b\"], model: 'x' /* c */ }}}, other: [1,]}\n",
		},
		"insert into empty entry": {
			config: `{"agents":{"entries":{"docs":{}}}}`,
			value:  `["a"]`,
			want:   `{"agents":{"entries":{"docs":{skills:["a"]}}}}`,
		},
		"replace existing list": {
			config: "{agents:{entries:{docs:{\n  skills: ['old'], // mine\n}}}}",
			value:  `[]`,
			want:   "{agents:{entries:{docs:{\n  skills: [], // mine\n}}}}",
		},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			patched, err := SetAgentSkills([]byte(test.config), "docs", test.value)
			if err != nil || string(patched) != test.want {
				t.Fatalf("patched = %q, %v; want %q", patched, err, test.want)
			}
			if value, found, err := AgentSkills(patched, "docs"); err != nil || !found || value != test.value {
				t.Fatalf("read back = %q %v %v", value, found, err)
			}
		})
	}
}

func TestSetAgentSkillsRemovesKeyWithItsComma(t *testing.T) {
	tests := map[string]struct{ config, want string }{
		"first of several": {config: "{agents:{entries:{docs:{skills:[\"a\"], model: 'x' }}}}", want: "{agents:{entries:{docs:{ model: 'x' }}}}"},
		"last of several":  {config: "{agents:{entries:{docs:{model: 'x', skills: [\"a\"]}}}}", want: "{agents:{entries:{docs:{model: 'x'}}}}"},
		"only key":         {config: "{agents:{entries:{docs:{skills:[\"a\"]}}}}", want: "{agents:{entries:{docs:{}}}}"},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			patched, err := SetAgentSkills([]byte(test.config), "docs", "")
			if err != nil || string(patched) != test.want {
				t.Fatalf("patched = %q, %v; want %q", patched, err, test.want)
			}
			if value, found, err := AgentSkills(patched, "docs"); err != nil || !found || value != "" {
				t.Fatalf("removed key still read as %q %v %v", value, found, err)
			}
		})
	}
}

func TestAgentSkillsRejectsUnsafeShapes(t *testing.T) {
	tests := map[string]struct{ config, want string }{
		"entries not object": {config: "{agents:{entries:[]}}", want: "agents.entries must be an object"},
		"entry not object":   {config: "{agents:{entries:{docs:true}}}", want: "agents.entries.docs must be an object"},
		"skills not strings": {config: "{agents:{entries:{docs:{skills:[1]}}}}", want: "array of strings"},
		"skills not array":   {config: "{agents:{entries:{docs:{skills:'a'}}}}", want: "array of strings"},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			if _, _, err := AgentSkills([]byte(test.config), "docs"); err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("error = %v, want %q", err, test.want)
			}
		})
	}
	if _, err := SetAgentSkills([]byte("{agents:{entries:{main:{}}}}"), "docs", `["a"]`); err == nil || !strings.Contains(err.Error(), "agents.entries.docs does not exist") {
		t.Fatalf("missing entry error = %v", err)
	}
}

func TestValidAgentID(t *testing.T) {
	for id, want := range map[string]bool{"docs": true, "locked-down": true, "_ops": true, "Docs": false, "a/b": false, "": false} {
		if ValidAgentID(id) != want {
			t.Fatalf("ValidAgentID(%q) = %v", id, !want)
		}
	}
}
