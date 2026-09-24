package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"gitlab.com/ariel-frischer/profile-mango/pkg/install"
	"gitlab.com/ariel-frischer/profile-mango/pkg/profilemango"
)

func TestInstallOptionValidationTable(t *testing.T) {
	tests := map[string]struct {
		options installOptions
		wantErr string
	}{
		"requires-selection":   {options: installOptions{}, wantErr: "at least one"},
		"all-conflicts-target": {options: installOptions{all: true, targets: []string{"codex@0.154.0"}}, wantErr: "mutually exclusive"},
		"yes-needs-apply":      {options: installOptions{targets: []string{"codex@0.154.0"}, yes: true, expectPlan: "abc"}, wantErr: "--yes requires --apply"},
		"yes-needs-hash":       {options: installOptions{targets: []string{"codex@0.154.0"}, apply: true, yes: true}, wantErr: "--yes requires --expect-plan"},
		"json-apply-bound":     {options: installOptions{targets: []string{"codex@0.154.0"}, apply: true, jsonOutput: true}, wantErr: "JSON apply requires"},
		"plan-valid":           {options: installOptions{targets: []string{"codex@0.154.0"}}, wantErr: ""},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			err := validateInstallOptions(test.options)
			if test.wantErr == "" && err != nil {
				t.Fatal(err)
			}
			if test.wantErr != "" && (err == nil || !strings.Contains(err.Error(), test.wantErr)) {
				t.Fatalf("error = %v, want %q", err, test.wantErr)
			}
		})
	}
}

func TestInstallTargetAndConfigPathParsing(t *testing.T) {
	requests, err := installTargets(installOptions{
		targets: []string{"codex@0.154.0", "opencode@1.18.31"},
		configs: []string{"opencode=/synthetic/opencode.json", "codex=/synthetic/config.toml"},
	}, install.DefaultRegistry())
	if err != nil {
		t.Fatal(err)
	}
	if len(requests) != 2 || requests[0].Target.Name != "codex" || requests[1].ConfigPath != "/synthetic/opencode.json" {
		t.Fatalf("requests = %#v", requests)
	}
}

