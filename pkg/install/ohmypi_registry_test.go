package install

import "testing"

func TestDefaultRegistryOhMyPiQualification(t *testing.T) {
	cases := map[string]struct {
		target Target
		found  bool
		ready  bool
	}{
		"qualified exact version": {Target{"oh-my-pi", "18.6.0"}, true, true},
		"unknown version":         {Target{"oh-my-pi", "18.6.1"}, false, false},
		"Codex settings-only":     {Target{"codex", "0.157.1"}, true, true},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			adapter, found := DefaultRegistry().Lookup(tc.target)
			if found != tc.found {
				t.Fatalf("found = %v, want %v", found, tc.found)
			}
			if !found {
				return
			}
			metadata := adapter.Metadata()
			if metadata.Installable != tc.ready || (metadata.Status == StatusReady) != tc.ready {
				t.Fatalf("unexpected metadata: %#v", metadata)
			}
		})
	}
}
