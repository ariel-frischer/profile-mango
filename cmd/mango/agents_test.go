package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/ariel-frischer/profile-mango/internal/agentcheck"
)

func TestAgentsCheckShowsProgressWithoutPollutingJSON(t *testing.T) {
	started := make(chan struct{})
	release := make(chan struct{})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		close(started)
		select {
		case <-release:
			_, _ = w.Write([]byte("fixture"))
		case <-r.Context().Done():
		}
	}))
	defer server.Close()
	defer close(release)
	manifest := `{"schema_version":1,"targets":[{"id":"fixture","sources":[{"url":"` + server.URL + `","kind":"documentation"}]}]}`
	path := filepath.Join(t.TempDir(), "sources.json")
	if err := os.WriteFile(path, []byte(manifest), 0o600); err != nil {
		t.Fatal(err)
	}
	cmd := newAgentsCheckCmd()
	cmd.SetArgs([]string{"--manifest", path, "--json"})
	var stdout, stderr bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	done := make(chan error, 1)
	go func() { done <- cmd.Execute() }()
	select {
	case <-started:
	case <-time.After(time.Second):
		t.Fatal("source request did not start")
	}
	if got := stderr.String(); !strings.Contains(got, "1/1") || !strings.Contains(got, "fixture") {
		t.Fatalf("no progress while source is pending: %q", got)
	}
	if got := stdout.String(); got != "" {
		t.Fatalf("stdout before report = %q", got)
	}
	release <- struct{}{}
	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("agents check: %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("agents check did not finish")
	}
	var report agentcheck.Report
	if err := json.Unmarshal(stdout.Bytes(), &report); err != nil {
		t.Fatalf("JSON stdout invalid: %v: %q", err, stdout.String())
	}
	if report.Summary.Unversioned != 1 || !strings.Contains(stderr.String(), "unversioned") {
		t.Fatalf("report and progress disagree: %+v, %q", report.Summary, stderr.String())
	}
}