func TestInstallTargetSelectionTable(t *testing.T) {
	type want struct{ target, config, manifest, agent string }
	tests := map[string]struct {
		options installOptions
		want    []want
		wantErr string
	}{
		"bare target and bare config": {options: installOptions{targets: []string{"claude-code"}, configs: []string{"claude-code=/s/settings.json"}},
			want: []want{{target: "claude-code@2.1.278", config: "/s/settings.json"}}},
		"config alone implies target": {options: installOptions{configs: []string{"claude-code=/s/settings.json"}},
			want: []want{{target: "claude-code@2.1.278", config: "/s/settings.json"}}},
		"versioned config implies target": {options: installOptions{configs: []string{"codex@0.154.0=/s/config.toml"}},
			want: []want{{target: "codex@0.154.0", config: "/s/config.toml"}}},
		"deprecated config-path alias": {options: installOptions{legacyConfigs: []string{"pi=/s/pi.json"}},
			want: []want{{target: "pi@0.86.1", config: "/s/pi.json"}}},
		"target order then implied": {options: installOptions{targets: []string{"pi"}, configs: []string{"codex=/s/c", "pi=/s/p"}},
			want: []want{{target: "pi@0.86.1", config: "/s/p"}, {target: "codex@0.154.0", config: "/s/c"}}},
		"explicit unqualified version kept": {options: installOptions{targets: []string{"codex@0.153.0"}, configs: []string{"codex=/s/c"}},
			want: []want{{target: "codex@0.153.0", config: "/s/c"}}},
		"config-only unqualified version kept": {options: installOptions{configs: []string{"codex@0.153.0=/s/c"}},
			want: []want{{target: "codex@0.153.0", config: "/s/c"}}},
		"exact mapping wins over bare": {options: installOptions{targets: []string{"codex"}, configs: []string{"codex=/s/bare", "codex@0.154.0=/s/exact"}, manifests: []string{"codex=/s/m"}},
			want: []want{{target: "codex@0.154.0", config: "/s/exact", manifest: "/s/m"}}},
		"bare agent selector": {options: installOptions{agents: []string{"opencode=subagent:mango-review"}, configs: []string{"opencode=/s/agents/mango-review.md"}},
			want: []want{{target: "opencode@1.18.31", config: "/s/agents/mango-review.md", agent: "subagent:mango-review"}}},
		"config version mismatch":  {options: installOptions{targets: []string{"codex"}, configs: []string{"codex@0.153.0=/s/c"}}, wantErr: "--config target codex@0.153.0 does not match"},
		"manifest without target":  {options: installOptions{targets: []string{"codex"}, manifests: []string{"pi=/s/m"}}, wantErr: "--manifest target pi does not match"},
		"agent without target":     {options: installOptions{targets: []string{"opencode"}, agents: []string{"codex=primary:x"}}, wantErr: "--agent target codex does not match"},
		"all rejects unknown":      {options: installOptions{all: true, configs: []string{"nope=/s/c"}}, wantErr: "--config target nope does not match"},
		"duplicate resolved":       {options: installOptions{targets: []string{"codex", "codex@0.154.0"}}, wantErr: "duplicate target: codex@0.154.0"},
		"duplicate config":         {options: installOptions{targets: []string{"codex"}, configs: []string{"codex=/a"}, legacyConfigs: []string{"codex=/b"}}, wantErr: "duplicate --config mapping: codex"},
		"unknown bare target":      {options: installOptions{targets: []string{"ariel-jcode"}}, wantErr: "ariel-jcode has no qualified version"},
		"config missing separator": {options: installOptions{configs: []string{"/s/settings.json"}}, wantErr: "--config must use target[@version]=value"},
		"config empty version":     {options: installOptions{configs: []string{"codex@=/s/c"}}, wantErr: "target or target@version"},
		"config path as target":    {options: installOptions{configs: []string{"a/b=/s/c"}}, wantErr: "simple name"},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			requests, err := installTargets(test.options, install.DefaultRegistry())
			if test.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), test.wantErr) {
					t.Fatalf("error = %v, want %q", err, test.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			got := make([]want, 0, len(requests))
			for _, request := range requests {
				agent := ""
				if !request.Agent.Empty() {
					agent = request.Agent.Mode + ":" + request.Agent.Name
				}
				got = append(got, want{target: request.Target.String(), config: request.ConfigPath, manifest: request.ManifestPath, agent: agent})
			}
			if !slices.Equal(got, test.want) {
				t.Fatalf("requests = %+v, want %+v", got, test.want)
			}
		})
	}
}

func TestInstallCommandConfigFlagForms(t *testing.T) {
	root := t.TempDir()
	profiles, bindings := writeCodexInstallInputs(t, root)
	config := filepath.Join(root, "target", "config.toml")
	if err := os.MkdirAll(filepath.Dir(config), 0o755); err != nil {
		t.Fatal(err)
	}
	base := []string{"route-only", "--profiles", profiles, "--resource-root", root, "--bindings", bindings, "--json"}
	tests := map[string]struct {
		args       []string
		deprecated bool
	}{
		"bare target and config": {args: []string{"--target", "codex", "--config", "codex=" + config}},
		"config alone":           {args: []string{"--config", "codex=" + config}},
		"deprecated alias":       {args: []string{"--target", "codex@0.154.0", "--config-path", "codex=" + config}, deprecated: true},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			command := newInstallCmd()
			command.SetOut(&stdout)
			command.SetErr(&stderr)
			command.SetArgs(append(append([]string(nil), base...), test.args...))
			if err := command.Execute(); err != nil {
				t.Fatalf("install: %v; stdout=%s stderr=%s", err, stdout.String(), stderr.String())
			}
			if !strings.Contains(stdout.String(), `"version": "0.154.0"`) || !strings.Contains(stdout.String(), `"status": "ready"`) {
				t.Fatalf("unexpected plan: %s", stdout.String())
			}
			if got := strings.Contains(stdout.String()+stderr.String(), "use --config target[@version]=path instead"); got != test.deprecated {
				t.Fatalf("deprecation note = %v; stdout=%s stderr=%s", got, stdout.String(), stderr.String())
			}
		})
	}
}

