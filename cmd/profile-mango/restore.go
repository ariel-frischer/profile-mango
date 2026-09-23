package main

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"gitlab.com/ariel-frischer/profile-mango/pkg/install"
)

type restoreOptions struct {
	target, config, legacyConfig, originalPlan, expectPlan string
	apply, yes, jsonOutput                                 bool
}

func newRestoreCmd() *cobra.Command {
	var options restoreOptions
	cmd := &cobra.Command{Use: "restore", Short: "Preview or explicitly apply a guarded Codex install reversal", Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error { return runRestore(cmd, options) }}
	cmd.Flags().StringVar(&options.target, "target", "", "target codex or codex@0.154.0")
	cmd.Flags().StringVar(&options.config, "config", "", "explicit original Codex config path")
	cmd.Flags().StringVar(&options.legacyConfig, "config-path", "", "deprecated alias for --config")
	_ = cmd.Flags().MarkDeprecated("config-path", "use --config instead")
	cmd.Flags().StringVar(&options.originalPlan, "original-plan", "", "original 64-hex installation plan ID")
	cmd.Flags().BoolVar(&options.apply, "apply", false, "apply the previewed restore")
	cmd.Flags().BoolVar(&options.yes, "yes", false, "confirm apply without a terminal; requires --expect-plan")
	cmd.Flags().StringVar(&options.expectPlan, "expect-plan", "", "expected restore plan ID, not the original install ID")
	cmd.Flags().BoolVar(&options.jsonOutput, "json", false, "emit restore plan as JSON")
	return cmd
}

func runRestore(cmd *cobra.Command, options restoreOptions) error {
	if err := validateRestoreOptions(options); err != nil {
		return err
	}
	target, err := install.DefaultRegistry().ResolveTarget(options.target)
	if err != nil {
		return fmt.Errorf("resolve restore target: %w", err)
	}
	plan, err := install.BuildRestorePlan(target.String(), options.configPath(), options.originalPlan)
	if err != nil {
		return fmt.Errorf("plan Codex restore: %w", err)
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
	_, err = fmt.Fprintf(cmd.OutOrStdout(), "restore %s: committed\n", plan.PlanID)
	return err
}

func validateRestoreOptions(options restoreOptions) error {
	if options.config != "" && options.legacyConfig != "" {
		return fmt.Errorf("use --config only; --config-path is a deprecated alias")
	}
	if strings.TrimSpace(options.configPath()) == "" {
		return fmt.Errorf("restore requires --config")
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
		return fmt.Errorf("non-interactive or JSON restore apply requires --yes --expect-plan")
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
		return fmt.Errorf("interactive restore apply requires a terminal; use --yes --expect-plan")
	}
	return confirmInstall(cmd.InOrStdin(), cmd.OutOrStdout(), planID)
}

func writeRestorePlan(cmd *cobra.Command, plan install.RestorePlan, asJSON bool) error {
	if asJSON {
		return json.NewEncoder(cmd.OutOrStdout()).Encode(plan)
	}
	if _, err := fmt.Fprintf(cmd.OutOrStdout(), "restore plan %s: %s (original install %s)\n", plan.PlanID, plan.Status, plan.OriginalPlanID); err != nil {
		return err
	}
	for _, file := range plan.Files {
		if _, err := fmt.Fprintf(cmd.OutOrStdout(), "  %s %s: %s -> %s\n", humanPath(file.Action), humanPath(file.Path), humanPath(file.BeforeSHA256), humanPath(file.AfterSHA256)); err != nil {
			return err
		}
	}
	return nil
}
