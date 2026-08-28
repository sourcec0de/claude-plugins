package main

import (
	"testing"

	"github.com/sourcec0de/claude-plugins/hookio"
	"github.com/stretchr/testify/require"
)

type textCase struct {
	Name   string
	Line   string
	RuleID string
}

// The fixtures below are the exact strings the rules exist to reject, so each
// one carries the opt-out directive the hook honours. Without it this file
// could not be edited while the plugin is watching the repository.
func flaggedCases() []textCase {
	return []textCase{
		{
			Name: "session trailer",
			// astgrep-allow: no-session-url -- fixture the rule under test must match
			Line:   "Claude-Session: https://claude.ai/code/session_01UGQfHLtXYnF8xu",
			RuleID: "no-session-url",
		},
		{
			Name: "bare session link in prose",
			// astgrep-allow: no-session-url -- fixture the rule under test must match
			Line:   "Discussed in https://claude.ai/code/session_01UGQfHLtXYnF8xu earlier.",
			RuleID: "no-session-url",
		},
		{
			Name: "co-author trailer",
			// astgrep-allow: no-model-attribution -- fixture the rule under test must match
			Line:   "Co-Authored-By: Claude <noreply@anthropic.com>",
			RuleID: "no-model-attribution",
		},
		{
			Name: "co-author trailer lowercased and unspaced",
			// astgrep-allow: no-model-attribution -- fixture the rule under test must match
			Line:   "co-authored-by:Claude Opus 5",
			RuleID: "no-model-attribution",
		},
		{
			Name: "anthropic noreply address alone",
			// astgrep-allow: no-model-attribution -- fixture the rule under test must match
			Line:   "author = noreply@anthropic.com",
			RuleID: "no-model-attribution",
		},
	}
}

func cleanLines() []string {
	return []string{
		"See https://claude.com/claude-code for the CLI.",
		"The docs live at https://claude.ai/code with no session in the path.",
		"Co-Authored-By: Jane Doe <jane@example.com>",
		"This paragraph mentions Claude and Anthropic without attributing anything.",
	}
}

func TestScanTextFlagsAttribution(t *testing.T) {
	t.Parallel()

	for _, tc := range flaggedCases() {
		t.Run(tc.Name, func(t *testing.T) {
			t.Parallel()

			got := scanText(scanParams{FilePath: "NOTES.md", Content: tc.Line})
			require.Len(t, got, 1)
			require.Equal(t, tc.RuleID, got[0].RuleID)
			require.Equal(t, 1, got[0].Line)
			require.Equal(t, "NOTES.md", got[0].File)
			require.Equal(t, "error", got[0].Severity)
		})
	}
}

func TestScanTextAllowsCleanLines(t *testing.T) {
	t.Parallel()

	for _, line := range cleanLines() {
		require.Empty(t, scanText(scanParams{FilePath: "NOTES.md", Content: line}), line)
	}
}

func TestScanTextReportsTheOffendingLine(t *testing.T) {
	t.Parallel()

	// astgrep-allow: no-model-attribution -- fixture the rule under test must match
	content := "chore: init\n\nfixes a thing\n\nCo-Authored-By: Claude <noreply@anthropic.com>\n"
	got := scanText(scanParams{FilePath: "COMMIT_EDITMSG", Content: content})
	require.Len(t, got, 1)
	require.Equal(t, 5, got[0].Line)
}

func TestScanTextDoesNotFlagItsOwnRuleDefinitions(t *testing.T) {
	t.Parallel()

	for _, rule := range textRules {
		require.Empty(t, scanText(scanParams{FilePath: "attribution.go", Content: rule.Pattern.String()}), rule.ID)
		require.Empty(t, scanText(scanParams{FilePath: "attribution.go", Content: rule.Message}), rule.ID)
	}
}

func TestDecideDeniesAttributionInAnyFileType(t *testing.T) {
	t.Parallel()

	// astgrep-allow: no-model-attribution -- fixture the rule under test must match
	decision := runDecide(t, decideCase{
		FileName:     "CHANGELOG.md",
		WriteContent: "# Changelog\n\nCo-Authored-By: Claude <noreply@anthropic.com>\n",
	})
	require.Equal(t, hookio.PermissionDeny, decision.PermissionDecision())
	require.Contains(t, decision.Reason(), "no-model-attribution")
}

func TestDecideDeniesSessionLinkInAnyFileType(t *testing.T) {
	t.Parallel()

	// astgrep-allow: no-session-url -- fixture the rule under test must match
	decision := runDecide(t, decideCase{
		FileName:     "notes.txt",
		WriteContent: "context: https://claude.ai/code/session_01UGQfHLtXYnF8xu\n",
	})
	require.Equal(t, hookio.PermissionDeny, decision.PermissionDecision())
	require.Contains(t, decision.Reason(), "no-session-url")
}

func TestDecideAllowsAnUnrelatedEditToAFileThatAlreadyLeaks(t *testing.T) {
	t.Parallel()

	// astgrep-allow: no-session-url -- fixture the rule under test must match
	existing := "# Notes\n\nhttps://claude.ai/code/session_01UGQfHLtXYnF8xu\n"
	decision := runDecide(t, decideCase{
		FileName:       "notes.md",
		InitialContent: existing,
		OldString:      "# Notes",
		NewString:      "# Notes on the release",
	})
	require.True(t, decision.IsNoop())
}

func TestDecideHonoursTheOptOutDirective(t *testing.T) {
	t.Parallel()

	// astgrep-allow: no-model-attribution -- fixture the rule under test must match
	content := "# Guide\n\n# astgrep-allow: no-model-attribution -- documenting the trailer this repo bans\nCo-Authored-By: Claude <noreply@anthropic.com>\n"
	decision := runDecide(t, decideCase{FileName: "guide.md", WriteContent: content})
	require.True(t, decision.IsNoop())
}