func TestInstallHelpShowsOneConfigFlag(t *testing.T) {
	var output bytes.Buffer
	command := newInstallCmd()
	command.SetOut(&output)
	command.SetArgs([]string{"--help"})
	if err := command.Execute(); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(output.String(), "--config stringArray") || strings.Contains(output.String(), "--config-path") {
		t.Fatalf("help must show only --config: %s", output.String())
	}
}

func TestInstallNamedAgentSelection(t *testing.T) {
	options := installOptions{targets: []string{"opencode@1.18.31"}, agents: []string{"opencode@1.18.31=subagent:mango-review"}, configs: []string{"opencode@1.18.31=/synthetic/agents/mango-review.md"}}
	requests, err := installTargets(options, install.DefaultRegistry())
	if err != nil || len(requests) != 1 || requests[0].Agent != (install.AgentDestination{Mode: "subagent", Name: "mango-review"}) {
		t.Fatalf("requests = %#v, err = %v", requests, err)
	}
	for name, options := range map[string]installOptions{
		"missing target": {targets: []string{"opencode@1.18.31"}, agents: []string{"codex@0.154.0=primary:mango-review"}},
		"malformed":      {targets: []string{"opencode@1.18.31"}, agents: []string{"opencode@1.18.31=primary"}},
		"duplicate":      {targets: []string{"opencode@1.18.31"}, agents: []string{"opencode@1.18.31=primary:mango-review", "opencode@1.18.31=subagent:mango-review"}},
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := installTargets(options, install.DefaultRegistry()); err == nil {
				t.Fatal("invalid agent accepted")
			}
		})
	}
}

func TestInstallUnregisteredTargetBlockedWithoutTargetRead(t *testing.T) {
	root := t.TempDir()
	profiles, bindings := writeCodexInstallInputs(t, root)
	targetPath := filepath.Join(root, "must-not-be-read", "config.toml")
	var stdout, stderr bytes.Buffer
	cmd := &cobra.Command{}
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	err := runInstall(cmd, "route-only", installOptions{
		profiles: profiles, resourceRoot: root, bindings: bindings,
		targets: []string{"codex@0.153.0"}, configs: []string{"codex=" + targetPath},
		jsonOutput: true,
	})
	if err == nil || !strings.Contains(err.Error(), "blocked") {
		t.Fatalf("error = %v", err)
	}
	if !strings.Contains(stdout.String(), `"status": "blocked"`) || !strings.Contains(stdout.String(), "no static adapter is registered") {
		t.Fatalf("unexpected plan: %s", stdout.String())
	}
	if _, statErr := os.Stat(filepath.Dir(targetPath)); !os.IsNotExist(statErr) {
		t.Fatalf("blocked production plan touched target path: %v", statErr)
	}
}

func writeCodexInstallInputs(t *testing.T, root string) (string, string) {
	t.Helper()
	profiles := filepath.Join(root, "profiles")
	if err := os.MkdirAll(filepath.Join(profiles, "route-only"), 0o755); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(profiles, "route-only", "profile.yaml"), "apiVersion: profilemango.dev/v1alpha1\nkind: PolicyProfile\nmetadata:\n  name: route-only\nspec:\n  routeRef: route\n")
	bindings := filepath.Join(root, "bindings.yaml")
	writeFile(t, bindings, "routes:\n  route:\n    provider: openai\n    transport: native\n    authentication: oauth\n    model: gpt-5.6\n    effort: high\n")
	return profiles, bindings
}

