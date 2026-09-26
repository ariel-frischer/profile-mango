package main

import (
	"bytes"
	"strings"
	"testing"

	"github.com/ariel-frischer/profile-mango/internal/agentcheck"
	"github.com/ariel-frischer/profile-mango/pkg/install"
	"github.com/ariel-frischer/profile-mango/pkg/profilemango"
)

func TestOutputStylesColorModes(t *testing.T) {
	tests := map[string]struct {
		enabled  bool
		noColor  string
		term     string
		wantANSI bool
	}{
		"terminal enabled": {enabled: true, term: "xterm", wantANSI: true},
		"non-terminal":     {enabled: false, term: "xterm"},
		"NO_COLOR":         {enabled: true, noColor: "1", term: "xterm"},
		"dumb terminal":    {enabled: true, term: "dumb"},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			t.Setenv("NO_COLOR", test.noColor)
			t.Setenv("TERM", test.term)
			got := newOutputStyles(test.enabled).success("valid")
			if strings.Contains(got, "\x1b[") != test.wantANSI {
				t.Fatalf("ANSI presence = %t, want %t in %q", strings.Contains(got, "\x1b["), test.wantANSI, got)
			}
			if visible := stripANSI(got); visible != "valid" {
				t.Fatalf("visible value = %q, want %q", visible, "valid")
			}
		})
	}
}

func TestCompactInstallWarningAndPathStyles(t *testing.T) {
	tests := map[string]struct {
		enabled       bool
		noColor, term string
		wantANSI      bool
	}{
		"terminal":  {true, "", "xterm", true},
		"nonTTY":    {false, "", "xterm", false},
		"NO_COLOR":  {true, "1", "xterm", false},
		"TERM dumb": {true, "", "dumb", false},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			t.Setenv("NO_COLOR", test.noColor)
			t.Setenv("TERM", test.term)
			var output bytes.Buffer
			target := install.TargetPlan{Target: install.Target{Name: "codex", Version: "0.154.0"}, Status: install.StatusReady,
				Config:              &install.ConfigDestination{Path: "/sandbox/config.toml"},
				Files:               []install.FilePlan{{Path: "profile.toml", Action: install.ActionCreate}},
				SkippedRequirements: []install.SkippedRequirement{{Requirement: "tools"}},
				VersionCheck:        &install.VersionCheck{Status: install.VersionOutOfRange, Detected: "0.155.0", Range: ">=0.154.0 <0.155.0"}}
			if err := writeCompactTarget(&output, target, newOutputStyles(test.enabled)); err != nil {
				t.Fatal(err)
			}
			got := output.String()
			if strings.Contains(got, "\x1b[") != test.wantANSI {
				t.Fatalf("ANSI mismatch: %q", got)
			}
			for _, want := range []string{"destination: /sandbox/config.toml", "files: profile.toml create", "warning: installed version", "not installed for this agent: tools"} {
				if !strings.Contains(stripANSI(got), want) {
					t.Fatalf("missing %q in %q", want, got)
				}
			}
		})
	}
}

func TestStylesForNonTTYAndStructuredOutputArePlain(t *testing.T) {
	tests := map[string]struct {
		humanOutput bool
	}{
		"buffered human output": {humanOutput: true},
		"structured output":     {humanOutput: false},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			var output bytes.Buffer
			got := stylesFor(&output, test.humanOutput).success("valid")
			if strings.Contains(got, "\x1b[") {
				t.Fatalf("non-TTY output contains ANSI: %q", got)
			}
		})
	}
}

func TestStyledDiagnosticLinePreservesVisibleContent(t *testing.T) {
	t.Setenv("NO_COLOR", "")
	t.Setenv("TERM", "xterm")
	diagnostic := profilemango.Diagnostic{
		Severity: profilemango.SeverityError,
		Path:     "spec.routeRef",
		Message:  "routeRef is missing",
		Code:     "route.missing",
	}
	styles := newOutputStyles(true)
	got := styledDiagnosticLine(diagnostic, styles)
	if !strings.Contains(got, "\x1b[") {
		t.Fatalf("styled diagnostic has no ANSI: %q", got)
	}
	if visible := stripANSI(got); visible != diagnosticLine(diagnostic) {
		t.Fatalf("visible diagnostic = %q, want %q", visible, diagnosticLine(diagnostic))
	}
}

func TestStyledAgentStatusPreservesVisibleContent(t *testing.T) {
	t.Setenv("NO_COLOR", "")
	t.Setenv("TERM", "xterm")
	styles := newOutputStyles(true)
	for _, status := range []string{"unchanged", "changed", "unavailable", "not_checked"} {
		got := styledAgentStatus(agentStatus(status), styles)
		if !strings.Contains(got, "\x1b[") {
			t.Fatalf("status %q has no ANSI: %q", status, got)
		}
		if visible := stripANSI(got); visible != status {
			t.Fatalf("visible status = %q, want %q", visible, status)
		}
	}
}

func agentStatus(value string) agentcheck.Status {
	return agentcheck.Status(value)
}

func stripANSI(value string) string {
	for {
		start := strings.Index(value, "\x1b[")
		if start < 0 {
			return value
		}
		end := strings.IndexByte(value[start:], 'm')
		if end < 0 {
			return value
		}
		value = value[:start] + value[start+end+1:]
	}
}
