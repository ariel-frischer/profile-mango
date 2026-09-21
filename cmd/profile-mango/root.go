package main

import (
	"os"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
	"gitlab.com/ariel-frischer/profile-mango/internal/config"
	"gitlab.com/ariel-frischer/profile-mango/internal/version"
)

var rootCmd = &cobra.Command{
	Use:     "profile-mango",
	Short:   "Define portable coding-agent behavior once and compile it into deterministic, capability-aware target artifacts.",
	Version: version.Version,
}
var configPathOverride string

func init() {
	// Disable colors when not writing to a terminal.
	if fi, err := os.Stdout.Stat(); err == nil {
		if fi.Mode()&os.ModeCharDevice == 0 {
			color.NoColor = true
		}
	}

	var noColor bool
	rootCmd.PersistentFlags().BoolVar(&noColor, "no-color", false, "disable colored output")
	rootCmd.PersistentFlags().StringVar(&configPathOverride, "config", "", "config file path (default $PROFILE_MANGO_CONFIG or ~/.config/profile-mango/config.yaml)")
	rootCmd.PersistentPreRun = func(cmd *cobra.Command, args []string) {
		if noColor {
			color.NoColor = true
		}
	}

	rootCmd.SetHelpFunc(colorizedHelp)

	rootCmd.AddCommand(versionCmd)
	rootCmd.AddCommand(configCmd)
	rootCmd.AddCommand(newValidateCmd())
	rootCmd.AddCommand(newAgentsCmd())
}
func selectedConfigPath() string {
	return config.Path(configPathOverride)
}
