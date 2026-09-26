package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strconv"
	"strings"
	"unicode"

	"github.com/ariel-frischer/profile-mango/pkg/adapters/ohmypi"
	"github.com/ariel-frischer/profile-mango/pkg/install"
	"github.com/ariel-frischer/profile-mango/pkg/profilemango"
	"github.com/mattn/go-isatty"
	"github.com/spf13/cobra"
)

type installOptions struct {
	profiles       string
	resourceRoot   string
	bindings       string
	targets        []string
	agents         []string
	configs        []string
	manifests      []string
	legacyConfigs  []string
	all            bool
	apply          bool
	yes            bool
	expectPlan     string
	noBackup       bool
	override       bool
	strict         bool
	makeDefault    bool
	jsonOutput     bool
	verbose        bool
	nonInteractive bool
	// use (mango use) makes the profile each target's default and releases files it no longer writes.
	use bool
}

func newInstallCmd() *cobra.Command {
	var options installOptions
	cmd := &cobra.Command{
		Use:          "install <profile>",
		Aliases:      []string{"i"},
		Short:        "Plan installing the supported settings from a profile into a target; nothing is written until you pass --apply",
		Args:         cobra.ExactArgs(1),
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			options.nonInteractive = nonInteractive
			return runInstall(cmd, args[0], options)
		},
	}
	cmd.Flags().StringVar(&options.profiles, "profiles", "", "profile repository root (defaults to <home>/profiles)")
	cmd.Flags().StringVar(&options.resourceRoot, "resource-root", "", "resource package root (defaults to <home>)")
	cmd.Flags().StringVar(&options.bindings, "bindings", "", "local route bindings file (defaults to <home>/bindings/local.yaml)")
	cmd.Flags().StringArrayVarP(&options.targets, "target", "t", nil, "target[@version], comma-separated or repeated; a bare name selects its single qualified version")
	cmd.Flags().StringArrayVar(&options.agents, "agent", nil, "target[@version]=primary:name or subagent:name; requires --config ending agents/name.md")
	cmd.Flags().StringArrayVar(&options.configs, "config", nil, "target[@version]=explicit config path; selects the target; repeat for multiple targets")
	cmd.Flags().StringArrayVar(&options.legacyConfigs, "config-path", nil, "deprecated alias for --config")
	_ = cmd.Flags().MarkDeprecated("config-path", "use --config target[@version]=path instead")
	cmd.Flags().StringArrayVar(&options.manifests, "manifest", nil, "target[@version]=explicit ownership manifest path")
	cmd.Flags().BoolVar(&options.all, "all", false, "plan every supported agent that is installed; others are listed as skipped")
	cmd.Flags().BoolVar(&options.apply, "apply", false, "apply the already displayed plan after consent")
	cmd.Flags().BoolVar(&options.yes, "yes", false, "confirm non-interactive apply; requires --expect-plan")
	cmd.Flags().StringVar(&options.expectPlan, "expect-plan", "", "expected plan ID required for --yes and apply")
	cmd.Flags().BoolVar(&options.noBackup, "no-backup", false, "disable create-only pre-apply backups")
	cmd.Flags().BoolVar(&options.override, "override", false, "allow overwriting settings that were changed outside profile-mango, where the target supports it")
	cmd.Flags().BoolVar(&options.strict, "strict", false, "stop instead of skipping profile settings an agent cannot install")
	cmd.Flags().BoolVar(&options.makeDefault, "default", false, "also make this profile the agent's default, used without choosing a profile")
	cmd.Flags().BoolVar(&options.jsonOutput, "json", false, "emit the deterministic plan as JSON")
	cmd.Flags().BoolVarP(&options.verbose, "verbose", "v", false, "show full plan details, field paths, and diagnostics")
	return cmd
}

// installVersionDetector reports each ready target's installed agent version.
// Tests replace it so they never run real agent commands.
var installVersionDetector install.VersionDetector = detectAgentVersion

func runInstall(cmd *cobra.Command, profile string, options installOptions, registries ...*install.Registry) error {
	if err := validateInstallOptions(options); err != nil {
		return err
	}
	paths, err := resolveInstallPaths(options)
	if err != nil {
		return err
	}
	registry := install.DefaultRegistry()
	if len(registries) > 0 && registries[0] != nil {
		registry = registries[0]
	}
	targets, err := commandTargets(options, registry)
	if err != nil {
		return err
	}
	request := installPlanRequest(profile, paths, targets, options, registry)
	request.DetectVersion = versionDetectorWithProgress(cmd.ErrOrStderr(), installVersionDetector)
	plan, err := install.BuildPlan(request)
	if err != nil {
		return err
	}
	writePlan := !options.apply || !options.jsonOutput || plan.Status == install.StatusBlocked
	if writePlan {
		if err := writeInstallPlan(cmd, plan, options.jsonOutput, options.verbose); err != nil {
			return err
		}
	}
	if noAgentsFound(plan) {
		return fmt.Errorf("install plan is blocked: no supported agents found; run mango doctor")
	}
	if !options.apply {
		if plan.Status == install.StatusBlocked {
			return fmt.Errorf("install plan is blocked")
		}
		return writeApplyHint(cmd.OutOrStdout(), profile, options, plan)
	}
	if plan.Status == install.StatusBlocked {
		return fmt.Errorf("install plan is blocked; no files were written")
	}
	if err := authorizeInstall(cmd, options, plan); err != nil {
		return err
	}
	report, applyErr := install.ApplyPlan(plan, install.ApplyOptions{ExpectedPlanID: options.expectPlanOrPlanID(plan)})
	if applyErr != nil {
		if err := writeApplyReport(cmd, report, options.jsonOutput); err != nil {
			return err
		}
		return applyErr
	}
	return writeApplyReport(cmd, report, options.jsonOutput)
}

