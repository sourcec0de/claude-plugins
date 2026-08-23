package main

import (
	"strconv"
	"strings"
)

// formatViolations renders violations for the model. Matched text is truncated
// to its first line and messages are collapsed onto one line, so a denial reads
// as a compact list rather than a wall of re-printed source.
func formatViolations(violations []Violation) string {
	if len(violations) == 0 {
		return ""
	}
	var sb strings.Builder
	for i, v := range violations {
		if i > 0 {
			sb.WriteByte('\n')
		}
		sb.WriteString(v.RuleID)
		sb.WriteString(": ")
		sb.WriteString(firstLine(v.Text))
		sb.WriteString(" @ ")
		sb.WriteString(v.File)
		sb.WriteByte(':')
		sb.WriteString(strconv.Itoa(v.Line))
		sb.WriteByte('\n')
		sb.WriteString("  ")
		sb.WriteString(normalizeMessage(v.Message))
	}
	return sb.String()
}

func firstLine(s string) string {
	if idx := strings.IndexByte(s, '\n'); idx >= 0 {
		return s[:idx]
	}
	return s
}

func normalizeMessage(s string) string {
	return strings.Join(strings.Fields(strings.TrimSpace(s)), " ")
}
