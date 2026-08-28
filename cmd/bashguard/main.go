// Command bashguard rejects Bash commands that match a plugin's shell rules.
//
// It runs as a PreToolUse hook on the Bash tool. The command string is parsed
// with the bash grammar rather than pattern-matched as text, so a rule can
// distinguish `rm -rf build` from an `rm` mentioned inside a quoted string.
//
// Which rules apply is chosen by --rules, naming a rule tree in the package.
// One binary serves every shell-rule plugin this way, so a plugin that ships
// only rules — gitbutler, whose opinion is worth having only where GitButler
// is installed — needs no binary of its own and can be installed on its own.
package main

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/sourcec0de/claude-plugins/hookio"
)

const (
	defaultRuleTree = "bashguard"
	rulesFlag       = "--rules"
	rulesFlagAssign = "--rules="
)

// ErrMissingRuleTree reports a --rules that names nothing.
var ErrMissingRuleTree = errors.New("--rules requires a rule tree name, e.g. --rules gitbutler")

func main() {
	commands, ruleTree, err := parseArgs(os.Args[1:])
	if err != nil {
		fmt.Fprintf(os.Stderr, "bashguard: %v\n", err)
		os.Exit(1)
	}
	if len(commands) > 0 {
		os.Exit(runCLI(ruleTree, commands))
	}
	hookio.Run(func(event hookio.Event) hookio.Decision {
		return decide(ruleTree, event)
	})
}

// parseArgs pulls a leading --rules out of the argument list and returns what
// is left. Anything remaining is a command to check, which is what makes the
// hook mode and the bare-command mode distinguishable: a hook is passed no
// commands and reads its event from stdin.
func parseArgs(args []string) ([]string, string, error) {
	if len(args) == 0 || !strings.HasPrefix(args[0], rulesFlag) {
		return args, defaultRuleTree, nil
	}
	if value, found := strings.CutPrefix(args[0], rulesFlagAssign); found {
		return args[1:], value, nil
	}
	if args[0] != rulesFlag || len(args) < 2 {
		return nil, "", ErrMissingRuleTree
	}
	return args[2:], args[1], nil
}

func decide(ruleTree string, event hookio.Event) hookio.Decision {
	command := event.ToolInput.Command
	if command == "" {
		return hookio.Noop()
	}

	violations, err := scanCommand(scanParams{
		RuleTree: ruleTree,
		Cwd:      event.Cwd,
		Command:  command,
	})
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
func runCLI(ruleTree string, commands []string) int {
	failed := false
	for _, command := range commands {
		violations, err := scanCommand(scanParams{RuleTree: ruleTree, Command: command})
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