// authorizeInstall requires --yes --expect-plan or terminal consent before a plan
// writes. An all-unchanged plan writes nothing, so it needs no consent.
func authorizeInstall(cmd *cobra.Command, options installOptions, plan install.Plan) error {
	switch {
	case options.yes:
		if options.expectPlan == "" {
			return fmt.Errorf("non-interactive apply requires --expect-plan")
		}
		return nil
	case plan.Status == install.StatusNoop:
		return nil
	case !terminalInput(cmd.InOrStdin()):
		return fmt.Errorf("interactive apply requires a terminal; use --yes --expect-plan")
	}
	return confirmInstall(cmd.InOrStdin(), cmd.OutOrStdout(), plan.PlanID)
}

// installPlanRequest builds the plan request for one set of install flags.
// Doctor calls it with zero-value options so its readiness column is exactly
// what a plain "mango install --target <agent>" would plan.
func installPlanRequest(profile string, paths installPaths, targets []install.TargetRequest, options installOptions, registry *install.Registry) install.Request {
	return install.Request{
		ProfileName: profile, ProfilesRoot: paths.profiles, ResourceRoot: paths.resourceRoot,
		BindingsPath: paths.bindings, Targets: targets, All: false,
		Backup: !options.noBackup, Override: options.override, Strict: options.strict, Default: options.makeDefault || options.use, Registry: registry,
		Env: install.OSPathEnv(), SkipNotInstalled: options.all, Release: options.use,
	}
}

func versionDetectorWithProgress(w io.Writer, detector install.VersionDetector) install.VersionDetector {
	if detector == nil {
		return nil
	}
	return func(target install.Target) install.VersionDetection {
		_, _ = fmt.Fprintf(w, "checking agent version (%s)\n", target.Name)
		detection := detector(target)
		_, _ = fmt.Fprintf(w, "checked agent version (%s)\n", target.Name)
		return detection
	}
}

// noAgentsFound reports an --all plan whose every target was skipped as not installed.
func noAgentsFound(plan install.Plan) bool {
	for _, target := range plan.Targets {
		if target.Status != install.StatusSkipped {
			return false
		}
	}
	return len(plan.Targets) > 0
}

func validateInstallOptions(options installOptions) error {
	if !options.all && !options.use && len(options.targets) == 0 && len(options.configValues()) == 0 {
		return fmt.Errorf("at least one --target, --config target=path, or --all is required")
	}
	if options.all && len(options.targets) > 0 {
		return fmt.Errorf("--all and --target are mutually exclusive")
	}
	if options.all && len(options.agents) > 0 {
		return fmt.Errorf("--all and --agent are mutually exclusive")
	}
	if options.yes && !options.apply {
		return fmt.Errorf("--yes requires --apply")
	}
	if options.nonInteractive && options.apply && !options.yes {
		return fmt.Errorf("--non-interactive apply requires --yes --expect-plan")
	}
	if options.expectPlan != "" && (!options.apply || !options.yes) {
		return fmt.Errorf("--expect-plan requires --apply --yes")
	}
	if options.yes && options.expectPlan == "" {
		return fmt.Errorf("--yes requires --expect-plan")
	}
	if options.apply && options.jsonOutput && (!options.yes || options.expectPlan == "") {
		return fmt.Errorf("JSON apply requires --yes and --expect-plan")
	}
	return nil
}

type installPaths struct {
	profiles, resourceRoot, bindings string
}

func resolveInstallPaths(options installOptions) (installPaths, error) {
	values := []string{options.profiles, options.resourceRoot, options.bindings}
	explicit := 0
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			explicit++
		}
	}
	if explicit != 0 && explicit != len(values) {
		return installPaths{}, fmt.Errorf("set --profiles, --resource-root, and --bindings together, or omit all three to use the profile home")
	}
	if explicit == len(values) {
		return installPaths{profiles: options.profiles, resourceRoot: options.resourceRoot, bindings: options.bindings}, nil
	}
	home, err := selectedHome()
	if err != nil {
		return installPaths{}, err
	}
	return installPaths{profiles: filepath.Join(home, "profiles"), resourceRoot: home, bindings: filepath.Join(home, "bindings", "local.yaml")}, nil
}

