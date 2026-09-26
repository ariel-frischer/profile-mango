package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"slices"
	"sort"
	"strconv"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/spf13/cobra"
	"gitlab.com/ariel-frischer/profile-mango/pkg/adapters/claudecode"
	"gitlab.com/ariel-frischer/profile-mango/pkg/adapters/codex"
	"gitlab.com/ariel-frischer/profile-mango/pkg/adapters/hermes"
	"gitlab.com/ariel-frischer/profile-mango/pkg/adapters/jcodefork"
	"gitlab.com/ariel-frischer/profile-mango/pkg/adapters/ohmypi"
	"gitlab.com/ariel-frischer/profile-mango/pkg/adapters/openclaw"
	"gitlab.com/ariel-frischer/profile-mango/pkg/adapters/opencode"
	"gitlab.com/ariel-frischer/profile-mango/pkg/adapters/pi"
	"gitlab.com/ariel-frischer/profile-mango/pkg/install"
)

const (
	doctorAPIVersion     = "profilemango.dev/doctor/v1alpha1"
	doctorKind           = "DoctorReport"
	doctorProbeCapBytes  = 4096
	defaultDoctorProfile = "default"
)

// doctorProbeTimeout bounds each "<bin> --version" probe. It is a var, not a
// const, so tests can shorten it with t.Cleanup instead of waiting out the
// real timeout.
var doctorProbeTimeout = 3 * time.Second

// doctorBinaryNames maps each registered target to the documented CLI
// executable name observed in its adapter evidence (docs/dev/agents/ and
// docs/dev/target-evidence.md), since it is not always the target name
// (Oh My Pi ships "omp"; the experimental Jcode fork ships "jcode").
var doctorBinaryNames = map[string]string{
	claudecode.TargetName: "claude",
	codex.TargetName:      "codex",
	hermes.TargetName:     "hermes",
	ohmypi.TargetName:     "omp",
	openclaw.TargetName:   "openclaw",
	opencode.TargetName:   "opencode",
	pi.TargetName:         "pi",
	jcodefork.TargetName:  "jcode",
}

type doctorOptions struct {
	profile      string
	targets      []string
	jsonOutput   bool
	experimental bool
}

type doctorTarget struct {
	Target           string `json:"target"`
	Binary           string `json:"binary"`
	BinaryFound      bool   `json:"binaryFound"`
	BinaryPath       string `json:"binaryPath,omitempty"`
	DetectedVersion  string `json:"detectedVersion,omitempty"`
	ProbeError       string `json:"probeError,omitempty"`
	QualifiedVersion string `json:"qualifiedVersion"`
	VersionMatch     bool   `json:"versionMatch"`
	CompatibleRange  string `json:"compatibleRange,omitempty"`
	InRange          bool   `json:"inRange"`
	ConfigPath       string `json:"configPath,omitempty"`
	ConfigBlocked    string `json:"configBlocked,omitempty"`
	ConfigExists     bool   `json:"configExists"`
	PlanStatus       string `json:"planStatus,omitempty"`
	PlanReason       string `json:"planReason,omitempty"`
}

type doctorReport struct {
	APIVersion string         `json:"apiVersion"`
	Kind       string         `json:"kind"`
	Profile    string         `json:"profile"`
	Targets    []doctorTarget `json:"targets"`
}

func newDoctorCmd() *cobra.Command {
	var options doctorOptions
	cmd := &cobra.Command{
		Use:   "doctor",
		Short: "Check each supported agent's command, version, and default settings path; nothing is written",
		Long: "doctor reports, for each supported target: whether its command is on PATH, the version it reports, " +
			"whether that version is inside the range mango was tested with, its default settings path and whether that path " +
			"exists, and whether installing the chosen profile would be ready or blocked. It never writes a file.",
		Args:         cobra.NoArgs,
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runDoctor(cmd, options)
		},
	}
	cmd.Flags().StringVar(&options.profile, "profile", "", "profile to check install readiness for (defaults to \"default\")")
	cmd.Flags().StringArrayVarP(&options.targets, "target", "t", nil, "target[@version], comma-separated or repeated; omit for all public agents")
	cmd.Flags().BoolVar(&options.jsonOutput, "json", false, "emit a stable JSON report")
	cmd.Flags().BoolVar(&options.experimental, "experimental", false, "also check the experimental Jcode fork")
	_ = cmd.Flags().MarkHidden("experimental")
	return cmd
}

