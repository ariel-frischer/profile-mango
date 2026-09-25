package main

import (
	"fmt"

	"github.com/spf13/cobra"
	"gitlab.com/ariel-frischer/profile-mango/pkg/install"
)

func newUseCmd() *cobra.Command {
	options := installOptions{use: true}
	cmd := &cobra.Command{
		Use:   "use <profile>",
		Short: "Switch agents to a profile as their default; nothing is written until you pass --apply",
		Long: "Plans the profile as each target's default, including its global instruction files, and gives back " +
			"files the previous profile owned that this one does not write: files profile-mango created are removed " +
			"and adopted files get their pre-install bytes back from the create-only backup. Without --target or --all, " +
			"it switches every agent profile-mango already manages.",
		Args:         cobra.ExactArgs(1),
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			options.nonInteractive = nonInteractive
			return runInstall(cmd, args[0], options)
		},
	}
	flags := cmd.Flags()
	flags.StringVar(&options.profiles, "profiles", "", "profile repository root (defaults to <home>/profiles)")
	flags.StringVar(&options.resourceRoot, "resource-root", "", "resource package root (defaults to <home>)")
	flags.StringVar(&options.bindings, "bindings", "", "local route bindings file (defaults to <home>/bindings/local.yaml)")
	flags.StringArrayVarP(&options.targets, "target", "t", nil, "target[@version], comma-separated or repeated (defaults to every managed agent)")
	flags.StringArrayVar(&options.configs, "config", nil, "target[@version]=explicit config path; selects the target; repeat for multiple targets")
	flags.StringArrayVar(&options.manifests, "manifest", nil, "target[@version]=explicit ownership manifest path")
	flags.BoolVar(&options.all, "all", false, "plan every supported agent that is installed; others are listed as skipped")
	flags.BoolVar(&options.apply, "apply", false, "apply the already displayed plan after consent")
	flags.BoolVar(&options.yes, "yes", false, "confirm non-interactive apply; requires --expect-plan")
	flags.StringVar(&options.expectPlan, "expect-plan", "", "expected plan ID required for --yes and apply")
	flags.BoolVar(&options.noBackup, "no-backup", false, "disable create-only pre-apply backups")
	flags.BoolVar(&options.override, "override", false, "allow overwriting settings that were changed outside profile-mango, where the target supports it")
	flags.BoolVar(&options.strict, "strict", false, "stop instead of skipping profile settings an agent cannot install")
	flags.BoolVar(&options.jsonOutput, "json", false, "emit the deterministic plan as JSON")
	flags.BoolVarP(&options.verbose, "verbose", "v", false, "show full plan details, field paths, and diagnostics")
	return cmd
}

// command is the subcommand that rebuilds this plan in an apply hint.
func (options installOptions) command() string {
	if options.use {
		return "use"
	}
	return "install"
}

// commandTargets selects the flagged targets, or for mango use without any, every
// agent whose default-path ownership manifest exists.
func commandTargets(options installOptions, registry *install.Registry) ([]install.TargetRequest, error) {
	if !options.use || options.all || len(options.targets) > 0 || len(options.configValues()) > 0 || len(options.manifests) > 0 {
		return installTargets(options, registry)
	}
	report, err := install.InspectStatus(install.StatusRequest{Registry: registry, Env: install.OSPathEnv()})
	if err != nil {
		return nil, err
	}
	targets := report.Managed()
	if len(targets) == 0 {
		return nil, fmt.Errorf("no agent is managed by profile-mango yet; pass --target or --all, or run mango install first")
	}
	return targets, nil
}