func (options installOptions) configValues() []string {
	return append(append([]string(nil), options.configs...), options.legacyConfigs...)
}

// targetBinding is one target[@version]=value flag entry; an empty key version matches any version.
type targetBinding struct {
	flag, value string
	key         install.Target
}

func installTargets(options installOptions, registry *install.Registry) ([]install.TargetRequest, error) {
	configs, err := parseTargetBindings("--config", options.configValues())
	if err != nil {
		return nil, err
	}
	manifests, err := parseTargetBindings("--manifest", options.manifests)
	if err != nil {
		return nil, err
	}
	agents, err := parseAgentSelections(options.agents)
	if err != nil {
		return nil, err
	}
	targets, err := selectInstallTargets(options, registry, configs)
	if err != nil {
		return nil, err
	}
	for _, bindings := range [][]targetBinding{configs, manifests, agents} {
		if err := requireBindingsSelected(bindings, targets); err != nil {
			return nil, err
		}
	}
	result := make([]install.TargetRequest, 0, len(targets))
	for _, target := range targets {
		request := install.TargetRequest{Target: target, ConfigPath: bindingValue(configs, target), ManifestPath: bindingValue(manifests, target)}
		if spec := bindingValue(agents, target); spec != "" {
			request.Agent, _ = parseAgentSpec(spec)
		}
		result = append(result, request)
	}
	return result, nil
}

// selectInstallTargets resolves --target values, then adds any target implied by a --config entry.
func selectInstallTargets(options installOptions, registry *install.Registry, configs []targetBinding) ([]install.Target, error) {
	if options.all {
		if len(options.agents) > 0 {
			return nil, fmt.Errorf("--all and --agent are mutually exclusive")
		}
		return registry.Targets(), nil
	}
	values, err := targetList(options.targets)
	if err != nil {
		return nil, err
	}
	result := make([]install.Target, 0, len(values)+len(configs))
	for _, value := range values {
		target, err := resolveInstallTarget(registry, value)
		if err != nil {
			return nil, err
		}
		if !slices.Contains(result, target) {
			result = append(result, target)
		}
	}
	for _, config := range configs {
		if selectsName(result, config.key.Name) {
			continue
		}
		target, err := resolveInstallTarget(registry, bindingKey(config.key))
		if err != nil {
			return nil, err
		}
		result = append(result, target)
	}
	if len(result) == 0 {
		return nil, fmt.Errorf("provide --target, --config target=path, or --all")
	}
	return result, nil
}

func resolveInstallTarget(registry *install.Registry, value string) (install.Target, error) {
	target, err := registry.ResolveTarget(value)
	if err != nil {
		return install.Target{}, fmt.Errorf("resolve install target %q: %w", value, err)
	}
	return target, nil
}

func selectsName(targets []install.Target, name string) bool {
	for _, target := range targets {
		if target.Name == name {
			return true
		}
	}
	return false
}

func parseTargetBindings(flag string, values []string) ([]targetBinding, error) {
	result := make([]targetBinding, 0, len(values))
	seen := map[string]struct{}{}
	for _, value := range values {
		key, path, found := strings.Cut(value, "=")
		if !found || strings.TrimSpace(path) == "" {
			return nil, fmt.Errorf("%s must use target[@version]=value syntax", flag)
		}
		target, err := install.ParseTargetSelector(key)
		if err != nil {
			return nil, fmt.Errorf("parse %s %q: %w", flag, key, err)
		}
		if _, duplicate := seen[target.String()]; duplicate {
			return nil, fmt.Errorf("duplicate %s mapping: %s", flag, strings.TrimSpace(key))
		}
		seen[target.String()] = struct{}{}
		result = append(result, targetBinding{flag: flag, key: target, value: path})
	}
	return result, nil
}

func parseAgentSelections(values []string) ([]targetBinding, error) {
	bindings, err := parseTargetBindings("--agent", values)
	if err != nil {
		return nil, fmt.Errorf("--agent requires target[@version]=primary:name or subagent:name: %w", err)
	}
	for _, binding := range bindings {
		if _, err := parseAgentSpec(binding.value); err != nil {
			return nil, err
		}
	}
	return bindings, nil
}

func parseAgentSpec(spec string) (install.AgentDestination, error) {
	mode, name, found := strings.Cut(spec, ":")
	if !found || mode == "" || name == "" {
		return install.AgentDestination{}, fmt.Errorf("--agent requires target[@version]=primary:name or subagent:name")
	}
	return install.AgentDestination{Mode: mode, Name: name}, nil
}

func requireBindingsSelected(bindings []targetBinding, targets []install.Target) error {
	for _, binding := range bindings {
		if bindingSelected(binding, targets) {
			continue
		}
		return fmt.Errorf("%s target %s does not match any selected target", binding.flag, bindingKey(binding.key))
	}
	return nil
}

