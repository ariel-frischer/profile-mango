package agentcheck

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

func TestLoadRepositoryManifest(t *testing.T) {
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	manifestPath := filepath.Join(filepath.Dir(filename), "..", "..", "docs", "dev", "agents", "sources.json")
	manifest, err := LoadManifest(manifestPath)
	if err != nil {
		t.Fatalf("LoadManifest() error = %v", err)
	}
	if manifest.SchemaVersion != 1 || len(manifest.Targets) != 7 {
		t.Fatalf("manifest shape = version %d, %d targets", manifest.SchemaVersion, len(manifest.Targets))
	}
	if manifest.Targets[0].Sources[0].URL == "" {
		t.Fatal("manifest source URL was not loaded")
	}
}

func TestCheckRequiredSourceStates(t *testing.T) {
	stableBody := []byte("stable source")
	changedBody := []byte("changed source")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/unchanged":
			_, _ = w.Write(stableBody)
		case "/changed":
			_, _ = w.Write(changedBody)
		case "/unversioned":
			_, _ = w.Write([]byte("unversioned source"))
		case "/unavailable":
			http.Error(w, "fixture unavailable", http.StatusServiceUnavailable)
		case "/relocated":
			http.Redirect(w, r, "/relocated-final", http.StatusFound)
		case "/relocated-final":
			_, _ = w.Write(stableBody)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	manifest := Manifest{
		SchemaVersion: 1,
		RetrievedAt:   "2026-09-20",
		Targets: []Target{{
			ID:            "fixture",
			Name:          "Fixture Agent",
			ProductStatus: "developer_only",
			VersionContext: map[string]json.RawMessage{
				"kind":    json.RawMessage(`"release_pin"`),
				"version": json.RawMessage(`"v1.2.3"`),
			},
			Sources: []Source{
				{URL: server.URL + "/unavailable", Kind: "mutable_unversioned_documentation", SHA256: digest(stableBody)},
				{URL: server.URL + "/changed", Kind: "mutable_unversioned_schema", SHA256: digest([]byte("old source"))},
				{URL: server.URL + "/relocated", Kind: "release_pinned_documentation", SHA256: digest(stableBody)},
				{URL: server.URL + "/unchanged", Kind: "mutable_unversioned_documentation", SHA256: digest(stableBody)},
				{URL: server.URL + "/unversioned", Kind: "mutable_unversioned_documentation"},
				{Locator: "/synthetic/private/jcode/docs/SYSTEM_PROMPT_CONFIG.md", Kind: "local_snapshot_documentation"},
			},
		}},
	}
	manifestPath := writeManifest(t, manifest)

	report, err := CheckFile(context.Background(), manifestPath, Options{Timeout: time.Second})
	if err != nil {
		t.Fatalf("CheckFile() error = %v", err)
	}
	wantSummary := Summary{Unchanged: 1, Changed: 1, Unversioned: 1, Unavailable: 1, Relocated: 1, NotChecked: 1}
	if report.Summary != wantSummary {
		t.Fatalf("summary = %#v, want %#v", report.Summary, wantSummary)
	}

	states := make(map[string]Status)
	for _, source := range report.Targets[0].Sources {
		states[sourceKey(Source{URL: source.URL, Locator: source.Locator})] = source.State
		if source.State == StatusChanged && source.ChangedArea != "schema" {
			t.Errorf("changed source category = %q, want schema", source.ChangedArea)
		}
	}
	tests := map[string]struct {
		path  string
		state Status
	}{
		"unchanged":   {"/unchanged", StatusUnchanged},
		"changed":     {"/changed", StatusChanged},
		"unversioned": {"/unversioned", StatusUnversioned},
		"unavailable": {"/unavailable", StatusUnavailable},
		"relocated":   {"/relocated", StatusRelocated},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			if got := states[server.URL+test.path]; got != test.state {
				t.Fatalf("state = %q, want %q", got, test.state)
			}
		})
	}
	if states["/synthetic/private/jcode/docs/SYSTEM_PROMPT_CONFIG.md"] != StatusNotChecked {
		t.Fatalf("local source state = %q, want %q", states["/synthetic/private/jcode/docs/SYSTEM_PROMPT_CONFIG.md"], StatusNotChecked)
	}

	var text strings.Builder
	if err := FormatText(&text, report); err != nil {
		t.Fatalf("FormatText() error = %v", err)
	}
	for _, want := range []string{"target: fixture", "version=\"v1.2.3\"", "category=schema", "content drift is not a compatibility regression"} {
		if !strings.Contains(text.String(), want) {
			t.Errorf("text report missing %q:\n%s", want, text.String())
		}
	}
}

