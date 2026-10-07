package main

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/ariel-frischer/profile-mango/pkg/install"
	"github.com/spf13/cobra"
)

type restoreOptions struct {
	target, config, legacyConfig, originalPlan, expectPlan string
	targets                                                []string
	apply, yes, override, jsonOutput, verbose              bool
}

// newRestoreCmd builds "undo"; "restore" remains an alias for the earlier Codex-only command.
func newRestoreCmd() *cobra.Command {
	var options restoreOptions
	cmd := &cobra.Command{Use: "undo", Aliases: []string{"restore"}, Short: "Preview, then apply, undoing the latest install for a target and restoring its original settings", Args: cobra.NoArgs,
		SilenceUsage: true, RunE: func(cmd *cobra.Command, _ []string) error { return runRestore(cmd, options) }}
	cmd.Flags().StringArrayVarP(&options.targets, "target", "t", nil, "target[@version], comma-separated or repeated; a bare name selects its single qualified version")
	cmd.Flags().StringVar(&options.config, "config", "", "config path the install wrote (defaults to the target's documented config path)")
	cmd.Flags().StringVar(&options.legacyConfig, "config-path", "", "deprecated alias for --config")
	_ = cmd.Flags().MarkDeprecated("config-path", "use --config instead")
	cmd.Flags().StringVar(&options.originalPlan, "original-plan", "", "64-hex install plan ID to undo (defaults to the latest committed install)")
	cmd.Flags().BoolVar(&options.override, "override", false, "discard edits made to the config after the install")
	cmd.Flags().BoolVar(&options.apply, "apply", false, "apply the previewed undo")
	cmd.Flags().BoolVar(&options.yes, "yes", false, "confirm apply without a terminal; requires --expect-plan")
	cmd.Flags().StringVar(&options.expectPlan, "expect-plan", "", "expected undo plan ID, not the original install ID")
	cmd.Flags().BoolVar(&options.jsonOutput, "json", false, "emit the undo plan as JSON")
	cmd.Flags().BoolVarP(&options.verbose, "verbose", "v", false, "show full undo hashes and redacted diff")
	return cmd
}

func runRestore(cmd *cobra.Command, options restoreOptions) error {
	if err := validateRestoreOptions(options); err != nil {
		return err
	}
	values := options.targets
	if options.target != "" {
		values = append([]string{options.target}, values...)
	}
	selected, err := targetList(values)
	if err != nil {
		return err
	}
	selected, err = resolvedUndoTargets(selected)
	if err != nil {
		return err
	}
	if len(selected) > 1 {
		return previewMultipleUndo(cmd, options, selected)
	}
	options.target = selected[0]
	request, err := undoRequest(options, install.DefaultRegistry(), install.OSPathEnv())
	if err != nil {
		return err
	}
	plan, err := install.BuildUndoPlan(request)
	if err != nil {
		return fmt.Errorf("plan undo: %w", err)
	}
	if !options.apply || !options.jsonOutput {
		if err := writeRestorePlan(cmd, plan, options.jsonOutput, options.verbose); err != nil {
			return err
		}
	}
	if !options.apply {
		if !options.jsonOutput {
			return writeUndoHint(cmd.OutOrStdout(), plan, options)
		}
		return nil
	}
	if err := authorizeRestore(cmd, options, plan, terminalInput(cmd.InOrStdin())); err != nil {
		return err
	}
	if err := install.ApplyRestorePlan(plan, options.expectedID(plan.PlanID)); err != nil {
		return err
	}
	if options.jsonOutput {
		return json.NewEncoder(cmd.OutOrStdout()).Encode(map[string]string{"status": "committed", "planID": plan.PlanID})
	}
	_, err = fmt.Fprintf(cmd.OutOrStdout(), "undo %s: %s\n", plan.PlanID, styledPlanStatus("committed", stylesFor(cmd.OutOrStdout(), true)))
	return err
}

func resolvedUndoTargets(values []string) ([]string, error) {
	registry := install.DefaultRegistry()
	selected := make([]string, 0, len(values))
	seen := make(map[string]struct{})
	for _, value := range values {
		target, err := registry.ResolveTarget(value)
		if err != nil {
			return nil, fmt.Errorf("resolve undo target %q: %w", value, err)
		}
		if _, exists := seen[target.String()]; !exists {
			seen[target.String()] = struct{}{}
			selected = append(selected, target.String())
		}
	}
	return selected, nil
}

// Multi-target undo is preview-only: each plan ID authorizes exactly one target.
// Re-run apply once per target so a single consent cannot partially undo a set.
func previewMultipleUndo(cmd *cobra.Command, options restoreOptions, selected []string) error {
	if options.apply || options.configPath() != "" || options.originalPlan != "" {
		return fmt.Errorf("multi-target undo is preview-only; apply each target separately with its own --expect-plan (and --config or --original-plan)")
	}
	var plans []install.RestorePlan
	for _, name := range selected {
		options.target = name
		request, err := undoRequest(options, install.DefaultRegistry(), install.OSPathEnv())
		if err != nil {
			return err
		}
		plan, err := install.BuildUndoPlan(request)
		if err != nil {
			return fmt.Errorf("plan undo %s: %w", name, err)
		}
		plans = append(plans, plan)
	}
	if options.jsonOutput {
		return json.NewEncoder(cmd.OutOrStdout()).Encode(plans)
	}
	for _, plan := range plans {
		if err := writeRestorePlan(cmd, plan, false, options.verbose); err != nil {
			return err
		}
		if err := writeUndoHint(cmd.OutOrStdout(), plan, options); err != nil {
			return err
		}
	}
	return nil
}