func TestInstallCodexSettingsPlanWarnsWithoutWriting(t *testing.T) {
	root := t.TempDir()
	profiles, bindings := writeCodexInstallInputs(t, root)
	targetPath := filepath.Join(root, "target", "config.toml")
	if err := os.MkdirAll(filepath.Dir(targetPath), 0o755); err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer
	cmd := &cobra.Command{}
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	err := runInstall(cmd, "route-only", installOptions{
		profiles: profiles, resourceRoot: root, bindings: bindings,
		targets: []string{"codex@0.154.0"}, configs: []string{"codex=" + targetPath},
		jsonOutput: true,
	})
	if err != nil {
		t.Fatalf("Codex settings plan: %v; output=%s", err, stdout.String())
	}
	if !strings.Contains(stdout.String(), `"status": "ready"`) || !strings.Contains(stdout.String(), "authentication remains unmanaged") || !strings.Contains(stdout.String(), "codex login status") {
		t.Fatalf("unexpected plan: %s", stdout.String())
	}
	if _, statErr := os.Stat(targetPath); !os.IsNotExist(statErr) {
		t.Fatalf("read-only Codex plan touched target path: %v", statErr)
	}
}

func TestInstallHumanPlanDisplaysTargetWarning(t *testing.T) {
	var output bytes.Buffer
	command := &cobra.Command{}
	command.SetOut(&output)
	plan := install.Plan{PlanID: "synthetic", Status: install.StatusReady, Targets: []install.TargetPlan{{
		Target: install.Target{Name: "codex", Version: "0.154.0"}, Status: install.StatusReady,
		Diagnostics: profilemango.Diagnostics{{Severity: profilemango.SeverityWarning,
			Code: "codex.install.auth_unmanaged", Message: "authentication is unmanaged; codex login status does not prove exact OAuth"}},
	}}}
	if err := writeInstallPlan(command, plan, false); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(output.String(), "warning: authentication is unmanaged; codex login status does not prove exact OAuth") {
		t.Fatalf("human plan omitted required warning: %s", output.String())
	}
}

func TestInstallAllIncludesOpenCodeAndExcludesExperimentalJcode(t *testing.T) {
	registry := install.DefaultRegistry()
	requests := registry.Targets()
	joined := make([]string, 0, len(requests))
	for _, target := range requests {
		joined = append(joined, target.String())
	}
	text := strings.Join(joined, ",")
	if !strings.Contains(text, "opencode@1.18.31") || strings.Contains(text, "ariel-jcode") {
		t.Fatalf("production targets = %s", text)
	}
}

func TestInstallConsentRequiresTerminalOrHash(t *testing.T) {
	if terminalInput(strings.NewReader("y\n")) {
		t.Fatal("redirected input was accepted as terminal consent")
	}
	devNull, err := os.Open(os.DevNull)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = devNull.Close() }()
	if terminalInput(devNull) {
		t.Fatal("/dev/null was accepted as terminal consent")
	}
	options := installOptions{targets: []string{"codex@0.154.0"}, apply: true, yes: true, expectPlan: "plan"}
	if err := validateInstallOptions(options); err != nil {
		t.Fatal(err)
	}
	if got := options.expectPlanOrPlanID(install.Plan{PlanID: "other"}); got != "plan" {
		t.Fatalf("expected plan = %q", got)
	}
}

func TestInstallConfirmationAcceptsYesAndDefaultsNo(t *testing.T) {
	tests := map[string]struct {
		input  string
		wantOK bool
	}{
		"y":             {input: "y\n", wantOK: true},
		"yes":           {input: "YES\n", wantOK: true},
		"y without eof": {input: "y", wantOK: true},
		"n":             {input: "n\n"},
		"empty":         {input: "\n"},
		"eof":           {input: ""},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			var output bytes.Buffer
			err := confirmInstall(strings.NewReader(test.input), &output, "plan-123")
			if test.wantOK && err != nil {
				t.Fatalf("confirmInstall error = %v", err)
			}
			if !test.wantOK && (err == nil || !strings.Contains(err.Error(), "apply declined")) {
				t.Fatalf("confirmInstall error = %v, want apply declined", err)
			}
			if !strings.Contains(output.String(), "Apply plan plan-123? [y/N] ") {
				t.Fatalf("confirmation prompt = %q", output.String())
			}
		})
	}
}

