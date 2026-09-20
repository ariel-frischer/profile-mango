package agentprofile

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestParseProfileRejectsStrictInputFailures(t *testing.T) {
	t.Parallel()
	cases := map[string]struct {
		yaml string
		code string
	}{
		"unknown key":   {yaml: validProfileYAML() + "unknown: true\n", code: "yaml.strict"},
		"duplicate key": {yaml: strings.Replace(validProfileYAML(), "kind: PolicyProfile", "kind: PolicyProfile\nkind: PolicyProfile", 1), code: "yaml.duplicate_key"},
		"null":          {yaml: strings.Replace(validProfileYAML(), "routeRef: research-primary", "routeRef: null", 1), code: "yaml.null"},
		"version":       {yaml: strings.Replace(validProfileYAML(), APIVersion, "agentprofiles.dev/v9", 1), code: "profile.api_version"},
	}
	for name, test := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			_, diagnostics := ParseProfile([]byte(test.yaml))
			if !hasCode(diagnostics, test.code) {
				t.Fatalf("expected %s, got %#v", test.code, diagnostics)
			}
		})
	}
}

func TestResolveInheritanceAndDenyWins(t *testing.T) {
	t.Parallel()
	base := mustParse(t, `apiVersion: agentprofiles.dev/v1alpha1
kind: PolicyProfile
metadata:
  name: base
  labels: {team: core, tier: base}
spec:
  routeRef: primary
  permissions:
    mode: workspace-write
    network: allow
    shell: allow
  tools:
    allow: [read, edit, shell]
    deny: [deploy]
  instructions:
    append: [instructions/base.md]
  skills: [skills/base/SKILL.md]
`)
	child := mustParse(t, `apiVersion: agentprofiles.dev/v1alpha1
kind: PolicyProfile
metadata:
  name: research
  labels: {tier: research}
spec:
  extends: base
  permissions:
    mode: read-only
    shell: deny
  tools:
    allow: [read, search, edit]
    deny: [edit]
  instructions:
    append: [instructions/research.md]
  skills: []
`)
	resolved, diagnostics := Resolve(map[string]PolicyProfile{"base": base, "research": child}, "research")
	if diagnostics.HasErrors() {
		t.Fatal(diagnostics)
	}
	if got, want := resolved.RouteRef, "primary"; got != want {
		t.Fatalf("route %q, want %q", got, want)
	}
	if got := *resolved.Permissions.Network; got != "allow" {
		t.Fatalf("network %q", got)
	}
	if got := *resolved.Permissions.Shell; got != "deny" {
		t.Fatalf("shell %q", got)
	}
	if !reflect.DeepEqual(resolved.Tools.Allow, []string{"read", "search"}) {
		t.Fatalf("allow %#v", resolved.Tools.Allow)
	}
	if !resolved.Tools.Closed {
		t.Fatal("present allowlist must be closed")
	}
	if !reflect.DeepEqual(resolved.Instructions, []string{"instructions/base.md", "instructions/research.md"}) {
		t.Fatalf("instructions %#v", resolved.Instructions)
	}
	if len(resolved.Skills) != 0 {
		t.Fatalf("explicit empty skills must clear parent: %#v", resolved.Skills)
	}
	if resolved.Metadata.Labels["team"] != "core" || resolved.Metadata.Labels["tier"] != "research" {
		t.Fatalf("labels %#v", resolved.Metadata.Labels)
	}
}

func TestResolveFailures(t *testing.T) {
	t.Parallel()
	missing := mustParse(t, strings.Replace(validProfileYAML(), "name: research", "name: child", 1)+"  extends: absent\n")
	_, diagnostics := Resolve(map[string]PolicyProfile{"child": missing}, "child")
	if !hasCode(diagnostics, "resolution.missing_profile") {
		t.Fatalf("missing parent: %#v", diagnostics)
	}

	a := mustParse(t, profileWithParent("a", "b"))
	b := mustParse(t, profileWithParent("b", "a"))
	_, diagnostics = Resolve(map[string]PolicyProfile{"a": a, "b": b}, "a")
	if !hasCode(diagnostics, "resolution.cycle") {
		t.Fatalf("cycle: %#v", diagnostics)
	}
}

