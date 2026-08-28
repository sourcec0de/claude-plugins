// Command bashguard rejects Bash commands that match the plugin's shell rules.
//
// It runs as a PreToolUse hook on the Bash tool. The command string is parsed
// with the bash grammar rather than pattern-matched as text, so a rule can
// distinguish `rm -rf build` from an `rm` mentioned inside a quoted string.
package main

import (
	"fmt"
	"os"

	"github.com/sourcec0de/claude-plugins/hookio"
)

func main() {
	if len(os.Args) > 1 {
		os.Exit(runCLI(os.Args[1:]))
	}
	hookio.Run(decide)
}

func decide(event hookio.Event) hookio.Decision {
	command := event.ToolInput.Command
	if command == "" {
		return hookio.Noop()
	}

	violations, err := scanCommand(event.Cwd, command)
	if err != nil {
		return unavailable(event.HookEventName, err)
	}
	if len(violations) == 0 {
		return hookio.Noop()
	}
	return hookio.Deny(event.HookEventName, formatViolations(violations))
}

// unavailable fails open, because failing closed here would deny every Bash
// call and make the session unusable. It says so out loud rather than passing
// the command through as if it had been checked.
func unavailable(hookEventName string, err error) hookio.Decision {
	return hookio.Context(hookEventName, fmt.Sprintf(
		"bashguard could not check this command and allowed it unchecked: %v", err))
}

// runCLI checks command strings passed as arguments, for use as a bare command.
func runCLI(commands []string) int {
	failed := false
	for _, command := range commands {
		violations, err := scanCommand("", command)
		if err != nil {
			fmt.Fprintf(os.Stderr, "bashguard: %v\n", err)
			return 1
		}
		if len(violations) == 0 {
			continue
		}
		fmt.Fprintln(os.Stderr, formatViolations(violations))
		failed = true
	}
	if failed {
		return 1
	}
	return 0
}
