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
			"would change what is installed, naming each owned setting whose live value differs from the profile. Each agent is labeled " +
			"with its installed version, plus the tested version when they differ. It also lists each skill of an agent's default " +
			"profile whose copy in the global skills directory differs from the profile's copy, and which side is newer.",
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
	flags.StringVar(&options.globalSkills, "global-skills", "", "global skills directory compared with profile skills (defaults to ~/.agents/skills)")
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
		Registry: registry, Env: install.OSPathEnv(), Targets: targets, GlobalSkillsRoot: options.globalSkills,
		DetectVersion: versionDetectorWithProgress(cmd.ErrOrStderr(), installVersionDetector),
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
	return writeHumanSkillDrift(output, report.Skills, styles)
}

// writeHumanSkillDrift prints each profile skill whose global copy differs, the newer
// side, and the safe next step; the profile's copy is canonical.
func writeHumanSkillDrift(output io.Writer, drift []install.SkillDrift, styles outputStyles) error {
	for _, entry := range drift {
		if _, err := fmt.Fprintf(output, "%s  %s differs from profile %s (%s); newer: %s\n  files: %s\n  %s\n", styles.label("skill "+entry.Skill), entry.Global, entry.Profile, entry.Snapshot, entry.Newer, strings.Join(entry.Files, ", "), entry.Hint); err != nil {
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
		_, err := fmt.Fprintf(output, "%s  %s\n", styles.label(statusTargetLabel(target)), styles.dim(detail))
		return err
	}
	source := target.Source
	if target.SourceReason != "" {
		source += ": " + target.SourceReason
	}
	recorded := ""
	if target.RecordedVersion != "" {
		recorded = " (recorded at " + target.RecordedVersion + ")"
	}
	if _, err := fmt.Fprintf(output, "%s%s  profile %s (generation %d) in %s\n  sources: %s\n", styles.label(statusTargetLabel(target)), recorded, target.Profile, target.Generation, humanPath(filepath.Dir(target.ConfigPath)), source); err != nil {
		return err
	}
	for _, file := range target.Files {
		fields := ""
		if len(file.Fields) > 0 {
			fields = "  " + strings.Join(file.Fields, ", ")
		}
		if _, err := fmt.Fprintf(output, "  %-11s %-18s %s%s\n", file.State, file.Kind, file.Path, fields); err != nil {
			return err
		}
	}
	return writeHumanDrift(output, target.Drift)
}

// statusTargetLabel names the target at its installed version when detected,
// adding the tested version when they differ or the installed one is unknown.
func statusTargetLabel(target install.TargetStatus) string {
	check := target.VersionCheck
	switch {
	case check == nil:
		return target.Target.String()
	case check.Detected == check.Qualified:
		return target.Target.Name + "@" + check.Detected
	case check.Detected != "":
		return fmt.Sprintf("%s@%s (tested %s)", target.Target.Name, check.Detected, check.Qualified)
	case check.Status == install.VersionNotFound:
		return fmt.Sprintf("%s (tested %s; %s not found on PATH)", target.Target.Name, check.Qualified, check.Binary)
	default:
		return fmt.Sprintf("%s (tested %s; installed version unreadable)", target.Target.Name, check.Qualified)
	}
}

// writeHumanDrift prints one line per owned setting or file that differs from the profile.
func writeHumanDrift(output io.Writer, drift []install.FieldDrift) error {
	if len(drift) == 0 {
		return nil
	}
	if _, err := fmt.Fprintln(output, "  drift:"); err != nil {
		return err
	}
	for _, entry := range drift {
		line := fmt.Sprintf("    %s  live %s  profile %s", entry.Path, driftValue(entry.Live), driftValue(entry.Profile))
		if entry.State != "" {
			line = fmt.Sprintf("    %s  %s", entry.Path, entry.State)
		}
		if _, err := fmt.Fprintln(output, line); err != nil {
			return err
		}
	}
	return nil
}

func driftValue(value string) string {
	if value == "" {
		return "(absent)"
	}
	return value
}