func TestInstallConfirmationTreatsInputErrorAsDecline(t *testing.T) {
	var output bytes.Buffer
	err := confirmInstall(interruptedInput{}, &output, "plan-123")
	if err == nil || !strings.Contains(err.Error(), "apply declined") {
		t.Fatalf("confirmInstall error = %v, want default-no decline", err)
	}
}

func TestWriteInstallPlanHumanFieldDiffs(t *testing.T) {
	tests := map[string]struct {
		plan                 install.Plan
		want                 []string
		wantAbsent           []string
		preserveSensitiveRaw bool
	}{
		"target and file fields are deterministic and deduplicated": {
			plan: install.Plan{PlanID: "plan-123", Status: install.StatusReady, Targets: []install.TargetPlan{
				{Target: install.Target{Name: "zeta", Version: "1"}, Status: install.StatusReady, Fields: []install.FieldChange{
					{Path: "config.z", Before: "z", After: "Z"},
					{Path: "config.model", Before: "old", After: "new"},
				}, Files: []install.FilePlan{
					{Path: "z.json", Action: install.ActionUpdate, Fields: []install.FieldChange{{Path: "config.z"}, {Path: "resource.only"}}},
				}},
				{Target: install.Target{Name: "alpha", Version: "1"}, Status: install.StatusReady, Fields: []install.FieldChange{{Path: "config.model", Before: "old", After: "new"}}, Files: []install.FilePlan{
					{Path: "a.json", Action: install.ActionUpdate, Fields: []install.FieldChange{{Path: "config.model"}}},
				}},
			}},
			want: []string{
				"plan plan-123 (ready)\n",
				"  alpha@1: ready\n",
				"    field config.model: \"old\" -> \"new\"\n",
				"    a.json: update\n",
				"  zeta@1: ready\n",
				"    field config.model: \"old\" -> \"new\"\n",
				"    field config.z: \"z\" -> \"Z\"\n",
				"    z.json: update\n",
				"      field resource.only\n",
			},
			wantAbsent: []string{"      field config.model\n", "      field config.z\n"},
		},
		"sensitive values and controls are safe before normalization": {
			plan: install.Plan{PlanID: "plan-safe", Status: install.StatusReady, Targets: []install.TargetPlan{{
				Target: install.Target{Name: "safe", Version: "1"}, Status: install.StatusReady,
				Fields: []install.FieldChange{{Path: "config.\npath", Before: "SECRET-before", After: "after\tvalue\x1b[31m", Sensitive: true}},
				Files:  []install.FilePlan{{Path: "file\nname", Action: install.ActionUpdate}},
			}}},
			want:                 []string{"field \"config.\\npath\": \"<redacted>\" -> \"<redacted>\"\n", "\"file\\nname\": update\n"},
			wantAbsent:           []string{"SECRET-before", "after\tvalue", "\x1b[31m", "config.\npath", "file\nname"},
			preserveSensitiveRaw: true,
		},
		"normalized sensitive fields remain visible": {
			plan: install.Plan{PlanID: "plan-safe", Status: install.StatusReady, Targets: []install.TargetPlan{{
				Target: install.Target{Name: "safe", Version: "1"}, Status: install.StatusReady,
				Fields: []install.FieldChange{{Path: "config.secret", Before: "<redacted>", After: "<redacted>", Sensitive: true}},
			}}},
			want:       []string{"field config.secret\n"},
			wantAbsent: []string{" -> "},
		},
		"empty and noop fields do not invent changes": {
			plan: install.Plan{PlanID: "plan-noop", Status: install.StatusNoop, Targets: []install.TargetPlan{{
				Target: install.Target{Name: "same", Version: "1"}, Status: install.StatusNoop,
				Fields: []install.FieldChange{{Path: "config.same", Before: "same", After: "same"}, {Path: "config.empty"}},
				Files:  []install.FilePlan{{Path: "config.json", Action: install.ActionNoop, Fields: []install.FieldChange{{Path: "config.same"}, {Path: "config.empty"}}}},
			}}},
			want:       []string{"plan plan-noop (noop)\n", "  same@1: noop\n", "    config.json: noop\n"},
			wantAbsent: []string{"config.same:", "config.empty", " -> "},
		},
		"create fields show an empty before value": {
			plan: install.Plan{PlanID: "plan-create", Status: install.StatusReady, Targets: []install.TargetPlan{{
				Target: install.Target{Name: "new", Version: "1"}, Status: install.StatusReady,
				Fields: []install.FieldChange{{Path: "config.model", After: "new/model"}},
				Files:  []install.FilePlan{{Path: "config.json", Action: install.ActionCreate, Fields: []install.FieldChange{{Path: "config.model"}}}},
			}}},
			want: []string{"config.model: \"\" -> \"new/model\"\n", "config.json: create\n"},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			var output bytes.Buffer
			cmd := &cobra.Command{}
			cmd.SetOut(&output)
			if err := writeInstallPlan(cmd, test.plan, false); err != nil {
				t.Fatal(err)
			}
			got := output.String()
			for _, want := range test.want {
				if !strings.Contains(got, want) {
					t.Fatalf("output = %q, missing %q", got, want)
				}
			}
			for _, absent := range test.wantAbsent {
				if strings.Contains(got, absent) {
					t.Fatalf("output = %q, unexpectedly contains %q", got, absent)
				}
			}
			if test.preserveSensitiveRaw && test.plan.Targets[0].Fields[0].Before != "SECRET-before" {
				t.Fatalf("human rendering mutated sensitive field: %#v", test.plan.Targets[0].Fields[0])
			}
			if strings.Count(got, "      field config.model") != 0 {
				t.Fatalf("output duplicates target/file field: %q", got)
			}
		})
	}
}

