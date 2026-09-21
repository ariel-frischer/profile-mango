package main

import (
	"io"
	"os"

	"github.com/fatih/color"
	"github.com/mattn/go-isatty"
	"gitlab.com/ariel-frischer/profile-mango/internal/agentcheck"
	"gitlab.com/ariel-frischer/profile-mango/pkg/profilemango"
)

type outputStyles struct {
	heading func(...interface{}) string
	label   func(...interface{}) string
	dim     func(...interface{}) string
	success func(...interface{}) string
	warning func(...interface{}) string
	failure func(...interface{}) string
	path    func(...interface{}) string
}

func stylesFor(writer io.Writer, humanOutput bool) outputStyles {
	return newOutputStyles(humanOutput && colorEnabled(writer))
}

func newOutputStyles(enabled bool) outputStyles {
	enabled = enabled && os.Getenv("NO_COLOR") == "" && os.Getenv("TERM") != "dumb"
	return outputStyles{
		heading: colorSprint(enabled, color.FgYellow, color.Bold),
		label:   colorSprint(enabled, color.FgCyan, color.Bold),
		dim:     colorSprint(enabled, color.Faint),
		success: colorSprint(enabled, color.FgGreen, color.Bold),
		warning: colorSprint(enabled, color.FgYellow, color.Bold),
		failure: colorSprint(enabled, color.FgRed, color.Bold),
		path:    colorSprint(enabled, color.FgCyan),
	}
}

func colorSprint(enabled bool, attributes ...color.Attribute) func(...interface{}) string {
	style := color.New(attributes...)
	if enabled {
		style.EnableColor()
	} else {
		style.DisableColor()
	}
	return style.SprintFunc()
}

func colorEnabled(writer io.Writer) bool {
	if noColor || os.Getenv("NO_COLOR") != "" || os.Getenv("TERM") == "dumb" {
		return false
	}
	descriptor, ok := writer.(interface{ Fd() uintptr })
	if !ok {
		return false
	}
	fd := descriptor.Fd()
	return isatty.IsTerminal(fd) || isatty.IsCygwinTerminal(fd)
}

func styledDiagnosticLine(diagnostic profilemango.Diagnostic, styles outputStyles) string {
	path := diagnostic.Path
	if path == "" {
		path = "document"
	}
	severity := styles.warning(diagnostic.Severity)
	if diagnostic.Severity == profilemango.SeverityError {
		severity = styles.failure(diagnostic.Severity)
	}
	return severity + " " + styles.path(path) + ": " + diagnostic.Message + " (" + styles.dim(diagnostic.Code) + ")"
}

func styledAgentStatus(status agentcheck.Status, styles outputStyles) string {
	switch status {
	case agentcheck.StatusUnchanged:
		return styles.success(status)
	case agentcheck.StatusChanged, agentcheck.StatusUnavailable:
		return styles.failure(status)
	case agentcheck.StatusUnversioned, agentcheck.StatusRelocated:
		return styles.warning(status)
	default:
		return styles.dim(status)
	}
}
