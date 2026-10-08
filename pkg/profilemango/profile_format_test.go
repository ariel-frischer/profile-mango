package profilemango

import (
	"reflect"
	"testing"
)

const legacyBaseYAML = `apiVersion: profilemango.dev/v1alpha1
kind: PolicyProfile
metadata:
  name: base
  description: Shared base
  labels: {team: core}
spec:
  routeRef: primary
  permissions:
    mode: workspace-write
    network: allow
  tools:
    allow: [read, edit]
    deny: [deploy]
  instructions:
    append: [instructions/base.md]
  skills: [skills/base/SKILL.md]
`

const flatBaseYAML = `name: base
description: Shared base
labels: {team: core}
route: primary
permissions:
  mode: workspace-write
  network: allow
tools:
  allow: [read, edit]
  deny: [deploy]
instructions:
  append: [instructions/base.md]
skills: [skills/base/SKILL.md]
`

const legacyChildYAML = `apiVersion: profilemango.dev/v1alpha1
kind: PolicyProfile
metadata:
  name: review
  labels: {tier: review}
spec:
  extends: base
  routeRef: secondary
  permissions:
    mode: read-only
  tools:
    allow: [read]
  instructions:
    append: [instructions/review.md]
  skills: []
`

const flatChildYAML = `labels: {tier: review}
extends: base
route: secondary
permissions:
  mode: read-only
tools:
  allow: [read]
instructions:
  append: [instructions/review.md]
skills: []
`

func TestLegacyAndFlatProfilesResolveIdentically(t *testing.T) {
	t.Parallel()
	cases := map[string]struct {
		legacy  map[string]string
		flat    map[string]string
		resolve string
	}{
		"single":    {legacy: map[string]string{"base": legacyBaseYAML}, flat: map[string]string{"base": flatBaseYAML}, resolve: "base"},
		"inherited": {legacy: map[string]string{"base": legacyBaseYAML, "review": legacyChildYAML}, flat: map[string]string{"base": flatBaseYAML, "review": flatChildYAML}, resolve: "review"},
		"mixed":     {legacy: map[string]string{"base": legacyBaseYAML, "review": flatChildYAML}, flat: map[string]string{"base": flatBaseYAML, "review": flatChildYAML}, resolve: "review"},
	}
	for name, test := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			legacy := resolveDocuments(t, test.legacy, test.resolve)
			flat := resolveDocuments(t, test.flat, test.resolve)
			if !reflect.DeepEqual(legacy, flat) {
				t.Fatalf("legacy and flat differ\nlegacy %#v\nflat   %#v", legacy, flat)
			}
		})
	}
}

func TestLegacyProfileWarnsDeprecation(t *testing.T) {
	t.Parallel()
	cases := map[string]struct {
		yaml string
		warn bool
	}{
		"legacy": {yaml: legacyBaseYAML, warn: true},
		"flat":   {yaml: flatBaseYAML, warn: false},
	}
	for name, test := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			_, diagnostics := ParseProfile([]byte(test.yaml))
			if diagnostics.HasErrors() {
				t.Fatal(diagnostics)
			}
			if got := hasCode(diagnostics, "profile.legacy_format"); got != test.warn {
				t.Fatalf("legacy warning = %v, want %v: %v", got, test.warn, diagnostics)
			}
			for _, diagnostic := range diagnostics {
				if diagnostic.Code == "profile.legacy_format" && diagnostic.Severity != SeverityWarning {
					t.Fatalf("legacy diagnostic severity %q", diagnostic.Severity)
				}
			}
		})
	}
}

func TestParseProfileAtDefaultsAndMatchesName(t *testing.T) {
	t.Parallel()
	cases := map[string]struct {
		yaml string
		dir  string
		name string
		code string
	}{
		"omitted defaults": {yaml: "route: main\n", dir: "daily", name: "daily"},
		"matching":         {yaml: "name: daily\nroute: main\n", dir: "daily", name: "daily"},
		"mismatch":         {yaml: "name: other\nroute: main\n", dir: "daily", code: "repository.name_mismatch"},
		"legacy mismatch":  {yaml: legacyBaseYAML, dir: "daily", code: "repository.name_mismatch"},
		"invalid folder":   {yaml: "route: main\n", dir: "Daily", code: "profile.name_invalid"},
	}
	for name, test := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			profile, diagnostics := ParseProfileAt([]byte(test.yaml), test.dir)
			if test.code != "" {
				if !hasCode(diagnostics, test.code) {
					t.Fatalf("missing %s in %v", test.code, diagnostics)
				}
				return
			}
			if diagnostics.HasErrors() || profile.Name != test.name {
				t.Fatalf("name %q, diagnostics %v", profile.Name, diagnostics)
			}
		})
	}
}

func resolveDocuments(t *testing.T, documents map[string]string, name string) ResolvedProfile {
	t.Helper()
	profiles := make(map[string]PolicyProfile, len(documents))
	for dir, document := range documents {
		profile, diagnostics := ParseProfileAt([]byte(document), dir)
		if diagnostics.HasErrors() {
			t.Fatal(diagnostics)
		}
		profiles[dir] = profile
	}
	resolved, diagnostics := Resolve(profiles, name)
	if diagnostics.HasErrors() {
		t.Fatal(diagnostics)
	}
	return resolved
}
