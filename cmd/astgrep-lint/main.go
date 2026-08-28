// Command astgrep-lint rejects model edits that introduce ast-grep violations.
//
// With no arguments it behaves as a PreToolUse hook, reading an event on stdin.
// With file arguments it scans those files and reports to stderr, which is how
// it is used as a bare command from the Bash tool via the plugin's bin/ entry.
package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/sourcec0de/claude-plugins/hookio"
)

func main() {
	if len(os.Args) > 1 {
		os.Exit(runCLI(os.Args[1:]))
	}
	hookio.Run(decide)
}

// decide lints the file as it would exist after the pending edit and denies the
// call when the edit introduces a violation that was not already there. Every
// file is checked, not only the ones a grammar exists for, because the text
// rules have to hold everywhere.
func decide(event hookio.Event) hookio.Decision {
	filePath := event.ToolInput.FilePath
	if filePath == "" {
		return hookio.Noop()
	}

	afterContent := contentAfterEdit(event)
	if afterContent == "" {
		return hookio.Noop()
	}

	before, beforeContent, err := scanExisting(event)
	if err != nil {
		return failClosed(event.HookEventName, err)
	}

	after, err := scanViolations(scanParams{
		Cwd:      event.Cwd,
		FilePath: filePath,
		Content:  afterContent,
	})
	if err != nil {
		return failClosed(event.HookEventName, err)
	}

	introduced := diffViolations(violationDelta{
		Before: filterSuppressed(before, beforeContent),
		After:  filterSuppressed(after, afterContent),
	})
	if len(introduced) == 0 {
		return hookio.Noop()
	}
	return hookio.Deny(event.HookEventName, formatViolations(introduced))
}

// failClosed denies the edit when the scan itself could not run. A linter that
// silently stops linting is worse than one that is loudly unavailable.
func failClosed(hookEventName string, err error) hookio.Decision {
	return hookio.Deny(hookEventName, fmt.Sprintf(
		"Static analysis could not run, so this edit is rejected (fail-closed): %v\n"+
			"  Install ast-grep (nix profile add nixpkgs#ast-grep, or npm i -g @ast-grep/cli) and retry.",
		err))
}

// contentAfterEdit reconstructs the full post-edit file. A PreToolUse hook sees
// only a fragment on an Edit, so the replacement is applied against the file on
// disk to give ast-grep something parseable.
func contentAfterEdit(event hookio.Event) string {
	if event.ToolInput.Content != "" {
		return event.ToolInput.Content
	}
	if event.ToolInput.NewString == "" {
		return ""
	}
	existing, err := os.ReadFile(event.ToolInput.FilePath)
	if err != nil || event.ToolInput.OldString == "" {
		return event.ToolInput.NewString
	}
	count := 1
	if event.ToolInput.ReplaceAll {
		count = -1
	}
	return strings.Replace(string(existing), event.ToolInput.OldString, event.ToolInput.NewString, count)
}
