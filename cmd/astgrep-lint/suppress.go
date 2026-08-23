package main

import (
	"regexp"
	"strings"
	"unicode"
)

// ignoreDirectivePattern matches an opt-out comment on the line directly above
// a violation, in either // or # comment syntax:
//
//	// astgrep-allow: no-fmt-println -- intentional CLI diagnostic output
var ignoreDirectivePattern = regexp.MustCompile(`^\s*(?://|#)\s*astgrep-allow:\s*(\S+)\s*--\s*(.+?)\s*$`)

// minJustificationChars is the shortest justification accepted. It exists to
// stop "-- ok" style opt-outs from passing as reasoning.
const minJustificationChars = 10

func filterSuppressed(violations []Violation, content string) []Violation {
	if len(violations) == 0 {
		return violations
	}
	lines := strings.Split(content, "\n")
	out := make([]Violation, 0, len(violations))
	for _, v := range violations {
		if isSuppressed(v, lines) {
			continue
		}
		out = append(out, v)
	}
	return out
}

func isSuppressed(v Violation, lines []string) bool {
	idx := v.Line - 2
	if idx < 0 || idx >= len(lines) {
		return false
	}
	match := ignoreDirectivePattern.FindStringSubmatch(lines[idx])
	if match == nil {
		return false
	}
	if match[1] != v.RuleID {
		return false
	}
	return substantiveCharCount(strings.TrimSpace(match[2])) >= minJustificationChars
}

func substantiveCharCount(s string) int {
	count := 0
	for _, r := range s {
		if !unicode.IsSpace(r) {
			count++
		}
	}
	return count
}
