package main

import (
	"fmt"

	"github.com/spf13/cobra"
)

func newHomeCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "home",
		Aliases: []string{"h"},
		Short:   "Print the effective profile package home",
		Args:    cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			path, err := selectedHome()
			if err != nil {
				return err
			}
			if _, err := fmt.Fprintln(cmd.OutOrStdout(), path); err != nil {
				return fmt.Errorf("writing profile home: %w", err)
			}
			return nil
		},
	}
}
