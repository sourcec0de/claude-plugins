package main

import (
	"regexp"
	"strings"
)

// Text rules ban model attribution from every file, whatever its language.
//
// ast-grep cannot carry these. A rule there is bound to one grammar, and the
// strings being banned have to stay out of Markdown, YAML, JSON, commit
// message files and plain text just as much as out of Go — anything that can
// reach a commit. So these run as a plain line scan over the post-edit
// content, alongside the ast-grep pass rather than inside it.
//
// The patterns are written with escapes on purpose. `claude\.ai` matches a
// literal dot, so the pattern text does not match itself and this file is not
// a violation of its own rule. Rewriting it as `claude[.]ai` would break that.
type textRule struct {
	ID      string
	Pattern *regexp.Regexp
	Message string
}

// textLine is one line of a file, addressed for reporting.
type textLine struct {
	FilePath string
	Number   int
	Content  string
}

var textRules = []textRule{
	{
		ID:      "no-session-url",
		Pattern: regexp.MustCompile(`(?i)claude\.ai/[a-z0-9/_-]*session[_-][a-z0-9]+|claude-session\s*:`),
		Message: "A Claude session link must never be written to a file. It points at a private transcript and belongs in chat, not in anything committed or published. Delete the link and the trailer carrying it.",
	},
	{
		ID:      "no-model-attribution",
		Pattern: regexp.MustCompile(`(?i)co-authored-by\s*:[^\n]*(claude|anthropic)|noreply@anthropic\.com`),
		Message: "Do not attribute authorship to the model. A co-author trailer naming Claude or Anthropic must not appear in a commit message, a changelog, or any other file. Write the message as the human author.",
	},
}

// scanText reports every text-rule match in the content, one violation per
// rule per line, so a line carrying both a link and a trailer reports both.
func scanText(p scanParams) []Violation {
	var out []Violation
	for i, content := range strings.Split(p.Content, "\n") {
		out = append(out, lineViolations(textLine{
			FilePath: p.FilePath,
			Number:   i + 1,
			Content:  content,
		})...)
	}
	return out
}

func lineViolations(line textLine) []Violation {
	var out []Violation
	for _, rule := range textRules {
		match := rule.Pattern.FindString(line.Content)
		if match == "" {
			continue
		}
		out = append(out, Violation{
			RuleID:   rule.ID,
			Text:     match,
			Line:     line.Number,
			File:     line.FilePath,
			Message:  rule.Message,
			Severity: "error",
		})
	}
	return out
}
