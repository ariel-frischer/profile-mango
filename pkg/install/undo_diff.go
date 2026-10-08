package install

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"unicode"
)

const maxDiffLines = 200

// sensitiveLine matches config lines that may carry credentials; their content never reaches a diff.
var sensitiveLine = regexp.MustCompile(`(?i)(key|token|secret|password|passwd|credential|auth|bearer|cookie)`)

// UnifiedDiff renders one hunk covering the changed middle of before and after, after trimming
// the common leading and trailing lines; afterLabel names the right side, e.g. "after undo".
// It is a bounded, redacted preview, not a minimal diff.
func UnifiedDiff(path, afterLabel string, before, after []byte) string {
	old, updated := diffLines(before), diffLines(after)
	prefix := 0
	for prefix < len(old) && prefix < len(updated) && old[prefix] == updated[prefix] {
		prefix++
	}
	suffix := 0
	for suffix < len(old)-prefix && suffix < len(updated)-prefix && old[len(old)-1-suffix] == updated[len(updated)-1-suffix] {
		suffix++
	}
	removed, added := old[prefix:len(old)-suffix], updated[prefix:len(updated)-suffix]
	var builder strings.Builder
	label := path
	if strings.IndexFunc(label, unicode.IsControl) >= 0 {
		label = strconv.Quote(label)
	}
	fmt.Fprintf(&builder, "--- %s (current)\n+++ %s (%s)\n", label, label, afterLabel)
	if len(removed) == 0 && len(added) == 0 {
		builder.WriteString("  (no content change)\n")
		return builder.String()
	}
	fmt.Fprintf(&builder, "@@ -%d,%d +%d,%d @@\n", prefix+1, len(removed), prefix+1, len(added))
	written := writeDiffLines(&builder, "-", removed, 0)
	writeDiffLines(&builder, "+", added, written)
	return builder.String()
}

func diffLines(data []byte) []string {
	if len(data) == 0 {
		return nil
	}
	return strings.SplitAfter(string(data), "\n")
}

func writeDiffLines(builder *strings.Builder, marker string, lines []string, written int) int {
	for _, line := range lines {
		if written == maxDiffLines {
			builder.WriteString("  (diff truncated)\n")
			return written + 1
		}
		if written > maxDiffLines {
			return written
		}
		text := strings.TrimRight(line, "\r\n")
		switch {
		case sensitiveLine.MatchString(text):
			text = "<redacted: line may contain a credential>"
		case strings.IndexFunc(text, unicode.IsControl) >= 0:
			text = strconv.Quote(text)
		}
		builder.WriteString(marker + text + "\n")
		written++
	}
	return written
}