func bindingSelected(binding targetBinding, targets []install.Target) bool {
	for _, target := range targets {
		if binding.key.Matches(target) {
			return true
		}
	}
	return false
}

// bindingValue prefers an exact target@version entry over a bare-name entry.
func bindingValue(bindings []targetBinding, target install.Target) string {
	fallback := ""
	for _, binding := range bindings {
		if binding.key == target {
			return binding.value
		}
		if binding.key.Version == "" && binding.key.Name == target.Name {
			fallback = binding.value
		}
	}
	return fallback
}

func bindingKey(target install.Target) string {
	if target.Version == "" {
		return target.Name
	}
	return target.String()
}

func writeInstallPlan(cmd *cobra.Command, plan install.Plan, jsonOutput, verbose bool) error {
	if jsonOutput {
		data, err := plan.JSON()
		if err != nil {
			return err
		}
		_, err = cmd.OutOrStdout().Write(data)
		return err
	}
	if verbose {
		return writeHumanInstallPlan(cmd.OutOrStdout(), plan)
	}
	return writeCompactInstallPlan(cmd.OutOrStdout(), plan)
}

func writeCompactInstallPlan(output io.Writer, plan install.Plan) error {
	styles := stylesFor(output, true)
	if _, err := fmt.Fprintf(output, "%s %s (%s)\n", styles.heading("plan"), humanPath(plan.PlanID), styledPlanStatus(plan.Status, styles)); err != nil {
		return err
	}
	counts := map[string]int{}
	files := map[string]int{}
	for _, target := range orderedTargetPlans(plan.Targets) {
		counts[target.Status]++
		for _, file := range target.Files {
			files[file.Action]++
		}
		if err := writeCompactTarget(output, target, styles); err != nil {
			return err
		}
	}
	extra := ""
	for _, action := range []string{install.ActionDelete, install.ActionRestore} {
		if files[action] > 0 {
			extra += fmt.Sprintf(", %d %s", files[action], action)
		}
	}
	_, err := fmt.Fprintf(output, "Summary: %d ready, %d unchanged, %d blocked, %d conflict, %d skipped; files: %d create, %d update, %d unchanged%s. Unrelated target settings are preserved.\n", counts[install.StatusReady], counts[install.StatusNoop], counts[install.StatusBlocked], counts[install.StatusConflict], counts[install.StatusSkipped], files[install.ActionCreate], files[install.ActionUpdate], files[install.ActionNoop], extra)
	return err
}

func styledPlanStatus(status string, styles outputStyles) string {
	switch status {
	case install.StatusReady, "committed":
		return styles.success(status)
	case install.StatusBlocked, install.StatusConflict, "failed":
		return styles.failure(status)
	case install.StatusSkipped:
		return styles.warning(status)
	default:
		return styles.dim(status)
	}
}

func writeCompactTarget(output io.Writer, target install.TargetPlan, styles outputStyles) error {
	if _, err := fmt.Fprintf(output, "  %s: %s", styles.label(humanPath(target.Target.String())), styledPlanStatus(target.Status, styles)); err != nil {
		return err
	}
	if target.Reason != "" && (target.Status == install.StatusBlocked || target.Status == install.StatusConflict || target.Status == install.StatusSkipped) {
		if _, err := fmt.Fprintf(output, " (%s)", humanPath(target.Reason)); err != nil {
			return err
		}
	}
	if target.Config != nil && target.Status != install.StatusSkipped {
		if _, err := fmt.Fprintf(output, " | destination: %s", styles.path(humanPath(primaryInstallPath(target)))); err != nil {
			return err
		}
		if target.Install != nil && target.Install.SetsDefault {
			if _, err := fmt.Fprintf(output, " | also the default: %s", styles.path(humanPath(target.Config.Path))); err != nil {
				return err
			}
		}
	}
	if command := compactUseCommand(target); command != "" && (target.Status == install.StatusReady || target.Status == install.StatusNoop) {
		if _, err := fmt.Fprintf(output, " | use it: %s", humanPath(command)); err != nil {
			return err
		}
	} else if target.Install != nil && target.Install.Mode == install.InstallModeDefaultConfig && (target.Status == install.StatusReady || target.Status == install.StatusNoop) {
		if _, err := fmt.Fprint(output, " | default settings"); err != nil {
			return err
		}
	}
	if _, err := fmt.Fprintln(output); err != nil {
		return err
	}
	return writeCompactEffects(output, target, styles)
}

func compactUseCommand(target install.TargetPlan) string {
	if target.Install == nil || target.Install.Mode != install.InstallModeNamedProfile {
		return ""
	}
	return target.Install.UseCommand
}

