package main

import (
	"github.com/ariel-frischer/profile-mango/internal/profilehome"
	"github.com/ariel-frischer/profile-mango/internal/version"
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:     "mango",
	Short:   "Install one coding-agent profile across Claude Code, Codex, OpenCode, and more",
	Version: version.Version,
}

var (
	homePathOverride string
	noColor          bool
	nonInteractive   bool
)

func init() {
	rootCmd.PersistentFlags().BoolVar(&noColor, "no-color", false, "disable colored output")
	rootCmd.PersistentFlags().StringVar(&homePathOverride, "home", "", "profile package home (default $PROFILE_MANGO_HOME or ~/.profile-mango)")
	rootCmd.PersistentFlags().BoolVar(&nonInteractive, "non-interactive", false, "disable prompts; commands that write files still require --yes and --expect-plan")

	rootCmd.SetHelpFunc(colorizedHelp)

	rootCmd.AddCommand(versionCmd)
	rootCmd.AddCommand(newHomeCmd())
	rootCmd.AddCommand(newInitCmd())
	rootCmd.AddCommand(newValidateCmd())
	rootCmd.AddCommand(newAgentsCmd())
	rootCmd.AddCommand(newRenderCmd())
	rootCmd.AddCommand(newInstallCmd())
	rootCmd.AddCommand(newUseCmd())
	rootCmd.AddCommand(newRouteCmd())
	rootCmd.AddCommand(newStatusCmd())
	rootCmd.AddCommand(newRestoreCmd())
	rootCmd.AddCommand(newDoctorCmd())
}

func selectedHome() (string, error) {
	return profilehome.Resolve(homePathOverride)
}
