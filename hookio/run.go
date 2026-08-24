package hookio

import (
	"fmt"
	"os"
	"path/filepath"
)

// Handler turns a hook event into a decision.
type Handler func(Event) Decision

// Run reads a hook event from stdin, dispatches it to handler, emits the
// decision and exits with the matching status code.
func Run(handler Handler) {
	event, err := DecodeEvent(os.Stdin)
	if err != nil {
		fmt.Fprintf(os.Stderr, "hookio: %v\n", err)
		os.Exit(1)
	}
	os.Exit(handler(event).Emit(os.Stdout, os.Stderr))
}

// ErrNoRuleRoot reports that the rule tree could not be located.
var ErrNoRuleRoot = fmt.Errorf("cannot locate rules: neither CLAUDE_PLUGIN_ROOT nor CLAUDE_PLUGINS_RULE_ROOT is set and no go.mod was found above the working directory")

// RuleRoot returns the directory holding the ast-grep rule trees, resolved in
// three tiers.
//
// CLAUDE_PLUGIN_ROOT wins. Claude Code exports it when it spawns a hook, and
// honouring it is what lets a plugin update ship new rules without reinstalling
// the binary. Resolving against it is also mandatory for correctness: the
// plugin is copied into ~/.claude/plugins/cache and the hook runs with the
// user's project as its working directory, so anything that searched upward
// from there would silently lint against the user's own config.
//
// CLAUDE_PLUGINS_RULE_ROOT is set by the Nix wrapper and points at the rules
// bundled with the binary, so the commands work standalone outside a hook.
//
// The go.mod walk is last, covering `go test` and running from a checkout.
func RuleRoot() (string, error) {
	for _, key := range []string{"CLAUDE_PLUGIN_ROOT", "CLAUDE_PLUGINS_RULE_ROOT"} {
		if root := os.Getenv(key); root != "" {
			return root, nil
		}
	}
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for {
		if _, statErr := os.Stat(filepath.Join(dir, "go.mod")); statErr == nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", ErrNoRuleRoot
		}
		dir = parent
	}
}

// ConfigPath resolves a rule-tree-relative path against the rule root.
func ConfigPath(parts ...string) (string, error) {
	root, err := RuleRoot()
	if err != nil {
		return "", err
	}
	return filepath.Join(append([]string{root}, parts...)...), nil
}