// undoRequest resolves the target like install: a bare name selects its qualified version, and an omitted
// --config falls back to the target's documented default config path.
func undoRequest(options restoreOptions, registry *install.Registry, env install.PathEnv) (install.UndoRequest, error) {
	target, err := registry.ResolveTarget(options.target)
	if err != nil {
		return install.UndoRequest{}, fmt.Errorf("resolve undo target: %w", err)
	}
	config := options.configPath()
	if strings.TrimSpace(config) == "" {
		if config, err = registry.DefaultConfigPath(target, env); err != nil {
			return install.UndoRequest{}, fmt.Errorf("resolve undo config: %w", err)
		}
	}
	return install.UndoRequest{Target: target, ConfigPath: config, OriginalPlanID: options.originalPlan, Override: options.override, Registry: registry}, nil
}

func validateRestoreOptions(options restoreOptions) error {
	if options.config != "" && options.legacyConfig != "" {
		return fmt.Errorf("use --config only; --config-path is a deprecated alias")
	}
	if strings.TrimSpace(options.target) == "" && len(options.targets) == 0 {
		return fmt.Errorf("undo requires --target")
	}
	if options.yes && !options.apply {
		return fmt.Errorf("--yes requires --apply")
	}
	if options.expectPlan != "" && (!options.apply || !options.yes) {
		return fmt.Errorf("--expect-plan requires --apply --yes")
	}
	if options.yes && options.expectPlan == "" {
		return fmt.Errorf("--yes requires --expect-plan")
	}
	if options.apply && (options.jsonOutput || nonInteractive) && !options.yes {
		return fmt.Errorf("non-interactive or JSON undo apply requires --yes --expect-plan")
	}
	return nil
}

func (options restoreOptions) configPath() string {
	if options.config != "" {
		return options.config
	}
	return options.legacyConfig
}

func (options restoreOptions) expectedID(planID string) string {
	if options.yes {
		return options.expectPlan
	}
	return planID
}

func authorizeRestore(cmd *cobra.Command, options restoreOptions, plan install.RestorePlan, isTerminal bool) error {
	if options.yes {
		return nil
	}
	if !isTerminal || options.jsonOutput || nonInteractive {
		return fmt.Errorf("interactive undo apply requires a terminal; to apply the undo plan shown above, run:\n  %s", undoApplyCommand(plan, options))
	}
	return confirmInstall(cmd.InOrStdin(), cmd.OutOrStdout(), plan.PlanID)
}

func writeRestorePlan(cmd *cobra.Command, plan install.RestorePlan, asJSON, verbose bool) error {
	if asJSON {
		return json.NewEncoder(cmd.OutOrStdout()).Encode(plan)
	}
	output := cmd.OutOrStdout()
	if !verbose {
		return writeCompactUndo(output, plan)
	}
	if _, err := fmt.Fprintf(output, "undo plan %s (%s), original install %s\n  target: %s\n", plan.PlanID, plan.Status, plan.OriginalPlanID, humanPath(plan.Target)); err != nil {
		return err
	}
	for _, file := range plan.Files {
		before, after := orAbsent(file.BeforeSHA256), orAbsent(file.AfterSHA256)
		if _, err := fmt.Fprintf(output, "  %s %s: %s -> %s\n", humanPath(file.Action), humanPath(file.Path), humanPath(before), humanPath(after)); err != nil {
			return err
		}
		if err := writeRestoreDiff(output, file.Diff); err != nil {
			return err
		}
	}
	return nil
}

func writeCompactUndo(output io.Writer, plan install.RestorePlan) error {
	styles := stylesFor(output, true)
	if _, err := fmt.Fprintf(output, "%s %s (%s), original install %s\n  target: %s\n", styles.heading("undo plan"), humanPath(plan.PlanID), styledPlanStatus(plan.Status, styles), humanPath(plan.OriginalPlanID), styles.label(humanPath(plan.Target))); err != nil {
		return err
	}
	for _, file := range plan.Files {
		if _, err := fmt.Fprintf(output, "  %s %s\n", humanPath(file.Action), styles.path(humanPath(file.Path))); err != nil {
			return err
		}
		if err := writeRestoreDiff(output, file.Diff); err != nil {
			return err
		}
	}
	return nil
}

func writeUndoHint(output io.Writer, plan install.RestorePlan, options restoreOptions) error {
	_, err := fmt.Fprintf(output, "Nothing was written. Apply this undo plan with:\n  %s\n", undoApplyCommand(plan, options))
	return err
}

// undoApplyCommand rebuilds the undo arguments, then appends non-interactive consent.
func undoApplyCommand(plan install.RestorePlan, options restoreOptions) string {
	args := []string{"mango"}
	if homePathOverride != "" {
		args = append(args, "--home", homePathOverride)
	}
	args = append(args, "undo", "--target", plan.Target)
	if options.configPath() != "" {
		args = append(args, "--config", options.configPath())
	}
	if options.originalPlan != "" {
		args = append(args, "--original-plan", options.originalPlan)
	}
	args = append(args, "--apply", "--yes", "--expect-plan", plan.PlanID)
	for index := range args {
		args[index] = install.ShellQuote(args[index])
	}
	return strings.Join(args, " ")
}

func orAbsent(hash string) string {
	if hash == "" {
		return "absent"
	}
	return hash
}

func writeRestoreDiff(output io.Writer, diff string) error {
	for _, line := range strings.Split(strings.TrimSuffix(diff, "\n"), "\n") {
		if line == "" {
			continue
		}
		// unifiedDiff already quotes control characters and redacts credential-like lines.
		if _, err := fmt.Fprintf(output, "    %s\n", line); err != nil {
			return err
		}
	}
	return nil
}
