package main

import (
	"github.com/spf13/cobra"
	"gitlab.com/ariel-frischer/profile-mango/internal/profilehome"
	"gitlab.com/ariel-frischer/profile-mango/internal/version"
)

var rootCmd = &cobra.Command{
	Use:     "mango",
	Short:   "Define portable coding-agent behavior once and compile it into deterministic, capability-aware target artifacts.",
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
	rootCmd.PersistentFlags().BoolVar(&nonInteractive, "non-interactive", false, "disable prompts; write commands still require explicit consent flags")

	rootCmd.SetHelpFunc(colorizedHelp)

	rootCmd.AddCommand(versionCmd)
	rootCmd.AddCommand(newHomeCmd())
	rootCmd.AddCommand(newInitCmd())
	rootCmd.AddCommand(newValidateCmd())
	rootCmd.AddCommand(newAgentsCmd())
	rootCmd.AddCommand(newRenderCmd())
	rootCmd.AddCommand(newInstallCmd())
}

func selectedHome() (string, error) {
	return profilehome.Resolve(homePathOverride)
}