func TestFailedFetchIsUnavailable(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "not used", http.StatusInternalServerError)
	}))
	url := server.URL
	server.Close()
	manifest := Manifest{
		SchemaVersion: 1,
		Targets: []Target{{
			ID:      "fixture",
			Sources: []Source{{URL: url, Kind: "mutable_unversioned_documentation", SHA256: digest([]byte("stable"))}},
		}},
	}
	report, err := Check(context.Background(), manifest, Options{Timeout: 100 * time.Millisecond})
	if err != nil {
		t.Fatalf("Check() error = %v", err)
	}
	source := report.Targets[0].Sources[0]
	if source.State != StatusUnavailable || report.Summary.Unchanged != 0 {
		t.Fatalf("failed fetch = state %q, summary %#v", source.State, report.Summary)
	}
}

func TestFormatJSONIsStructured(t *testing.T) {
	report := Report{SchemaVersion: 1, ManifestPath: "sources.json", Summary: Summary{Changed: 1}}
	var output strings.Builder
	if err := FormatJSON(&output, report); err != nil {
		t.Fatalf("FormatJSON() error = %v", err)
	}
	var decoded Report
	if err := json.Unmarshal([]byte(output.String()), &decoded); err != nil {
		t.Fatalf("JSON report decode error = %v", err)
	}
	if decoded.Summary.Changed != 1 || !strings.Contains(output.String(), "\"summary\"") {
		t.Fatalf("JSON report = %s", output.String())
	}
}

func writeManifest(t *testing.T, manifest Manifest) string {
	t.Helper()
	data, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		t.Fatalf("marshal manifest: %v", err)
	}
	path := filepath.Join(t.TempDir(), "sources.json")
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatalf("write manifest: %v", err)
	}
	return path
}

func digest(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

func ExampleFormatText() {
	report := Report{
		ManifestPath: "docs/dev/agents/sources.json",
		RetrievedAt:  "2026-09-20",
		Targets: []TargetReport{{
			ID:            "codex",
			Name:          "Codex",
			ProductStatus: "candidate",
			VersionContext: map[string]json.RawMessage{
				"version": json.RawMessage(`"0.154.0"`),
			},
			Sources: []SourceReport{{
				URL:            "https://example.test/schema.json",
				Kind:           "mutable_unversioned_schema",
				ChangedArea:    "schema",
				State:          StatusUnchanged,
				Recommendation: "No source refresh is indicated; this does not establish target compatibility.",
			}},
		}},
		Summary: Summary{Unchanged: 1},
	}
	if err := FormatText(os.Stdout, report); err != nil {
		fmt.Println(err)
	}
	// Output:
	// manifest: docs/dev/agents/sources.json
	// retrieved_at: 2026-09-20
	// target: codex (Codex) product_status=candidate version_context=version="0.154.0"
	//   source: https://example.test/schema.json kind=mutable_unversioned_schema category=schema state=unchanged
	//     recommendation: No source refresh is indicated; this does not establish target compatibility.
	// summary: unchanged=1 changed=0 unversioned=0 unavailable=0 relocated=0 not_checked=0
}