func TestDigestResourcesDeterministicAndSafe(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	mustWrite(t, filepath.Join(root, "instructions", "system.md"), "system\n")
	mustWrite(t, filepath.Join(root, "skills", "research", "SKILL.md"), "skill\n")
	profile := ResolvedProfile{Instructions: []string{"instructions/system.md"}, Skills: []string{"skills/research/SKILL.md"}}
	first, diagnostics := DigestResources(root, profile)
	if diagnostics.HasErrors() {
		t.Fatal(diagnostics)
	}
	secondRoot := t.TempDir()
	mustWrite(t, filepath.Join(secondRoot, "instructions", "system.md"), "system\n")
	mustWrite(t, filepath.Join(secondRoot, "skills", "research", "SKILL.md"), "skill\n")
	second, diagnostics := DigestResources(secondRoot, profile)
	if diagnostics.HasErrors() {
		t.Fatal(diagnostics)
	}
	if !reflect.DeepEqual(first, second) {
		t.Fatalf("root-dependent digest\n%#v\n%#v", first, second)
	}

	_, diagnostics = DigestResources(root, ResolvedProfile{Instructions: []string{"../outside.md"}})
	if !hasCode(diagnostics, "resource.path_escape") {
		t.Fatalf("traversal: %#v", diagnostics)
	}

	outside := filepath.Join(t.TempDir(), "outside.md")
	mustWrite(t, outside, "outside")
	if err := os.Symlink(outside, filepath.Join(root, "escape.md")); err != nil {
		t.Fatal(err)
	}
	_, diagnostics = DigestResources(root, ResolvedProfile{Instructions: []string{"escape.md"}})
	if !hasCode(diagnostics, "resource.symlink_escape") {
		t.Fatalf("symlink: %#v", diagnostics)
	}
}

func TestCheckedInFixtures(t *testing.T) {
	t.Parallel()
	root := filepath.Join("testdata")
	profiles := map[string]PolicyProfile{}
	for _, name := range []string{"route-only", "read-only"} {
		data, err := os.ReadFile(filepath.Join(root, "fixtures", name, "profile.yaml"))
		if err != nil {
			t.Fatal(err)
		}
		profiles[name] = mustParse(t, string(data))
	}
	resolved, diagnostics := Resolve(profiles, "read-only")
	if diagnostics.HasErrors() {
		t.Fatal(diagnostics)
	}
	resources, diagnostics := DigestResources(root, resolved)
	if diagnostics.HasErrors() {
		t.Fatal(diagnostics)
	}
	if len(resources) != 3 {
		t.Fatalf("resources %d, want 3", len(resources))
	}
	data, err := CanonicalJSON(resolved)
	if err != nil {
		t.Fatal(err)
	}
	golden, err := os.ReadFile(filepath.Join(root, "golden", "read-only.resolved.json"))
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != string(golden) {
		t.Fatalf("golden mismatch\n%s", data)
	}

	unsupported, err := os.ReadFile(filepath.Join(root, "fixtures", "unsupported", "profile.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	_, diagnostics = ParseProfile(unsupported)
	if !hasCode(diagnostics, "yaml.strict") {
		t.Fatalf("unsupported fixture must fail strictly: %#v", diagnostics)
	}

	bindingsData, err := os.ReadFile(filepath.Join(root, "fixtures", "bindings.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	bindings, diagnostics := ParseBindings(bindingsData)
	if diagnostics.HasErrors() {
		t.Fatal(diagnostics)
	}
	plan, err := BuildPlan(resolved, "codex", bindings, resources)
	if err != nil {
		t.Fatal(err)
	}
	if plan.Route.Authentication != "oauth" {
		t.Fatalf("route %#v", plan.Route)
	}
}

func TestVersionedContractsRoundTrip(t *testing.T) {
	t.Parallel()
	plan := Plan{APIVersion: PlanVersion, Kind: "Plan", Profile: "research", Target: "codex", Route: RouteBinding{Provider: "openai", Transport: "native", Authentication: "oauth", Model: "gpt", Effort: "high"}}
	data, err := CanonicalJSON(plan)
	if err != nil {
		t.Fatal(err)
	}
	var decoded Plan
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(plan, decoded) {
		t.Fatalf("round trip %#v", decoded)
	}
	manifest := Manifest{APIVersion: ManifestVersion, Kind: "Manifest", Owner: "agent-profile", Generation: 1, Profile: "research", Target: "codex"}
	if _, err := CanonicalJSON(manifest); err != nil {
		t.Fatal(err)
	}
}

func mustParse(t *testing.T, input string) PolicyProfile {
	t.Helper()
	profile, diagnostics := ParseProfile([]byte(input))
	if diagnostics.HasErrors() {
		t.Fatal(diagnostics)
	}
	return profile
}

func hasCode(diagnostics Diagnostics, code string) bool {
	for _, diagnostic := range diagnostics {
		if diagnostic.Code == code {
			return true
		}
	}
	return false
}

func mustWrite(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func validProfileYAML() string {
	return `apiVersion: agentprofiles.dev/v1alpha1
kind: PolicyProfile
metadata:
  name: research
spec:
  routeRef: research-primary
`
}

func profileWithParent(name, parent string) string {
	return `apiVersion: agentprofiles.dev/v1alpha1
kind: PolicyProfile
metadata:
  name: ` + name + `
spec:
  extends: ` + parent + `
`
}
