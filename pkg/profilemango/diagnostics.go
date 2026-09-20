package profilemango

import (
	"fmt"
	"sort"
	"strings"
)

// Severity distinguishes blocking diagnostics from useful policy warnings.
type Severity string

const (
	SeverityError   Severity = "error"
	SeverityWarning Severity = "warning"
)

// Diagnostic is stable across repository roots because it stores no absolute path.
type Diagnostic struct {
	Severity Severity `json:"severity"`
	Code     string   `json:"code"`
	Path     string   `json:"path,omitempty"`
	Message  string   `json:"message"`
	Line     int      `json:"line,omitempty"`
	Column   int      `json:"column,omitempty"`
}

// Diagnostics is both a collection and an error for callers that want one return value.
type Diagnostics []Diagnostic

// Add appends one diagnostic with optional source coordinates.
func (d *Diagnostics) Add(severity Severity, code, path, message string, line, column int) {
	*d = append(*d, Diagnostic{
		Severity: severity,
		Code:     code,
		Path:     path,
		Message:  message,
		Line:     line,
		Column:   column,
	})
}

// HasErrors reports whether validation must fail closed.
func (d Diagnostics) HasErrors() bool {
	for _, item := range d {
		if item.Severity == SeverityError {
			return true
		}
	}
	return false
}

// Errors returns only blocking diagnostics in stable order.
func (d Diagnostics) Errors() Diagnostics {
	filtered := make(Diagnostics, 0, len(d))
	for _, item := range d {
		if item.Severity == SeverityError {
			filtered = append(filtered, item)
		}
	}
	return filtered.Sorted()
}

// Sorted returns a copy ordered by field path, code, coordinates, and message.
func (d Diagnostics) Sorted() Diagnostics {
	result := append(Diagnostics(nil), d...)
	sort.SliceStable(result, func(i, j int) bool {
		left, right := result[i], result[j]
		if left.Path != right.Path {
			return left.Path < right.Path
		}
		if left.Code != right.Code {
			return left.Code < right.Code
		}
		if left.Line != right.Line {
			return left.Line < right.Line
		}
		if left.Column != right.Column {
			return left.Column < right.Column
		}
		return left.Message < right.Message
	})
	return result
}

// Error formats all diagnostics in stable order for command-line use.
func (d Diagnostics) Error() string {
	if len(d) == 0 {
		return ""
	}
	lines := make([]string, 0, len(d))
	for _, item := range d.Sorted() {
		location := item.Path
		if location == "" {
			location = "document"
		}
		if item.Line > 0 {
			location = fmt.Sprintf("%s:%d:%d", location, item.Line, item.Column)
		}
		lines = append(lines, fmt.Sprintf("%s %s: %s (%s)", item.Severity, location, item.Message, item.Code))
	}
	return strings.Join(lines, "\n")
}

// Err returns nil when there are no blocking diagnostics.
func (d Diagnostics) Err() error {
	if !d.HasErrors() {
		return nil
	}
	return d.Errors()
}
