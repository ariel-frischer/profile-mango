package testenv

import (
	"os"
	"path/filepath"
	"testing"
)

func TestWithin(t *testing.T) {
	root := filepath.Join(string(filepath.Separator), "sandbox")
	tests := map[string]struct {
		path string
		want bool
	}{
		"root":      {path: root, want: true},
		"child":     {path: filepath.Join(root, "home", ".codex"), want: true},
		"sibling":   {path: root + "-other", want: false},
		"parent":    {path: filepath.Dir(root), want: false},
		"real home": {path: filepath.Join(string(filepath.Separator), "home", "user"), want: false},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			if got := Within(root, test.path); got != test.want {
				t.Fatalf("Within(%q) = %v", test.path, got)
			}
		})
	}
}

func TestSandboxIsolatesHomeAndRelocations(t *testing.T) {
	for _, name := range append([]string{"HOME", "XDG_CONFIG_HOME"}, RelocationVars...) {
		t.Setenv(name, "/real/"+name)
	}
	root, cleanup, err := Sandbox()
	if err != nil {
		t.Fatal(err)
	}
	defer cleanup()
	if home, _ := os.UserHomeDir(); !Within(root, home) || !Within(root, os.Getenv("XDG_CONFIG_HOME")) {
		t.Fatalf("home=%q xdg=%q not under %q", home, os.Getenv("XDG_CONFIG_HOME"), root)
	}
	for _, name := range RelocationVars {
		if value, set := os.LookupEnv(name); set {
			t.Fatalf("%s still set to %q", name, value)
		}
	}
}
