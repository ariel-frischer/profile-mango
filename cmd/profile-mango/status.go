package main

import (
	"fmt"
	"io"
	"path/filepath"
	"strings"

	"github.com/ariel-frischer/profile-mango/pkg/install"
	"github.com/spf13/cobra"
)

func newStatusCmd() *cobra.Command {
	var options installOptions
	cmd := &cobra.Command{
		Use:   "status",
		Short: "Show which profile each agent uses and whether its owned files are edited or out of date; never writes",
		Long: "Reads each agent's ownership manifest at its default config path (or --config), reports the recorded " +
			"profile, every owned file as in-sync, edited, or missing, and whether the profile's current sources " +
			"would change what is installed.",
		Args:         cobra.NoArgs,
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runStatus(cmd, options)
		},
	}
	flags := cmd.Flags()
	flags.StringVar(&options.profiles, "profiles", "", "profile repository root (defaults to <home>/profiles)")
	flags.StringVar(&options.resourceRoot, "resource-root", "", "resource package root (defaults to <home>)")
	flags.StringVar(&options.bindings, "bindings", "", "local route bindings file (defaults to <home>/bindings/local.yaml)")
	flags.StringArrayVarP(&options.targets, "target", "t", nil, "target[@version], comma-separated or repeated (defaults to every supported agent)")
	flags.StringArrayVar(&options.configs, "config", nil, "target[@version]=explicit config path; selects the target")
	flags.StringArrayVar(&options.manifests, "manifest", nil, "target[@version]=explicit ownership manifest path")
	flags.BoolVar(&options.jsonOutput, "json", false, "emit the deterministic status as JSON")
	return cmd
}

func runStatus(cmd *cobra.Command, options installOptions) error {
	paths, err := resolveInstallPaths(options)
	if err != nil {
		return err
	}
	registry := install.DefaultRegistry()
	var targets []install.TargetRequest
	if len(options.targets) > 0 || len(options.configValues()) > 0 || len(options.manifests) > 0 {
		if targets, err = installTargets(options, registry); err != nil {
			return err
		}
	}
	report, err := install.InspectStatus(install.StatusRequest{
		ProfilesRoot: paths.profiles, ResourceRoot: paths.resourceRoot, BindingsPath: paths.bindings,
		Registry: registry, Env: install.OSPathEnv(), Targets: targets,
	})
	if err != nil {
		return err
	}
	if options.jsonOutput {
		data, err := report.JSON()
		if err != nil {
			return err
		}
		_, err = cmd.OutOrStdout().Write(data)
		return err
	}
	return writeHumanStatus(cmd.OutOrStdout(), report)
}

func writeHumanStatus(output io.Writer, report install.StatusReport) error {
	styles := stylesFor(output, true)
	for _, target := range report.Targets {
		if err := writeHumanTargetStatus(output, target, styles); err != nil {
			return err
		}
	}
	return nil
}

func writeHumanTargetStatus(output io.Writer, target install.TargetStatus, styles outputStyles) error {
	if target.State != install.StatusManaged {
		detail := "not managed"
		if target.Reason != "" {
			detail += " (" + target.Reason + ")"
		}
		_, err := fmt.Fprintf(output, "%s  %s\n", styles.label(target.Target.String()), styles.dim(detail))
		return err
	}
	source := target.Source
	if target.SourceReason != "" {
		source += ": " + target.SourceReason
	}
	if _, err := fmt.Fprintf(output, "%s  profile %s (generation %d) in %s\n  sources: %s\n", styles.label(target.Target.String()), target.Profile, target.Generation, humanPath(filepath.Dir(target.ConfigPath)), source); err != nil {
		return err
	}
	for _, file := range target.Files {
		fields := ""
		if len(file.Fields) > 0 {
			fields = "  " + strings.Join(file.Fields, ", ")
		}
		if _, err := fmt.Fprintf(output, "  %-8s %-18s %s%s\n", file.State, file.Kind, file.Path, fields); err != nil {
			return err
		}
	}
	return nil
}