// primaryInstallPath asks the qualified adapter where its named profile lives.
// This only calls the path resolver, never target inspection or I/O.
func primaryInstallPath(target install.TargetPlan) string {
	config := target.Config.Path
	if target.Install == nil || target.Install.Mode != install.InstallModeNamedProfile {
		return config
	}
	adapter, ok := install.DefaultRegistry().Lookup(target.Target)
	if !ok {
		return config
	}
	if resolver, ok := adapter.(interface {
		NamedProfilePath(string, string) (string, error)
	}); ok {
		if path, err := resolver.NamedProfilePath(config, target.Install.ProfileName); err == nil {
			return path
		}
	}
	if resolver, ok := adapter.(interface{ NamedProfileFile(string) (string, error) }); ok {
		if path, err := resolver.NamedProfileFile(target.Install.ProfileName); err == nil {
			if filepath.IsAbs(path) {
				return path
			}
			return filepath.Join(filepath.Dir(config), path)
		}
	}
	return config
}

func writeCompactRoute(output io.Writer, target install.TargetPlan) error {
	var model, effort string
	for _, field := range target.Fields {
		if field.Sensitive {
			continue
		}
		path := field.Path
		if target.Install != nil && target.Install.ProfileName != "" {
			path = strings.TrimPrefix(path, target.Install.ProfileName+".")
		}
		switch path {
		case "config.modelRoles.default", "profile.modelRoles.default":
			model, effort = ohmypi.SplitRoleSelector(field.After)
		case "profile.model", "agent.model", "config.model", "config.model.default", "config.defaultModel", "config.agents.defaults.model.primary":
			model = field.After
		case "config.model_reasoning_effort", "config.agent.reasoning_effort", "config.defaultThinkingLevel", "config.agents.defaults.thinkingDefault", "profile.effortLevel", "config.effortLevel", "agent.variant":
			effort = field.After
		}
	}
	if model == "" && effort == "" {
		return nil
	}
	if skipped, found := skippedEffort(target.SkippedRequirements); found && effort == "" {
		effort = skipped.Value + " NOT APPLIED"
	}
	_, err := fmt.Fprintf(output, "    route: model %s, effort %s\n", humanPath(orUnknown(model)), humanPath(orUnknown(effort)))
	return err
}

// skippedEffort is the base route's unapplied effort; role subagent efforts are listed apart.
func skippedEffort(skipped []install.SkippedRequirement) (install.SkippedRequirement, bool) {
	for _, requirement := range skipped {
		if requirement.Requirement == install.RequirementEffort && requirement.Role == "" {
			return requirement, true
		}
	}
	return install.SkippedRequirement{}, false
}

func orUnknown(value string) string {
	if value == "" {
		return "not installed"
	}
	return value
}

func writeCompactEffects(output io.Writer, target install.TargetPlan, styles outputStyles) error {
	seen := make(map[string]struct{})
	effects := compactFieldEffects(target.Fields, seen, target.Status != install.StatusNoop)
	var files []string
	for _, file := range orderedFilePlans(target.Files) {
		files = append(files, styles.path(humanPath(file.Path))+" "+humanPath(file.Action))
		effects = append(effects, compactFieldEffects(file.Fields, seen, file.Action != install.ActionNoop)...)
	}
	var route strings.Builder
	if err := writeCompactRoute(&route, target); err != nil {
		return err
	}
	parts := make([]string, 0, 2)
	if route.Len() > 0 {
		parts = append(parts, strings.TrimSpace(route.String()))
	}
	if len(effects) > 0 {
		parts = append(parts, "changes: "+strings.Join(effects, ", "))
	}
	if len(parts) > 0 {
		if _, err := fmt.Fprintf(output, "    %s\n", strings.Join(parts, " | ")); err != nil {
			return err
		}
	}
	if len(files) > 0 {
		if _, err := fmt.Fprintf(output, "    files: %s\n", strings.Join(files, ", ")); err != nil {
			return err
		}
	}
	if err := writeCompactSkipped(output, target.SkippedRequirements, styles); err != nil {
		return err
	}
	return writeCompactWarnings(output, target, styles)
}

func writeCompactSkipped(output io.Writer, skipped []install.SkippedRequirement, styles outputStyles) error {
	if len(skipped) == 0 {
		return nil
	}
	var line strings.Builder
	if err := writeSkippedRequirements(&line, skipped); err != nil {
		return err
	}
	text := strings.Replace(line.String(), "not installed for this agent", styles.warning("not installed for this agent"), 1)
	_, err := fmt.Fprint(output, strings.Replace(text, "NOT APPLIED", styles.warning("NOT APPLIED"), 1))
	return err
}

func compactFieldEffects(fields []install.FieldChange, seen map[string]struct{}, showEmpty bool) []string {
	var effects []string
	for _, field := range orderedFieldChanges(fields) {
		if field.Path == "" {
			continue
		}
		if _, exists := seen[field.Path]; exists {
			continue
		}
		if field.Before == field.After && !field.Sensitive {
			continue
		}
		if field.Before == field.After && !showEmpty {
			continue
		}
		seen[field.Path] = struct{}{}
		label := humanPath(semanticFieldLabel(field.Path))
		if field.Sensitive {
			field.Before, field.After = "<redacted>", "<redacted>"
		}
		effects = append(effects, fmt.Sprintf("%s %s -> %s", label, strconv.Quote(field.Before), strconv.Quote(field.After)))
	}
	return effects
}

