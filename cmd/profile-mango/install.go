package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"unicode"

	"github.com/mattn/go-isatty"
	"github.com/spf13/cobra"
	"gitlab.com/ariel-frischer/profile-mango/pkg/install"
)

type installOptions struct {
	profiles       string
	resourceRoot   string
	bindings       string
	targets        []string
	agents         []string
	configs        []string
	manifests      []string
	configPaths    []string
	all            bool
	apply          bool
	yes            bool
	expectPlan     string
	noBackup       bool
	override       bool
	jsonOutput     bool
	nonInteractive bool
}

func newInstallCmd() *cobra.Command {
	var options installOptions
	cmd := &cobra.Command{
		Use:          "install <profile>",
		Aliases:      []string{"i"},
		Short:        "Plan a bounded target installation; apply only with explicit hash-bound consent",
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
	cmd.Flags().StringArrayVar(&options.targets, "target", nil, "exact target@version; repeat for multiple targets")
	cmd.Flags().StringArrayVar(&options.agents, "agent", nil, "target@version=primary:name or subagent:name; requires --config-path ending agents/name.md")
	cmd.Flags().StringArrayVar(&options.configs, "config-path", nil, "target=explicit config path; repeat for multiple targets")
	cmd.Flags().StringArrayVar(&options.configPaths, "config", nil, "target@version=explicit config path; repeat for multiple targets")
	cmd.Flags().StringArrayVar(&options.manifests, "manifest", nil, "target@version=explicit ownership manifest path")
	cmd.Flags().BoolVar(&options.all, "all", false, "plan every statically registered public target")
	cmd.Flags().BoolVar(&options.apply, "apply", false, "apply the already displayed plan after consent")
	cmd.Flags().BoolVar(&options.yes, "yes", false, "confirm non-interactive apply; requires --expect-plan")
	cmd.Flags().StringVar(&options.expectPlan, "expect-plan", "", "expected plan ID required for --yes and apply")
	cmd.Flags().BoolVar(&options.noBackup, "no-backup", false, "disable create-only pre-apply backups")
	cmd.Flags().BoolVar(&options.override, "override", false, "request an adapter-approved narrow ownership override")
	cmd.Flags().BoolVar(&options.jsonOutput, "json", false, "emit the deterministic plan as JSON")
	return cmd
}

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
	targets, err := installTargets(options, registry)
	if err != nil {
		return err
	}
	request := install.Request{
		ProfileName: profile, ProfilesRoot: paths.profiles, ResourceRoot: paths.resourceRoot,
		BindingsPath: paths.bindings, Targets: targets, All: false,
		Backup: !options.noBackup, Override: options.override, Registry: registry,
	}
	plan, err := install.BuildPlan(request)
	if err != nil {
		return err
	}
	writePlan := !options.apply || !options.jsonOutput || plan.Status == install.StatusBlocked
	if writePlan {
		if err := writeInstallPlan(cmd, plan, options.jsonOutput); err != nil {
			return err
		}
	}
	if !options.apply {
		if plan.Status == install.StatusBlocked {
			return fmt.Errorf("install plan is blocked")
		}
		return nil
	}
	if plan.Status == install.StatusBlocked {
		return fmt.Errorf("install plan is blocked; no files were written")
	}
	if options.yes {
		if options.expectPlan == "" {
			return fmt.Errorf("non-interactive apply requires --expect-plan")
		}
	} else if !terminalInput(cmd.InOrStdin()) {
		return fmt.Errorf("interactive apply requires a terminal; use --yes --expect-plan")
	} else if err := confirmInstall(cmd.InOrStdin(), cmd.OutOrStdout(), plan.PlanID); err != nil {
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

func validateInstallOptions(options installOptions) error {
	if !options.all && len(options.targets) == 0 {
		return fmt.Errorf("at least one --target target@version or --all is required")
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

func installTargets(options installOptions, registry *install.Registry) ([]install.TargetRequest, error) {
	agents, err := parseAgentSelections(options.agents)
	if err != nil {
		return nil, err
	}
	configValues := append([]string(nil), options.configs...)
	configValues = append(configValues, options.configPaths...)
	configMap, err := parseTargetPaths(configValues)
	if err != nil {
		return nil, err
	}
	manifestMap, err := parseTargetPaths(options.manifests)
	if err != nil {
		return nil, err
	}
	if options.all {
		if len(agents) > 0 {
			return nil, fmt.Errorf("--all and --agent are mutually exclusive")
		}
		return targetRequestsFromAll(registry, configMap, manifestMap), nil
	}
	if len(options.targets) == 0 {
		return nil, fmt.Errorf("provide --target target@version or --all")
	}
	result := make([]install.TargetRequest, 0, len(options.targets))
	seen := map[string]struct{}{}
	for _, value := range options.targets {
		target, err := install.ParseTarget(value)
		if err != nil {
			return nil, err
		}
		if _, found := seen[target.String()]; found {
			return nil, fmt.Errorf("duplicate target: %s", target.String())
		}
		seen[target.String()] = struct{}{}
		result = append(result, install.TargetRequest{Target: target, Agent: agents[target.String()], ConfigPath: targetPath(configMap, target), ManifestPath: targetPath(manifestMap, target)})
	}
	for key := range agents {
		if _, found := seen[key]; !found {
			return nil, fmt.Errorf("--agent target %s must also be selected with --target", key)
		}
	}
	return result, nil
}

func parseAgentSelections(values []string) (map[string]install.AgentDestination, error) {
	result := make(map[string]install.AgentDestination, len(values))
	for _, value := range values {
		selector, spec, found := strings.Cut(value, "=")
		if !found {
			return nil, fmt.Errorf("--agent requires target@version=primary:name or subagent:name")
		}
		target, err := install.ParseTarget(selector)
		if err != nil {
			return nil, err
		}
		mode, name, found := strings.Cut(spec, ":")
		if !found || mode == "" || name == "" {
			return nil, fmt.Errorf("--agent requires target@version=primary:name or subagent:name")
		}
		if _, exists := result[target.String()]; exists {
			return nil, fmt.Errorf("duplicate --agent selection for %s", target.String())
		}
		result[target.String()] = install.AgentDestination{Mode: mode, Name: name}
	}
	return result, nil
}

func targetRequestsFromAll(registry *install.Registry, configs, manifests map[string]string) []install.TargetRequest {
	targets := registry.Targets()
	result := make([]install.TargetRequest, 0, len(targets))
	for _, target := range targets {
		result = append(result, install.TargetRequest{Target: target, ConfigPath: targetPath(configs, target), ManifestPath: targetPath(manifests, target)})
	}
	return result
}

func parseTargetPaths(values []string) (map[string]string, error) {
	result := make(map[string]string, len(values))
	for _, value := range values {
		key, path, found := strings.Cut(value, "=")
		if !found || strings.TrimSpace(key) == "" || strings.TrimSpace(path) == "" {
			return nil, fmt.Errorf("path mapping must use target@version=path syntax")
		}
		mapKey := strings.TrimSpace(key)
		if strings.Contains(mapKey, "@") {
			target, err := install.ParseTarget(mapKey)
			if err != nil {
				return nil, err
			}
			mapKey = target.String()
		} else if filepath.Base(mapKey) != mapKey || strings.ContainsAny(mapKey, "/\\") {
			return nil, fmt.Errorf("path mapping target must be a simple name or target@version")
		}
		if _, found := result[mapKey]; found {
			return nil, fmt.Errorf("duplicate path mapping: %s", mapKey)
		}
		result[mapKey] = path
	}
	return result, nil
}

func targetPath(paths map[string]string, target install.Target) string {
	if value := paths[target.String()]; value != "" {
		return value
	}
	return paths[target.Name]
}

func writeInstallPlan(cmd *cobra.Command, plan install.Plan, jsonOutput bool) error {
	if jsonOutput {
		data, err := plan.JSON()
		if err != nil {
			return err
		}
		_, err = cmd.OutOrStdout().Write(data)
		return err
	}
	return writeHumanInstallPlan(cmd.OutOrStdout(), plan)
}

func writeHumanInstallPlan(output io.Writer, plan install.Plan) error {
	if _, err := fmt.Fprintf(output, "plan %s: %s\n", humanPath(plan.PlanID), humanPath(plan.Status)); err != nil {
		return err
	}
	for _, target := range orderedTargetPlans(plan.Targets) {
		if err := writeHumanTarget(output, target); err != nil {
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
	for _, target := range statuses {
		if _, err := fmt.Fprintf(cmd.OutOrStdout(), "applied %s: %s\n", target.Target, target.Status); err != nil {
			return err
		}
	}
	return nil
}