func TestWriteInstallPlanHumanOutputIsDeterministic(t *testing.T) {
	plan := install.Plan{PlanID: "plan-repeat", Status: install.StatusReady, Targets: []install.TargetPlan{
		{Target: install.Target{Name: "b", Version: "1"}, Status: install.StatusReady, Fields: []install.FieldChange{{Path: "z", Before: "1", After: "2"}, {Path: "a", Before: "1", After: "2"}}},
		{Target: install.Target{Name: "a", Version: "1"}, Status: install.StatusReady, Fields: []install.FieldChange{{Path: "a", Before: "1", After: "2"}}},
	}}
	var first, second bytes.Buffer
	for _, output := range []*bytes.Buffer{&first, &second} {
		cmd := &cobra.Command{}
		cmd.SetOut(output)
		if err := writeInstallPlan(cmd, plan, false); err != nil {
			t.Fatal(err)
		}
	}
	if first.String() != second.String() {
		t.Fatalf("human output changed between runs:\nfirst:\n%s\nsecond:\n%s", first.String(), second.String())
	}
}

func TestInstallJSONApplyEmitsOneReportAndDoesNotReadStdin(t *testing.T) {
	options, _, _ := cliOpenCodeInstallFixture(t)
	plan := buildCLIInstallPlan(t, options)
	options.apply = true
	options.yes = true
	options.expectPlan = plan.PlanID
	options.jsonOutput = true
	options.nonInteractive = true

	var output bytes.Buffer
	input := &countingInput{}
	cmd := &cobra.Command{}
	cmd.SetIn(input)
	cmd.SetOut(&output)
	if err := runInstall(cmd, "route-only", options); err != nil {
		t.Fatal(err)
	}
	var report install.ApplyReport
	decoder := json.NewDecoder(bytes.NewReader(output.Bytes()))
	if err := decoder.Decode(&report); err != nil {
		t.Fatalf("decode apply report: %v\n%s", err, output.String())
	}
	if report.Status != "committed" {
		t.Fatalf("apply report = %#v", report)
	}
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		t.Fatalf("JSON apply emitted extra value, decode error = %v", err)
	}
	if input.reads != 0 {
		t.Fatalf("non-interactive apply read stdin %d time(s)", input.reads)
	}
}

