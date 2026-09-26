package install

import (
	"strings"
	"testing"
)

func TestRegistryResolveTargetTable(t *testing.T) {
	adapter := func(target, version string, installable bool) Adapter {
		return testAdapter{metadata: AdapterMetadata{Target: target, Version: version, Installable: installable}}
	}
	registry := NewRegistry(
		adapter("one", "1.0.0", true),
		adapter("one", "0.9.0", false),
		adapter("two", "1.0.0", true),
		adapter("two", "2.0.0", true),
		adapter("blocked", "1.0.0", false),
	)
	tests := map[string]struct {
		value   string
		want    Target
		wantErr string
	}{
		"bare single qualified":        {value: "one", want: Target{Name: "one", Version: "1.0.0"}},
		"explicit qualified":           {value: "two@2.0.0", want: Target{Name: "two", Version: "2.0.0"}},
		"explicit unqualified is kept": {value: "one@0.9.0", want: Target{Name: "one", Version: "0.9.0"}},
		"explicit unregistered kept":   {value: "one@9.9.9", want: Target{Name: "one", Version: "9.9.9"}},
		"bare several qualified":       {value: "two", wantErr: "several qualified versions (two@1.0.0, two@2.0.0)"},
		"bare only unqualified":        {value: "blocked", wantErr: "blocked has no qualified version"},
		"bare unknown":                 {value: "missing", wantErr: `unknown target "missing"; valid targets: blocked, one, two`},
		"empty":                        {value: "", wantErr: "target or target@version"},
		"empty version":                {value: "one@", wantErr: "target or target@version"},
		"path name":                    {value: "../one", wantErr: "simple name"},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			got, err := registry.ResolveTarget(test.value)
			if test.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), test.wantErr) {
					t.Fatalf("error = %v, want %q", err, test.wantErr)
				}
				return
			}
			if err != nil || got != test.want {
				t.Fatalf("got %v, %v; want %v", got, err, test.want)
			}
		})
	}
}

func TestParseTargetStillRequiresVersion(t *testing.T) {
	if _, err := ParseTarget("codex"); err == nil || !strings.Contains(err.Error(), "exact target@version") {
		t.Fatalf("error = %v", err)
	}
}