func semanticFieldLabel(path string) string {
	name := path[strings.LastIndex(path, ".")+1:]
	if (strings.HasPrefix(path, "config.modelRoles.") || strings.HasPrefix(path, "profile.modelRoles.")) && name != "default" {
		return "role " + name
	}
	switch name {
	case "default", "primary":
		return "model"
	case "model_reasoning_effort", "reasoning_effort", "effortLevel", "variant":
		return "effort"
	case "defaultThinkingLevel", "thinkingDefault":
		return "thinking level"
	case "maxEffort":
		return "subagent max effort"
	case "model_provider", "defaultProvider", "provider":
		return "provider"
	case "defaultModel", "modelRoles":
		return "model"
	}
	return name
}

func writeCompactWarnings(output io.Writer, target install.TargetPlan, styles outputStyles) error {
	if target.VersionCheck != nil && target.VersionCheck.Status == install.VersionOutOfRange {
		if _, err := fmt.Fprintf(output, "    %s: installed version %s is outside tested range %s; check agent compatibility before applying\n", styles.warning("warning"), humanPath(target.VersionCheck.Detected), humanPath(target.VersionCheck.Range)); err != nil {
			return err
		}
	}
	for _, diagnostic := range target.Diagnostics {
		if diagnostic.Message == target.Reason {
			continue
		}
		if strings.HasSuffix(diagnostic.Code, ".install.profile_state_separate") {
			note := "named profile has separate target-owned state; sign in there if needed (authentication was not inspected)"
			if target.Target.Name == "openclaw" {
				note += "; OPENCLAW_CONFIG_PATH or OPENCLAW_STATE_DIR may override its location"
			}
			if _, err := fmt.Fprintf(output, "    %s: %s\n", styles.warning("warning"), note); err != nil {
				return err
			}
			continue
		}
		if diagnostic.Severity == profilemango.SeverityError || diagnostic.Severity == profilemango.SeverityWarning && (diagnostic.Code == "install.version_not_found" || diagnostic.Code == "install.version_unknown" || diagnostic.Code == "install.adopt_backup") {
			if _, err := fmt.Fprintf(output, "    %s: %s\n", styles.warning("warning"), humanPath(diagnostic.Message)); err != nil {
				return err
			}
		}
	}
	return nil
}

func writeHumanInstallPlan(output io.Writer, plan install.Plan) error {
	if _, err := fmt.Fprintf(output, "plan %s (%s)\n", humanPath(plan.PlanID), humanPath(plan.Status)); err != nil {
		return err
	}
	for _, target := range orderedTargetPlans(plan.Targets) {
		if err := writeHumanTarget(output, target); err != nil {
			return err
		}
		if err := writeTargetWarnings(output, target.Diagnostics); err != nil {
			return err
		}
	}
	return nil
}

// writeApplyHint ends a ready human plan with the exact command that applies it.
func writeApplyHint(output io.Writer, profile string, options installOptions, plan install.Plan) error {
	if options.jsonOutput || plan.Status != install.StatusReady {
		return nil
	}
	_, err := fmt.Fprintf(output, "\nNothing was written. Apply this plan with --apply to confirm interactively, or run:\n  %s\n", installApplyCommand(profile, options, plan.PlanID))
	return err
}

// installApplyCommand rebuilds the planning arguments, then appends non-interactive consent.
func installApplyCommand(profile string, options installOptions, planID string) string {
	args := []string{"mango"}
	if homePathOverride != "" {
		args = append(args, "--home", homePathOverride)
	}
	args = append(args, options.command(), profile)
	args = append(args, installPlanArgs(options)...)
	args = append(args, "--apply", "--yes", "--expect-plan", planID)
	for index, arg := range args {
		args[index] = install.ShellQuote(arg)
	}
	return strings.Join(args, " ")
}

func installPlanArgs(options installOptions) []string {
	var args []string
	for _, pair := range [][2]string{{"--profiles", options.profiles}, {"--resource-root", options.resourceRoot}, {"--bindings", options.bindings}} {
		if pair[1] != "" {
			args = append(args, pair[0], pair[1])
		}
	}
	for _, group := range []struct {
		flag   string
		values []string
	}{{"--target", options.targets}, {"--agent", options.agents}, {"--config", options.configValues()}, {"--manifest", options.manifests}} {
		for _, value := range group.values {
			args = append(args, group.flag, value)
		}
	}
	for _, flag := range []struct {
		name string
		set  bool
	}{{"--all", options.all}, {"--no-backup", options.noBackup}, {"--override", options.override}, {"--strict", options.strict}, {"--default", options.makeDefault}} {
		if flag.set {
			args = append(args, flag.name)
		}
	}
	return args
}