func runDoctor(cmd *cobra.Command, options doctorOptions) error {
	registry := install.DefaultRegistry()
	targets, err := selectedDoctorTargets(registry, options)
	if err != nil {
		return err
	}
	profileName := options.profile
	if strings.TrimSpace(profileName) == "" {
		profileName = defaultDoctorProfile
	}
	report := doctorReport{APIVersion: doctorAPIVersion, Kind: doctorKind, Profile: profileName}
	planFor := doctorPlanner(registry, profileName)
	for index, target := range targets {
		_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "checking agent %d/%d (%s)\n", index+1, len(targets), target.Name)
		report.Targets = append(report.Targets, checkDoctorTarget(registry, target, planFor))
		_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "checked agent %d/%d (%s)\n", index+1, len(targets), target.Name)
	}
	if options.jsonOutput {
		return writeDoctorJSON(cmd, report)
	}
	return writeDoctorHuman(cmd, report)
}

func selectedDoctorTargets(registry *install.Registry, options doctorOptions) ([]install.Target, error) {
	available := doctorTargets(registry, options.experimental)
	selected, err := targetList(options.targets)
	if err != nil || len(selected) == 0 {
		return available, err
	}
	result := make([]install.Target, 0, len(selected))
	for _, value := range selected {
		selector, err := install.ParseTargetSelector(value)
		if err != nil {
			return nil, fmt.Errorf("resolve doctor target %q: %w", value, err)
		}
		found := false
		for _, target := range available {
			if selector.Matches(target) {
				if !slices.Contains(result, target) {
					result = append(result, target)
				}
				found = true
			}
		}
		if !found {
			return nil, fmt.Errorf("unknown or unqualified doctor target %q (experimental fork needs --experimental)", value)
		}
	}
	sort.Slice(result, func(i, j int) bool { return result[i].String() < result[j].String() })
	return result, nil
}

// doctorTargets lists every public registered target, sorted by name@version.
// Jcode fork has no install adapter at all (it is
// render-only), so it is added only when the hidden --experimental flag was
// given; its config-path and plan columns then report the resulting "no
// static adapter is registered" reason instead of crashing.
func doctorTargets(registry *install.Registry, experimental bool) []install.Target {
	result := append([]install.Target(nil), registry.Targets()...)
	if experimental {
		result = append(result, install.Target{Name: jcodefork.TargetName, Version: jcodefork.TargetVersion})
	}
	sort.Slice(result, func(i, j int) bool { return result[i].String() < result[j].String() })
	return result
}

func checkDoctorTarget(registry *install.Registry, target install.Target, planFor func(install.Target) (string, string)) doctorTarget {
	result := doctorTarget{Target: target.String(), QualifiedVersion: target.Version}
	detection := detectAgentVersion(target)
	result.Binary, result.BinaryFound = detection.Binary, detection.Found
	if detection.Found {
		result.BinaryPath = detection.Path
		result.DetectedVersion = detection.Output
		if detection.Err != nil {
			result.ProbeError = detection.Err.Error()
		}
		result.VersionMatch = detection.Output != "" && strings.Contains(detection.Output, target.Version)
	}
	metadata := install.AdapterMetadata{Version: target.Version}
	if adapter, found := registry.Lookup(target); found {
		metadata = adapter.Metadata()
	}
	if check, err := install.CheckVersion(metadata, detection); err == nil {
		result.CompatibleRange = check.Range
		result.InRange = check.Status == install.VersionInRange
	}
	env := install.OSPathEnv()
	configPath, err := registry.DefaultConfigPath(target, env)
	if err != nil {
		result.ConfigBlocked = err.Error()
	} else {
		result.ConfigPath = configPath
		result.ConfigExists = env.Exists(configPath)
	}
	result.PlanStatus, result.PlanReason = planFor(target)
	return result
}

