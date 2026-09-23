package main

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/spf13/cobra"
	"gitlab.com/ariel-frischer/profile-mango/pkg/install"
)

type restoreOptions struct {
	target, config, legacyConfig, originalPlan, expectPlan string
	apply, yes, override, jsonOutput                       bool
}

// newRestoreCmd builds "undo"; "restore" remains an alias for the earlier Codex-only command.
func newRestoreCmd() *cobra.Command {
	var options restoreOptions
	cmd := &cobra.Command{Use: "undo", Aliases: []string{"restore"}, Short: "Preview, then apply, undoing the latest install for a target and restoring its original settings", Args: cobra.NoArgs,
		SilenceUsage: true, RunE: func(cmd *cobra.Command, _ []string) error { return runRestore(cmd, options) }}
	cmd.Flags().StringVar(&options.target, "target", "", "target or target@version; a bare name selects its single qualified version")
	cmd.Flags().StringVar(&options.config, "config", "", "config path the install wrote (defaults to the target's documented config path)")
	cmd.Flags().StringVar(&options.legacyConfig, "config-path", "", "deprecated alias for --config")
	_ = cmd.Flags().MarkDeprecated("config-path", "use --config instead")
	cmd.Flags().StringVar(&options.originalPlan, "original-plan", "", "64-hex install plan ID to undo (defaults to the latest committed install)")
	cmd.Flags().BoolVar(&options.override, "override", false, "discard edits made to the config after the install")
	cmd.Flags().BoolVar(&options.apply, "apply", false, "apply the previewed undo")
	cmd.Flags().BoolVar(&options.yes, "yes", false, "confirm apply without a terminal; requires --expect-plan")
	cmd.Flags().StringVar(&options.expectPlan, "expect-plan", "", "expected undo plan ID, not the original install ID")
	cmd.Flags().BoolVar(&options.jsonOutput, "json", false, "emit the undo plan as JSON")
	return cmd
}

func runRestore(cmd *cobra.Command, options restoreOptions) error {
	if err := validateRestoreOptions(options); err != nil {
		return err
	}
	request, err := undoRequest(options, install.DefaultRegistry(), install.OSPathEnv())
	if err != nil {
		return err
	}
	plan, err := install.BuildUndoPlan(request)
	if err != nil {
		return fmt.Errorf("plan undo: %w", err)
	}
	if !options.apply || !options.jsonOutput {
		if err := writeRestorePlan(cmd, plan, options.jsonOutput); err != nil {
			return err
		}
	}
	if !options.apply {
		return nil
	}
	if err := authorizeRestore(cmd, options, plan.PlanID, terminalInput(cmd.InOrStdin())); err != nil {
		return err
	}
	if err := install.ApplyRestorePlan(plan, options.expectedID(plan.PlanID)); err != nil {
		return err
	}
	if options.jsonOutput {
		return json.NewEncoder(cmd.OutOrStdout()).Encode(map[string]string{"status": "committed", "planID": plan.PlanID})
	}
	_, err = fmt.Fprintf(cmd.OutOrStdout(), "undo %s: committed\n", plan.PlanID)
	return err
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
	if strings.TrimSpace(options.target) == "" {
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

func authorizeRestore(cmd *cobra.Command, options restoreOptions, planID string, isTerminal bool) error {
	if options.yes {
		return nil
	}
	if !isTerminal || options.jsonOutput || nonInteractive {
		return fmt.Errorf("interactive undo apply requires a terminal; use --yes --expect-plan")
	}
	return confirmInstall(cmd.InOrStdin(), cmd.OutOrStdout(), planID)
}

func writeRestorePlan(cmd *cobra.Command, plan install.RestorePlan, asJSON bool) error {
	if asJSON {
		return json.NewEncoder(cmd.OutOrStdout()).Encode(plan)
	}
	output := cmd.OutOrStdout()
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