func writeTargetWarnings(output io.Writer, diagnostics profilemango.Diagnostics) error {
	for _, diagnostic := range diagnostics {
		if diagnostic.Severity != profilemango.SeverityWarning {
			continue
		}
		if _, err := fmt.Fprintf(output, "    warning: %s\n", humanPath(diagnostic.Message)); err != nil {
			return err
		}
	}
	return nil
}

func writeHumanTarget(output io.Writer, target install.TargetPlan) error {
	if _, err := fmt.Fprintf(output, "  %s: %s", humanPath(target.Target.String()), humanPath(target.Status)); err != nil {
		return err
	}
	if target.Reason != "" {
		if _, err := fmt.Fprintf(output, " (%s)", humanPath(target.Reason)); err != nil {
			return err
		}
	}
	if _, err := fmt.Fprintln(output); err != nil {
		return err
	}
	if target.Config != nil {
		if _, err := fmt.Fprintf(output, "    config: %s (%s)\n", humanPath(target.Config.Path), humanPath(target.Config.Source)); err != nil {
			return err
		}
	}
	if err := writeInstallMode(output, target.Install); err != nil {
		return err
	}
	if err := writeHumanVersionCheck(output, target.VersionCheck); err != nil {
		return err
	}
	if err := writeSkippedRequirements(output, target.SkippedRequirements); err != nil {
		return err
	}
	seen := make(map[string]struct{}, len(target.Fields))
	if err := writeHumanFields(output, target.Fields, "    ", seen, target.Status != install.StatusNoop); err != nil {
		return err
	}
	for _, file := range orderedFilePlans(target.Files) {
		if _, err := fmt.Fprintf(output, "    %s: %s\n", humanPath(file.Path), humanPath(file.Action)); err != nil {
			return err
		}
		if err := writeHumanFields(output, file.Fields, "      ", seen, file.Action != install.ActionNoop); err != nil {
			return err
		}
	}
	return nil
}

// writeInstallMode says how to use a named profile, or notes that the agent has no profiles.
func writeInstallMode(output io.Writer, mode *install.InstallMode) error {
	if mode == nil {
		return nil
	}
	if mode.Mode != install.InstallModeNamedProfile {
		_, err := fmt.Fprintln(output, "    note: "+install.NoProfilesNote)
		return err
	}
	if _, err := fmt.Fprintf(output, "    use it: %s\n", humanPath(mode.UseCommand)); err != nil {
		return err
	}
	if !mode.SetsDefault {
		return nil
	}
	_, err := fmt.Fprintln(output, "    also the default: the agent uses this profile when none is chosen")
	return err
}

// writeSkippedRequirements lists, on one line, the profile settings this agent does
// not receive, after one explicit line per route effort it does not apply.
func writeSkippedRequirements(output io.Writer, skipped []install.SkippedRequirement) error {
	names := make([]string, 0, len(skipped))
	for _, requirement := range skipped {
		switch {
		case requirement.Requirement == install.RequirementEffort:
			if _, err := fmt.Fprintf(output, "    effort %s%s: NOT APPLIED (%s)\n", humanPath(requirement.Value), roleSuffix(requirement.Role), humanPath(requirement.Reason)); err != nil {
				return err
			}
		case requirement.Role != "":
			names = append(names, requirement.Requirement+roleSuffix(requirement.Role)+": "+humanPath(requirement.Reason))
		case requirement.Requirement == install.RequirementInstructions && requirement.Count == 1:
			names = append(names, requirement.Requirement+" (1 file)")
		case requirement.Requirement == install.RequirementInstructions:
			names = append(names, fmt.Sprintf("%s (%d files)", requirement.Requirement, requirement.Count))
		case requirement.Count > 0:
			names = append(names, fmt.Sprintf("%s (%d)", requirement.Requirement, requirement.Count))
		case requirement.Value != "":
			names = append(names, requirement.Requirement+" "+requirement.Value)
		default:
			names = append(names, requirement.Requirement)
		}
	}
	if len(names) == 0 {
		return nil
	}
	_, err := fmt.Fprintf(output, "    not installed for this agent: %s\n", strings.Join(names, ", "))
	return err
}

// roleSuffix names the portable role a skipped requirement belongs to, if any.
func roleSuffix(role string) string {
	if role == "" {
		return ""
	}
	return " (role " + humanPath(role) + ")"
}

// writeHumanVersionCheck notes the installed agent version against the tested
// range; any mismatch is also listed as a warning.
func writeHumanVersionCheck(output io.Writer, check *install.VersionCheck) error {
	if check == nil {
		return nil
	}
	var note string
	switch check.Status {
	case install.VersionInRange:
		note = fmt.Sprintf("%s (in tested range %s)", check.Detected, check.Range)
	case install.VersionOutOfRange:
		note = fmt.Sprintf("%s (outside tested range %s)", check.Detected, check.Range)
	case install.VersionNotFound:
		note = check.Binary + " not found on PATH"
	default:
		note = "unknown"
	}
	_, err := fmt.Fprintf(output, "    installed version: %s\n", humanPath(note))
	return err
}

