package main

import (
	"fmt"
	"time"

	"github.com/ariel-frischer/profile-mango/internal/agentcheck"
	"github.com/spf13/cobra"
)

func newAgentsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "agents",
		Aliases: []string{"a"},
		Short:   "Check whether documented agent sources are still reachable, without changing anything",
		Hidden:  true,
	}
	cmd.AddCommand(newAgentsCheckCmd())
	return cmd
}

func newAgentsCheckCmd() *cobra.Command {
	var manifestPath string
	var targetIDs []string
	var timeout time.Duration
	var maxBodyBytes int64
	var maxRedirects int
	var jsonOutput bool

	cmd := &cobra.Command{
		Use:     "check",
		Aliases: []string{"c"},
		Short:   "Check each documented agent source for changes or outages",
		Args:    cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			selected, err := targetList(targetIDs)
			if err != nil {
				return err
			}
			report, err := agentcheck.CheckFile(cmd.Context(), manifestPath, agentcheck.Options{
				TargetIDs:    selected,
				Timeout:      timeout,
				MaxBodyBytes: maxBodyBytes,
				MaxRedirects: maxRedirects,
				OnProgress: func(progress agentcheck.Progress) {
					if progress.State == "" {
						_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "checking source %d/%d (%s)\n", progress.Completed+1, progress.Total, progress.TargetID)
						return
					}
					_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "checked source %d/%d (%s): %s\n", progress.Completed, progress.Total, progress.TargetID, progress.State)
				},
			})
			if err != nil {
				return err
			}
			if jsonOutput {
				if err := agentcheck.FormatJSON(cmd.OutOrStdout(), report); err != nil {
					return err
				}
			} else {
				styles := stylesFor(cmd.OutOrStdout(), true)
				formatStatus := func(status agentcheck.Status) string { return styledAgentStatus(status, styles) }
				if err := agentcheck.FormatTextStyled(cmd.OutOrStdout(), report, formatStatus); err != nil {
					return err
				}
			}
			if report.Summary.Unavailable > 0 {
				return unavailableError(report.Summary.Unavailable)
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&manifestPath, "manifest", agentcheck.DefaultManifestPath, "source manifest path")
	cmd.Flags().StringArrayVarP(&targetIDs, "target", "t", nil, "check target IDs, comma-separated or repeated")
	cmd.Flags().DurationVar(&timeout, "timeout", agentcheck.DefaultTimeout, "HTTP request timeout")
	cmd.Flags().Int64Var(&maxBodyBytes, "max-body-bytes", agentcheck.DefaultMaxBodyBytes, "maximum response body size to hash")
	cmd.Flags().IntVar(&maxRedirects, "max-redirects", agentcheck.DefaultMaxRedirects, "maximum redirects to follow")
	cmd.Flags().BoolVar(&jsonOutput, "json", false, "write a deterministic JSON report")
	return cmd
}

func unavailableError(count int) error {
	return fmt.Errorf("source check found %d unavailable source(s); unavailable is not unchanged", count)
}
