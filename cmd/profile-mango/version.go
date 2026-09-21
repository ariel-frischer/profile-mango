package main

import (
	"fmt"
	"io"
	"runtime"
	"strings"

	"github.com/spf13/cobra"
	"gitlab.com/ariel-frischer/profile-mango/internal/version"
)

var versionPlain bool

var versionCmd = &cobra.Command{
	Use:     "version",
	Aliases: []string{"v"},
	Short:   "Display version information",
	Run: func(cmd *cobra.Command, _ []string) {
		if versionPlain {
			printPlainVersion(cmd.OutOrStdout())
		} else {
			printPrettyVersion(cmd.OutOrStdout())
		}
	},
}

func init() {
	versionCmd.Flags().BoolVar(&versionPlain, "plain", false, "Plain output without formatting")
}

func printPlainVersion(writer io.Writer) {
	_, _ = fmt.Fprintf(writer, "profile-mango %s\n", version.Version)
	_, _ = fmt.Fprintf(writer, "commit: %s\n", version.Commit)
	_, _ = fmt.Fprintf(writer, "built: %s\n", version.BuildDate)
	_, _ = fmt.Fprintf(writer, "go: %s\n", runtime.Version())
	_, _ = fmt.Fprintf(writer, "platform: %s/%s\n", runtime.GOOS, runtime.GOARCH)
}

func printPrettyVersion(writer io.Writer) {
	styles := stylesFor(writer, true)
	_, _ = fmt.Fprintf(writer, "\n%s\n\n", styles.dim("  profile-mango — Define portable coding-agent behavior once and compile it into deterministic, capability-aware target artifacts."))
	printVersionBox(writer, styles)
	_, _ = fmt.Fprintln(writer)
}

func printVersionBox(writer io.Writer, styles outputStyles) {
	const boxWidth = 44
	_, _ = fmt.Fprintln(writer, "╭"+strings.Repeat("─", boxWidth-2)+"╮")
	_, _ = fmt.Fprintln(writer, "│"+strings.Repeat(" ", boxWidth-2)+"│")
	for _, item := range versionInfo() {
		label := styles.heading(fmt.Sprintf("%10s", item.label))
		value := styles.success(item.value)
		padding := max(boxWidth-18-len(item.value), 0)
		_, _ = fmt.Fprintln(writer, "│   "+label+"  "+value+strings.Repeat(" ", padding)+" │")
	}
	_, _ = fmt.Fprintln(writer, "│"+strings.Repeat(" ", boxWidth-2)+"│")
	_, _ = fmt.Fprintln(writer, "╰"+strings.Repeat("─", boxWidth-2)+"╯")
}

func versionInfo() []struct{ label, value string } {
	return []struct{ label, value string }{
		{"Version", version.Version},
		{"Commit", truncateCommit(version.Commit)},
		{"Built", version.BuildDate},
		{"Go", runtime.Version()},
		{"Platform", fmt.Sprintf("%s/%s", runtime.GOOS, runtime.GOARCH)},
	}
}

func truncateCommit(commit string) string {
	if len(commit) > 8 {
		return commit[:8]
	}
	return commit
}
