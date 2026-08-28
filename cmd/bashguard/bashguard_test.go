package main

import (
	"testing"

	"github.com/sourcec0de/claude-plugins/hookio"
	"github.com/stretchr/testify/require"
)

func runDecide(command string) hookio.Decision {
	return decide(hookio.Event{
		HookEventName: "PreToolUse",
		ToolName:      "Bash",
		ToolInput:     hookio.ToolInput{Command: command},
	})
}

func TestDecideAllowsEmptyCommand(t *testing.T) {
	t.Parallel()

	require.True(t, runDecide("").IsNoop())
}

func TestDecideAllowsBenignCommands(t *testing.T) {
	t.Parallel()

	for _, command := range []string{
		"ls -la",
		"go test ./...",
		"rm stale.txt",
		"git status",
		"rmdir empty",
	} {
		t.Run(command, func(t *testing.T) {
			require.True(t, runDecide(command).IsNoop(), "must allow: %s", command)
		})
	}
}

func TestDecideDeniesDestructiveCommands(t *testing.T) {
	t.Parallel()

	cases := map[string]string{
		"rm -rf build":  "no-rm-rf",
		"rm -rf /":      "no-rm-rf",
		"cd /tmp":       "no-cd",
		"git push":      "no-git-write-ops",
		"git commit -m": "no-git-write-ops",
	}
	for command, wantRule := range cases {
		t.Run(command, func(t *testing.T) {
			got := runDecide(command)
			require.False(t, got.IsNoop(), "must deny: %s", command)
			require.Equal(t, hookio.PermissionDeny, got.PermissionDecision())
			require.Contains(t, got.Reason(), wantRule)
		})
	}
}

func TestDecideFailsOpenLoudlyWhenAstGrepMissing(t *testing.T) {
	t.Setenv("PATH", "/nonexistent")

	got := runDecide("rm -rf /")
	require.False(t, got.IsNoop(), "an unchecked command must not pass silently")
	require.Empty(t, got.PermissionDecision(),
		"bashguard fails open: it must not deny when it could not check")
	require.Contains(t, got.Reason(), "allowed it unchecked")
}

func TestScanCommandReportsRuleAndMessage(t *testing.T) {
	t.Parallel()

	got, err := scanCommand("", "rm -rf node_modules")
	require.NoError(t, err)
	require.NotEmpty(t, got)
	require.Equal(t, "no-rm-rf", got[0].RuleID)
	require.Contains(t, got[0].Message, "rm -rf is banned")
}

func TestFormatViolationsCollapsesMessage(t *testing.T) {
	t.Parallel()

	got := formatViolations([]Violation{{
		RuleID:  "no-rm-rf",
		Text:    "rm -rf x",
		Message: "first line.\nsecond line.\n",
	}})
	require.Equal(t, "no-rm-rf: rm -rf x\n  first line. second line.", got)
}

func TestFormatViolationsEmpty(t *testing.T) {
	t.Parallel()

	require.Empty(t, formatViolations(nil))
}