func writeHumanFields(output io.Writer, fields []install.FieldChange, indent string, seen map[string]struct{}, showEmpty bool) error {
	for _, field := range orderedFieldChanges(fields) {
		if field.Path == "" {
			continue
		}
		if _, found := seen[field.Path]; found {
			continue
		}
		if field.Before == field.After {
			if (field.Before == "" || field.Sensitive) && showEmpty {
				if _, err := fmt.Fprintf(output, "%sfield %s\n", indent, humanPath(field.Path)); err != nil {
					return err
				}
				seen[field.Path] = struct{}{}
			}
			continue
		}
		if field.Sensitive {
			field.Before, field.After = "<redacted>", "<redacted>"
		}
		if _, err := fmt.Fprintf(output, "%sfield %s: %s -> %s\n", indent, humanPath(field.Path), strconv.Quote(field.Before), strconv.Quote(field.After)); err != nil {
			return err
		}
		seen[field.Path] = struct{}{}
	}
	return nil
}

func orderedTargetPlans(targets []install.TargetPlan) []install.TargetPlan {
	result := append([]install.TargetPlan(nil), targets...)
	sort.SliceStable(result, func(i, j int) bool {
		return result[i].Target.String() < result[j].Target.String()
	})
	return result
}

func orderedFilePlans(files []install.FilePlan) []install.FilePlan {
	result := append([]install.FilePlan(nil), files...)
	sort.SliceStable(result, func(i, j int) bool {
		if result[i].Path != result[j].Path {
			return result[i].Path < result[j].Path
		}
		return result[i].Action < result[j].Action
	})
	return result
}

func orderedFieldChanges(fields []install.FieldChange) []install.FieldChange {
	ordered := append([]install.FieldChange(nil), fields...)
	sort.SliceStable(ordered, func(i, j int) bool {
		if ordered[i].Path != ordered[j].Path {
			return ordered[i].Path < ordered[j].Path
		}
		if ordered[i].Sensitive != ordered[j].Sensitive {
			return ordered[i].Sensitive
		}
		if ordered[i].Before != ordered[j].Before {
			return ordered[i].Before < ordered[j].Before
		}
		return ordered[i].After < ordered[j].After
	})
	result := make([]install.FieldChange, 0, len(ordered))
	for _, field := range ordered {
		if len(result) == 0 || result[len(result)-1].Path != field.Path {
			result = append(result, field)
			continue
		}
		merged := &result[len(result)-1]
		merged.Sensitive = merged.Sensitive || field.Sensitive
		if merged.Before == merged.After && field.Before != field.After {
			merged.Before, merged.After = field.Before, field.After
		}
	}
	return result
}

func humanPath(value string) string {
	if strings.IndexFunc(value, func(r rune) bool {
		return unicode.IsControl(r) || r == '\\' || r == '"'
	}) >= 0 {
		return strconv.Quote(value)
	}
	return value
}

func confirmInstall(input io.Reader, output io.Writer, planID string) error {
	if _, err := fmt.Fprintf(output, "Apply plan %s? [y/N] ", planID); err != nil {
		return err
	}
	line, err := bufio.NewReader(input).ReadString('\n')
	if err != nil && err != io.EOF {
		return fmt.Errorf("apply declined: confirmation input failed: %w", err)
	}
	answer := strings.ToLower(strings.TrimSpace(line))
	if answer != "y" && answer != "yes" {
		return fmt.Errorf("apply declined")
	}
	return nil
}

func terminalInput(reader io.Reader) bool {
	file, ok := reader.(*os.File)
	if !ok {
		return false
	}
	fd := file.Fd()
	return isatty.IsTerminal(fd) || isatty.IsCygwinTerminal(fd)
}

func (options installOptions) expectPlanOrPlanID(plan install.Plan) string {
	if options.expectPlan != "" {
		return options.expectPlan
	}
	return plan.PlanID
}

func writeApplyReport(cmd *cobra.Command, report install.ApplyReport, jsonOutput bool) error {
	if jsonOutput {
		data, err := json.MarshalIndent(report, "", "  ")
		if err != nil {
			return fmt.Errorf("encode apply report: %w", err)
		}
		_, err = cmd.OutOrStdout().Write(append(data, '\n'))
		return err
	}
	statuses := append([]install.ApplyTargetResult(nil), report.Targets...)
	sort.Slice(statuses, func(i, j int) bool { return statuses[i].Target < statuses[j].Target })
	styles := stylesFor(cmd.OutOrStdout(), true)
	for _, target := range statuses {
		if _, err := fmt.Fprintf(cmd.OutOrStdout(), "applied %s: %s\n", humanPath(target.Target), styledPlanStatus(target.Status, styles)); err != nil {
			return err
		}
	}
	return nil
}
