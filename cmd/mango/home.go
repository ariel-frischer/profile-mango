package main

import (
	"fmt"

	"github.com/spf13/cobra"
)

func newHomeCmd() *cobra.Command {
	var state bool
	cmd := &cobra.Command{
		Use:     "home",
		Aliases: []string{"h"},
		Short:   "Print the folder mango is using for profiles and settings",
		Args:    cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			resolve := selectedHome
			if state {
				resolve = selectedStateDir
			}
			path, err := resolve()
			if err != nil {
				return err
			}
			if _, err := fmt.Fprintln(cmd.OutOrStdout(), path); err != nil {
				return fmt.Errorf("writing profile home: %w", err)
			}
			return nil
		},
	}
	cmd.Flags().BoolVar(&state, "state", false, "print the install history folder instead ($PROFILE_MANGO_STATE_DIR, else $XDG_STATE_HOME/profile-mango, else ~/.local/state/profile-mango)")
	return cmd
}
