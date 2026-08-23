package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

// The plugin is copied into ~/.claude/plugins/cache and the hook runs with the
// user's project as its working directory. Left to itself ast-grep would search
// upward from there and lint against the project's rules, so these tests pin
// the config to the plugin root instead.

func TestSgconfigPathResolvesUnderPluginRoot(t *testing.T) {
	t.Setenv("CLAUDE_PLUGIN_ROOT", "/home/u/.claude/plugins/cache/astgrep-lint")

	got, err := sgconfigPath()
	require.NoError(t, err)
	require.Equal(t, "/home/u/.claude/plugins/cache/astgrep-lint/astgrep/sgconfig.yml", got)
}

func TestScanIgnoresProjectSgconfigInWorkingDirectory(t *testing.T) {
	pluginRoot, err := os.Getwd()
	require.NoError(t, err)
	pluginRoot = filepath.Join(pluginRoot, "..", "..")

	// A decoy project config whose only rule would fire on the sample below.
	project := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(project, "rules"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(project, "sgconfig.yml"),
		[]byte("ruleDirs:\n  - rules\n"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(project, "rules", "decoy.yml"), []byte(
		"id: project-decoy-rule\nlanguage: go\nseverity: error\nmessage: decoy\nrule:\n  pattern: return \"ok\"\n"), 0o644))

	t.Setenv("CLAUDE_PLUGIN_ROOT", pluginRoot)

	content := "package clean\n\nfunc Ok() string {\n\treturn \"ok\"\n}\n"
	got, err := scanViolations(scanParams{Cwd: project, FilePath: "clean.go", Content: content})
	require.NoError(t, err)

	for _, v := range got {
		require.NotEqual(t, "project-decoy-rule", v.RuleID,
			"the user's project rules must never be applied by the plugin: %+v", got)
	}
	require.Empty(t, got, "clean content must produce no violations under the plugin's own rules")
}
