package main

import (
	"fmt"
	"io"
	"strings"

	"github.com/spf13/cobra"
)

func colorizedHelp(cmd *cobra.Command, _ []string) {
	writer := cmd.OutOrStdout()
	styles := stylesFor(writer, true)
	_, _ = fmt.Fprintf(writer, "%s — %s\n", styles.label(cmd.CommandPath()), cmd.Short)
	printHelpDetails(writer, cmd, styles)
	printHelpCommands(writer, cmd, styles)
	printHelpFlags(writer, cmd, styles)
	if cmd.HasAvailableSubCommands() {
		_, _ = fmt.Fprintf(writer, "\n%s %s %s\n", styles.dim("Use"), styles.label(fmt.Sprintf("%s [command] --help", cmd.CommandPath())), styles.dim("for more information about a command."))
	}
	_, _ = fmt.Fprintln(writer)
}

func printHelpDetails(writer io.Writer, cmd *cobra.Command, styles outputStyles) {
	if cmd.Long != "" {
		_, _ = fmt.Fprintf(writer, "\n%s\n", styles.dim(cmd.Long))
	}
	if cmd.Runnable() {
		_, _ = fmt.Fprintf(writer, "\n%s\n  %s\n", styles.heading("Usage:"), cmd.UseLine())
	}
	if len(cmd.Aliases) > 0 {
		_, _ = fmt.Fprintf(writer, "\n%s\n  %s\n", styles.heading("Aliases:"), strings.Join(cmd.Aliases, ", "))
	}
}

func printHelpCommands(writer io.Writer, cmd *cobra.Command, styles outputStyles) {
	commands := visibleSubcommands(cmd)
	if len(commands) == 0 {
		return
	}
	_, _ = fmt.Fprintf(writer, "\n%s\n", styles.heading("Commands:"))
	maxLen := maxCommandNameLen(commands)
	for _, sub := range commands {
		padding := strings.Repeat(" ", maxLen-len(sub.Name())+2)
		_, _ = fmt.Fprintf(writer, "  %s%s%s\n", styles.label(sub.Name()), padding, styles.dim(sub.Short))
	}
}

func printHelpFlags(writer io.Writer, cmd *cobra.Command, styles outputStyles) {
	if flags := cmd.LocalFlags().FlagUsages(); flags != "" {
		_, _ = fmt.Fprintf(writer, "\n%s\n", styles.heading("Flags:"))
		printColorizedFlags(writer, flags, styles)
	}
	if flags := cmd.InheritedFlags().FlagUsages(); flags != "" {
		_, _ = fmt.Fprintf(writer, "\n%s\n", styles.heading("Global Flags:"))
		printColorizedFlags(writer, flags, styles)
	}
}

func printColorizedFlags(writer io.Writer, flagUsages string, styles outputStyles) {
	for _, line := range strings.Split(strings.TrimRight(flagUsages, "\n"), "\n") {
		if line == "" {
			_, _ = fmt.Fprintln(writer)
			continue
		}
		trimmed := strings.TrimLeft(line, " ")
		indent := len(line) - len(trimmed)
		parts := splitFlagLine(trimmed)
		if len(parts) == 2 {
			_, _ = fmt.Fprintf(writer, "%s%s  %s\n", strings.Repeat(" ", indent), styles.success(parts[0]), styles.dim(parts[1]))
		} else {
			_, _ = fmt.Fprintf(writer, "%s%s\n", strings.Repeat(" ", indent), styles.success(trimmed))
		}
	}
}

func splitFlagLine(s string) []string {
	for i := 0; i < len(s)-2; i++ {
		if s[i] != ' ' && s[i+1] == ' ' && s[i+2] == ' ' && i+3 < len(s) {
			desc := strings.TrimLeft(s[i+1:], " ")
			if desc != "" {
				return []string{s[:i+1], desc}
			}
		}
	}
	return []string{s}
}

func visibleSubcommands(cmd *cobra.Command) []*cobra.Command {
	var commands []*cobra.Command
	for _, sub := range cmd.Commands() {
		if !sub.Hidden && sub.Name() != "help" {
			commands = append(commands, sub)
		}
	}
	return commands
}

func maxCommandNameLen(commands []*cobra.Command) int {
	maxLen := 0
	for _, command := range commands {
		if len(command.Name()) > maxLen {
			maxLen = len(command.Name())
		}
	}
	return maxLen
}
