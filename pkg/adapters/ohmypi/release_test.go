package ohmypi

import (
	"reflect"
	"testing"
)

func TestReleaseSettingsTable(t *testing.T) {
	tests := map[string]struct {
		source  string
		prior   map[string]string
		want    string
		changes []SettingChange
	}{
		"absent before removes the key and its emptied block": {
			source:  "modelRoles:\n  default: a/b\ntask:\n  maxEffort: \"high\"\nother: 1\n",
			prior:   map[string]string{"task.maxEffort": ""},
			want:    "modelRoles:\n  default: a/b\nother: 1\n",
			changes: []SettingChange{{Path: "task.maxEffort", Before: "high"}},
		},
		"absent before keeps a block with unrelated keys": {
			source:  "task:\n  isolation: none\n  maxEffort: \"high\"\n",
			prior:   map[string]string{"task.maxEffort": ""},
			want:    "task:\n  isolation: none\n",
			changes: []SettingChange{{Path: "task.maxEffort", Before: "high"}},
		},
		"recorded prior is restored in place": {
			source:  "task:\n  maxEffort: \"medium\" # mine\n",
			prior:   map[string]string{"task.maxEffort": "max"},
			want:    "task:\n  maxEffort: \"max\" # mine\n",
			changes: []SettingChange{{Path: "task.maxEffort", Before: "medium", After: "max"}},
		},
		"key already gone is left alone": {
			source: "task:\n  isolation: none\n",
			prior:  map[string]string{"task.maxEffort": ""},
			want:   "task:\n  isolation: none\n",
		},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			content, changes, err := ReleaseSettings([]byte(test.source), test.prior)
			if err != nil {
				t.Fatal(err)
			}
			if string(content) != test.want {
				t.Fatalf("content = %q, want %q", content, test.want)
			}
			if !reflect.DeepEqual(changes, test.changes) {
				t.Fatalf("changes = %#v, want %#v", changes, test.changes)
			}
		})
	}
}
