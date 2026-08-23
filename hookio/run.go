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

// ErrNoPluginRoot reports that the plugin directory could not be located.
var ErrNoPluginRoot = fmt.Errorf("cannot locate plugin root: CLAUDE_PLUGIN_ROOT unset and no go.mod found above the working directory")

// PluginRoot returns the directory the plugin was installed to.
//
// Claude Code exports CLAUDE_PLUGIN_ROOT when it spawns a hook. Resolving
// config paths against it is mandatory: the plugin is copied into
// ~/.claude/plugins/cache, and any tool that searches the working directory
// instead would silently pick up the user's own project config.
//
// The go.mod walk is the fallback for `go test` and for bare CLI use from a
// checkout, where the variable is absent.
func PluginRoot() (string, error) {
	if root := os.Getenv("CLAUDE_PLUGIN_ROOT"); root != "" {
		return root, nil
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
			return "", ErrNoPluginRoot
		}
		dir = parent
	}
}

// ConfigPath resolves a plugin-relative path against the plugin root.
func ConfigPath(parts ...string) (string, error) {
	root, err := PluginRoot()
	if err != nil {
		return "", err
	}
	return filepath.Join(append([]string{root}, parts...)...), nil
}