func TestInstallRedirectedApplyFailsWithoutReadingStdin(t *testing.T) {
	options, _, _ := cliOpenCodeInstallFixture(t)
	options.apply = true
	options.jsonOutput = false
	input := &countingInput{}
	cmd := &cobra.Command{}
	cmd.SetIn(input)
	cmd.SetOut(&bytes.Buffer{})
	err := runInstall(cmd, "route-only", options)
	if err == nil || !strings.Contains(err.Error(), "requires a terminal") {
		t.Fatalf("runInstall error = %v, want terminal diagnostic", err)
	}
	if input.reads != 0 {
		t.Fatalf("redirected apply read stdin %d time(s)", input.reads)
	}
}

func TestInstallExpectedPlanRejectsChangedTargetBeforeWrite(t *testing.T) {
	options, config, _ := cliOpenCodeInstallFixture(t)
	plan := buildCLIInstallPlan(t, options)
	changed := `{ "model": "third-party/edit" }`
	writeFile(t, config, changed)

	options.apply = true
	options.yes = true
	options.expectPlan = plan.PlanID
	var output bytes.Buffer
	cmd := &cobra.Command{}
	cmd.SetOut(&output)
	err := runInstall(cmd, "route-only", options)
	if err == nil || !strings.Contains(err.Error(), "expected plan") {
		t.Fatalf("runInstall error = %v, want expected plan mismatch", err)
	}
	var report install.ApplyReport
	decoder := json.NewDecoder(bytes.NewReader(output.Bytes()))
	if decodeErr := decoder.Decode(&report); decodeErr != nil {
		t.Fatalf("decode failed apply report: %v\n%s", decodeErr, output.String())
	}
	if report.Status != install.StatusNotAttempted || len(report.Targets) != 1 {
		t.Fatalf("failed apply report = %#v", report)
	}
	if result := report.Targets[0]; result.Status != install.StatusNotAttempted || result.Error == "" {
		t.Fatalf("failed apply target report = %#v", result)
	}
	var extra any
	if decodeErr := decoder.Decode(&extra); decodeErr != io.EOF {
		t.Fatalf("failed apply emitted extra JSON value, decode error = %v", decodeErr)
	}
	data, readErr := os.ReadFile(config)
	if readErr != nil {
		t.Fatal(readErr)
	}
	if string(data) != changed {
		t.Fatalf("changed target was modified: %q", data)
	}
}

type countingInput struct {
	reads int
}

func (input *countingInput) Read([]byte) (int, error) {
	input.reads++
	return 0, io.EOF
}

type interruptedInput struct{}

func (interruptedInput) Read([]byte) (int, error) {
	return 0, errors.New("interrupted")
}

func cliOpenCodeInstallFixture(t *testing.T) (installOptions, string, string) {
	t.Helper()
	root := t.TempDir()
	profiles := filepath.Join(root, "profiles")
	if err := os.MkdirAll(filepath.Join(profiles, "route-only"), 0o755); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(profiles, "route-only", "profile.yaml"), "apiVersion: profilemango.dev/v1alpha1\nkind: PolicyProfile\nmetadata:\n  name: route-only\nspec:\n  routeRef: route\n")
	bindings := filepath.Join(root, "bindings.yaml")
	writeFile(t, bindings, "routes:\n  route:\n    provider: openai\n    transport: native\n    authentication: oauth\n    model: gpt-5.6\n    effort: high\n")
	config := filepath.Join(root, "target", "opencode.jsonc")
	if err := os.MkdirAll(filepath.Dir(config), 0o755); err != nil {
		t.Fatal(err)
	}
	before := "{\n  // keep\n  \"model\": \"old/model\",\n  \"unknown\": true,\n}\n"
	writeFile(t, config, before)
	return installOptions{
		profiles: profiles, resourceRoot: root, bindings: bindings,
		targets: []string{"opencode@1.18.31"}, configs: []string{"opencode=" + config},
		override: true, jsonOutput: true, makeDefault: true,
	}, config, before
}

func buildCLIInstallPlan(t *testing.T, options installOptions) install.Plan {
	t.Helper()
	var output bytes.Buffer
	cmd := &cobra.Command{}
	cmd.SetOut(&output)
	if err := runInstall(cmd, "route-only", options); err != nil {
		t.Fatal(err)
	}
	var plan install.Plan
	if err := json.Unmarshal(output.Bytes(), &plan); err != nil {
		t.Fatalf("decode install plan: %v\n%s", err, output.String())
	}
	if plan.Status != install.StatusReady {
		t.Fatalf("plan = %#v", plan)
	}
	return plan
}