// detectAgentVersion looks up a target's documented command on PATH and runs
// its bounded "--version" probe. Doctor and install share it.
func detectAgentVersion(target install.Target) install.VersionDetection {
	binary := doctorBinaryNames[target.Name]
	if binary == "" {
		binary = target.Name
	}
	path, found := lookupDoctorBinary(binary)
	if !found {
		return install.VersionDetection{Binary: binary}
	}
	output, err := probeDoctorVersion(path)
	return install.VersionDetection{Binary: binary, Path: path, Found: true, Output: output, Err: err}
}

func lookupDoctorBinary(name string) (string, bool) {
	path, err := exec.LookPath(name)
	if err != nil {
		return "", false
	}
	return path, true
}

// probeDoctorVersion runs "<path> --version" with a bounded timeout, a
// stripped-down environment that carries no credentials or proxy
// configuration, and capped stdout/stderr. Stdout comes first in the returned
// text so a stderr warning never shadows the version line. It never writes anything.
func probeDoctorVersion(path string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), doctorProbeTimeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, path, "--version")
	cmd.Env = doctorProbeEnv()
	stdout := &doctorCapBuffer{limit: doctorProbeCapBytes}
	stderr := &doctorCapBuffer{limit: doctorProbeCapBytes}
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	runErr := cmd.Run()
	text := joinProbeOutput(stdout.buf.String(), stderr.buf.String())
	if ctx.Err() == context.DeadlineExceeded {
		return text, fmt.Errorf("timed out after %s", doctorProbeTimeout)
	}
	if runErr != nil {
		return text, runErr
	}
	return text, nil
}

// joinProbeOutput puts trimmed stdout before stderr and caps the result.
func joinProbeOutput(stdout, stderr string) string {
	parts := make([]string, 0, 2)
	for _, part := range []string{stdout, stderr} {
		if trimmed := strings.TrimSpace(part); trimmed != "" {
			parts = append(parts, trimmed)
		}
	}
	text := strings.Join(parts, "\n")
	if len(text) > doctorProbeCapBytes {
		text = text[:doctorProbeCapBytes]
	}
	return text
}

// doctorProbeEnv keeps only the variables an agent CLI needs to start (PATH,
// a user-home variable, and scratch-directory locations) and drops
// credentials, tokens, and proxy configuration.
func doctorProbeEnv() []string {
	names := []string{"PATH", "HOME", "USERPROFILE", "TMPDIR", "TEMP", "TMP", "SystemRoot", "windir"}
	env := make([]string, 0, len(names))
	for _, name := range names {
		if value, ok := os.LookupEnv(name); ok {
			env = append(env, name+"="+value)
		}
	}
	return env
}

// doctorCapBuffer discards output past its limit instead of growing without
// bound, so a runaway or chatty agent binary cannot inflate the report.
type doctorCapBuffer struct {
	buf   bytes.Buffer
	limit int
}

func (w *doctorCapBuffer) Write(data []byte) (int, error) {
	if remaining := w.limit - w.buf.Len(); remaining > 0 {
		if len(data) > remaining {
			w.buf.Write(data[:remaining])
		} else {
			w.buf.Write(data)
		}
	}
	return len(data), nil
}

// doctorPlanner builds a target's install readiness check once, reusing the
// resolved profile home for every target. It plans with the same request a
// default "mango install --target <agent>" builds, so doctor and install share
// one readiness verdict. It only ever calls the read-only install.BuildPlan.
func doctorPlanner(registry *install.Registry, profileName string) func(install.Target) (string, string) {
	paths, err := resolveInstallPaths(installOptions{})
	if err != nil {
		reason := fmt.Sprintf("resolve profile home: %v", err)
		return func(install.Target) (string, string) { return install.StatusUnavailable, reason }
	}
	return func(target install.Target) (string, string) {
		request := installPlanRequest(profileName, paths, []install.TargetRequest{{Target: target}}, installOptions{}, registry)
		plan, err := install.BuildPlan(request)
		if err != nil {
			return install.StatusUnavailable, err.Error()
		}
		if len(plan.Targets) == 0 {
			return install.StatusUnavailable, "plan produced no target result"
		}
		return plan.Targets[0].Status, plan.Targets[0].Reason
	}
}

func writeDoctorJSON(cmd *cobra.Command, report doctorReport) error {
	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return fmt.Errorf("encode doctor report: %w", err)
	}
	_, err = cmd.OutOrStdout().Write(append(data, '\n'))
	return err
}

