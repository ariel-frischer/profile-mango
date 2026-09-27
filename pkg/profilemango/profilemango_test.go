package profilemango

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
		"unknown key":            {yaml: validProfileYAML() + "unknown: true\n", code: "yaml.strict"},
		"duplicate key":          {yaml: validProfileYAML() + "route: other\n", code: "yaml.duplicate_key"},
		"null":                   {yaml: strings.Replace(validProfileYAML(), "route: research-primary", "route: null", 1), code: "yaml.null"},
		"invalid name":           {yaml: strings.Replace(validProfileYAML(), "name: research", "name: Research", 1), code: "profile.name_invalid"},
		"old route key":          {yaml: strings.Replace(validProfileYAML(), "route:", "routeRef:", 1), code: "yaml.strict"},
		"mixed wrapper":          {yaml: validProfileYAML() + "kind: PolicyProfile\n", code: "yaml.strict"},
		"legacy version":         {yaml: "apiVersion: profilemango.dev/v9\nkind: PolicyProfile\nmetadata:\n  name: research\nspec:\n  routeRef: r\n", code: "profile.api_version"},
		"global unknown target":  {yaml: validProfileYAML() + "globalInstructions:\n  claude:\n    CLAUDE.md: a.md\n", code: "profile.global_instructions_target_unknown"},
		"global nested file":     {yaml: validProfileYAML() + "globalInstructions:\n  codex:\n    rules/AGENTS.md: a.md\n", code: "profile.global_instructions_file_invalid"},
		"global escaping source": {yaml: validProfileYAML() + "globalInstructions:\n  codex:\n    AGENTS.md: ../secret.md\n", code: "profile.global_instructions_path_invalid"},
		"agent unknown target":   {yaml: validProfileYAML() + "agentFiles:\n  omp:\n    scout.md: a.md\n", code: "profile.agent_files_target_unknown"},
		"agent home target":      {yaml: validProfileYAML() + "agentFiles:\n  home:\n    scout.md: a.md\n", code: "profile.agent_files_target_unknown"},
		"agent nested file":      {yaml: validProfileYAML() + "agentFiles:\n  oh-my-pi:\n    sub/scout.md: a.md\n", code: "profile.agent_files_file_invalid"},
		"agent bad extension":    {yaml: validProfileYAML() + "agentFiles:\n  oh-my-pi:\n    scout.txt: a.md\n", code: "profile.agent_files_file_invalid"},
		"agent escaping source":  {yaml: validProfileYAML() + "agentFiles:\n  oh-my-pi:\n    scout.md: ../secret.md\n", code: "profile.agent_files_path_invalid"},
		"agent absolute source":  {yaml: validProfileYAML() + "agentFiles:\n  oh-my-pi:\n    scout.md: /etc/passwd\n", code: "profile.agent_files_path_invalid"},
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

func TestParseProfileNamesFieldPathAndExpectedShape(t *testing.T) {
	t.Parallel()
	cases := map[string]struct {
		yaml    string
		path    string
		message string
	}{
		"labels list": {
			yaml:    "labels: [a, b]\n",
			path:    "labels",
			message: "expected a map of string keys to string values (e.g. labels: {team: core}), got a list",
		},
		"label value list": {
			yaml:    "labels:\n  team: [core]\n",
			path:    "labels.team",
			message: "expected a string value (e.g. labels: {team: core}), got a list",
		},
		"skills scalar": {
			yaml:    "skills: skills/review/SKILL.md\n",
			path:    "skills",
			message: "expected a list of skill paths (e.g. skills: [skills/review/SKILL.md]), got a scalar value",
		},
		"tools allow map": {
			yaml:    "tools:\n  allow: {read: true}\n",
			path:    "tools.allow",
			message: "expected a list of tool names (e.g. allow: [read, edit]), got a map",
		},
		"global file list": {
			yaml:    "globalInstructions:\n  codex: [AGENTS.md]\n",
			path:    "globalInstructions.codex",
			message: "expected a map of file names to resource paths (e.g. codex: {AGENTS.md: instructions/codex.md}), got a list",
		},
		"global resource map": {
			yaml:    "globalInstructions:\n  codex:\n    AGENTS.md: {path: a.md}\n",
			path:    "globalInstructions.codex.AGENTS.md",
			message: "expected a resource path string or a list of them, got a map",
		},
		"global fragment list": {
			yaml:    "globalInstructions:\n  codex:\n    AGENTS.md: [a.md, [b.md]]\n",
			path:    "globalInstructions.codex.AGENTS.md[1]",
			message: "expected a resource path string, got a list",
		},
	}
	for name, test := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			_, diagnostics := ParseProfile([]byte(test.yaml))
			errors := diagnostics.Errors()
			if len(errors) != 1 || errors[0].Code != "yaml.shape" || errors[0].Path != test.path || errors[0].Message != test.message {
				t.Fatalf("want one yaml.shape at %s %q, got %#v", test.path, test.message, errors)
			}
		})
	}
}

func TestResolveInheritanceAndDenyWins(t *testing.T) {
	t.Parallel()
	base := mustParse(t, `name: base
labels: {team: core, tier: base}
route: primary
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
	child := mustParse(t, `labels: {tier: research}
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
	missing := mustParse(t, strings.Replace(validProfileYAML(), "name: research", "name: child", 1)+"extends: absent\n")
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
		profile, diagnostics := ParseProfileAt(data, name)
		if len(diagnostics) > 0 {
			t.Fatal(diagnostics)
		}
		profiles[name] = profile
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
	manifest := Manifest{APIVersion: ManifestVersion, Kind: "Manifest", Owner: "profile-mango", Generation: 1, Profile: "research", Target: "codex"}
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
	return `name: research
route: research-primary
`
}

func profileWithParent(name, parent string) string {
	return "name: " + name + "\nextends: " + parent + "\n"
}