func TestInstallOpenCodeModelAtExplicitDisposablePath(t *testing.T) {
	root := t.TempDir()
	profiles := filepath.Join(root, "profiles")
	if err := os.MkdirAll(filepath.Join(profiles, "route-only"), 0o755); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(profiles, "route-only", "profile.yaml"), "apiVersion: profilemango.dev/v1alpha1\nkind: PolicyProfile\nmetadata:\n  name: route-only\nspec:\n  routeRef: route\n")
	bindings := filepath.Join(root, "bindings.yaml")
	writeFile(t, bindings, "routes:\n  route:\n    provider: openai\n    transport: native\n    authentication: oauth\n    model: gpt-5.6\n    effort: high\n")
	config := filepath.Join(root, "target", "opencode.jsonc")
	if err := os.MkdirAll(filepath.Dir(config), 0o755); err != nil {
		t.Fatal(err)
	}
	before := "{\n  // keep\n  \"model\": \"old/model\",\n  \"unknown\": true,\n}\n"
	writeFile(t, config, before)
	options := installOptions{
		profiles: profiles, resourceRoot: root, bindings: bindings,
		targets: []string{"opencode@1.18.31"}, configs: []string{"opencode=" + config},
		override: true, jsonOutput: true, makeDefault: true,
	}
	var planOutput bytes.Buffer
	planCommand := &cobra.Command{}
	planCommand.SetOut(&planOutput)
	if err := runInstall(planCommand, "route-only", options); err != nil {
		t.Fatal(err)
	}
	var plan install.Plan
	if err := json.Unmarshal(planOutput.Bytes(), &plan); err != nil {
		t.Fatal(err)
	}
	if plan.Status != install.StatusReady || plan.PlanID == "" {
		t.Fatalf("plan = %#v", plan)
	}

	options.apply, options.yes, options.expectPlan = true, true, plan.PlanID
	applyCommand := &cobra.Command{}
	applyCommand.SetOut(&bytes.Buffer{})
	if err := runInstall(applyCommand, "route-only", options); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(config)
	if err != nil {
		t.Fatal(err)
	}
	want := strings.Replace(before, `"old/model"`, `"openai/gpt-5.6"`, 1)
	if string(data) != want {
		t.Fatalf("config = %q, want %q", data, want)
	}
}

func TestInstallApplyCommand(t *testing.T) {
	tests := map[string]struct {
		home    string
		options installOptions
		want    string
	}{
		"default inputs": {
			options: installOptions{targets: []string{"claude-code"}},
			want:    "profile-mango install default --target claude-code --apply --yes --expect-plan abc",
		},
		"every planning flag is carried and quoted": {
			home: "/tmp/my home",
			options: installOptions{
				profiles: "/p", resourceRoot: "/r", bindings: "/b.yaml",
				targets: []string{"opencode"}, agents: []string{"opencode=primary:mango"},
				configs: []string{"opencode=/c/agents/mango.md"}, legacyConfigs: []string{"codex=/x's.toml"},
				manifests: []string{"opencode=/m.json"}, noBackup: true, override: true,
			},
			want: "profile-mango --home '/tmp/my home' install default --profiles /p --resource-root /r --bindings /b.yaml --target opencode --agent opencode=primary:mango --config opencode=/c/agents/mango.md --config 'codex=/x'\\''s.toml' --manifest opencode=/m.json --no-backup --override --apply --yes --expect-plan abc",
		},
		"all targets": {
			options: installOptions{all: true},
			want:    "profile-mango install default --all --apply --yes --expect-plan abc",
		},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			homePathOverride = test.home
			t.Cleanup(func() { homePathOverride = "" })
			if got := installApplyCommand("default", test.options, "abc"); got != test.want {
				t.Fatalf("apply command:\n got %s\nwant %s", got, test.want)
			}
		})
	}
}