func writeDoctorHuman(cmd *cobra.Command, report doctorReport) error {
	output := cmd.OutOrStdout()
	table := tabwriter.NewWriter(output, 0, 2, 2, ' ', 0)
	if _, err := fmt.Fprintln(table, "TARGET\tBINARY\tDETECTED\tIN RANGE\tCONFIG\tPLAN"); err != nil {
		return err
	}
	for _, target := range report.Targets {
		if err := writeDoctorRow(table, target); err != nil {
			return err
		}
	}
	if err := table.Flush(); err != nil {
		return fmt.Errorf("write doctor table: %w", err)
	}
	suggestion := doctorSuggestion(report.Profile, report.Targets)
	if suggestion == "" {
		return nil
	}
	_, err := fmt.Fprintln(output, suggestion)
	return err
}

func writeDoctorRow(table io.Writer, target doctorTarget) error {
	_, err := fmt.Fprintf(table, "%s\t%s\t%s\t%s\t%s\t%s\n",
		target.Target,
		doctorBinaryCell(target),
		doctorVersionCell(target),
		doctorRangeCell(target),
		doctorConfigCell(target),
		doctorPlanCell(target),
	)
	return err
}

func doctorBinaryCell(target doctorTarget) string {
	if !target.BinaryFound {
		return target.Binary + " (not found)"
	}
	return target.Binary
}

func doctorVersionCell(target doctorTarget) string {
	switch {
	case !target.BinaryFound:
		return "-"
	case target.ProbeError != "":
		return "error: " + target.ProbeError
	case target.DetectedVersion == "":
		return "(empty)"
	default:
		// The table is one row per target; show only the line install's version
		// check parses (or the first line) of a chatty multi-line "--version"
		// probe. The full text stays in the JSON report's detectedVersion field.
		lines := strings.Split(target.DetectedVersion, "\n")
		line := doctorVersionLine(lines)
		if len(lines) > 1 {
			return line + " (+" + strconv.Itoa(len(lines)-1) + " more line(s))"
		}
		return line
	}
}

// doctorVersionLine returns the first line holding a major.minor.patch
// version, the same number install.ParseVersion extracts, else the first line.
func doctorVersionLine(lines []string) string {
	for _, line := range lines {
		if _, ok := install.ParseVersion(line); ok {
			return line
		}
	}
	return lines[0]
}

// doctorRangeCell reports whether the detected version lies in the tested
// range: "yes", "no" (with the range), or "-" when nothing was detected.
func doctorRangeCell(target doctorTarget) string {
	if !target.BinaryFound {
		return "-"
	}
	if target.InRange {
		return "yes"
	}
	if target.CompatibleRange == "" {
		return "no"
	}
	return "no (tested " + target.CompatibleRange + ")"
}

func doctorConfigCell(target doctorTarget) string {
	if target.ConfigBlocked != "" {
		return "blocked"
	}
	state := "missing"
	if target.ConfigExists {
		state = "exists"
	}
	return fmt.Sprintf("%s (%s)", target.ConfigPath, state)
}

func doctorPlanCell(target doctorTarget) string {
	if target.PlanStatus == "" {
		return "-"
	}
	return target.PlanStatus
}

// doctorSuggestion names one next command: install the first target whose
// plan is already ready, or point at "init" when no profile home was found,
// or fall back to showing a plan for the first target otherwise.
func doctorSuggestion(profileName string, targets []doctorTarget) string {
	for _, target := range targets {
		if target.PlanStatus == install.StatusReady || target.PlanStatus == install.StatusNoop {
			return fmt.Sprintf("next: mango install %s --target %s", profileName, doctorBareTargetName(target.Target))
		}
	}
	for _, target := range targets {
		if target.PlanStatus == install.StatusUnavailable {
			return "next: run mango init to set up a profile home, then rerun doctor"
		}
	}
	if len(targets) == 0 {
		return ""
	}
	return fmt.Sprintf("next: mango install %s --target %s to see the full plan", profileName, doctorBareTargetName(targets[0].Target))
}

func doctorBareTargetName(target string) string {
	name, _, _ := strings.Cut(target, "@")
	return name
}
