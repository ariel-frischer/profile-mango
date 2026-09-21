package main

import (
	"os"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
	"gitlab.com/ariel-frischer/profile-mango/internal/profilehome"
	"gitlab.com/ariel-frischer/profile-mango/internal/version"
)

var rootCmd = &cobra.Command{
	Use:     "profile-mango",
	Short:   "Define portable coding-agent behavior once and compile it into deterministic, capability-aware target artifacts.",
	Version: version.Version,
}

var homePathOverride string

func init() {
	// Disable colors when not writing to a terminal.
	if fi, err := os.Stdout.Stat(); err == nil {
		if fi.Mode()&os.ModeCharDevice == 0 {
			color.NoColor = true
		}
	}

	var noColor bool
	rootCmd.PersistentFlags().BoolVar(&noColor, "no-color", false, "disable colored output")
	rootCmd.PersistentFlags().StringVar(&homePathOverride, "home", "", "profile package home (default $PROFILE_MANGO_HOME or ~/.profile-mango)")
	rootCmd.PersistentPreRun = func(cmd *cobra.Command, args []string) {
		if noColor {
			color.NoColor = true
		}
	}

	rootCmd.SetHelpFunc(colorizedHelp)

	rootCmd.AddCommand(versionCmd)
	rootCmd.AddCommand(newHomeCmd())
	rootCmd.AddCommand(newInitCmd())
	rootCmd.AddCommand(newValidateCmd())
	rootCmd.AddCommand(newAgentsCmd())
	rootCmd.AddCommand(newRenderCmd())
}

func selectedHome() (string, error) {
	return profilehome.Resolve(homePathOverride)
}
