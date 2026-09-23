package main

import (
	"github.com/spf13/cobra"
	"gitlab.com/ariel-frischer/profile-mango/internal/profilehome"
	"gitlab.com/ariel-frischer/profile-mango/internal/version"
)

var rootCmd = &cobra.Command{
	Use:     "profile-mango",
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
	rootCmd.AddCommand(newRestoreCmd())
}

func selectedHome() (string, error) {
	return profilehome.Resolve(homePathOverride)
}
