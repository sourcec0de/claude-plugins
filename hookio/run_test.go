package hookio_test

import (
	"path/filepath"
	"testing"

	"github.com/sourcec0de/claude-plugins/hookio"
	"github.com/stretchr/testify/require"
)

func TestRuleRootPrefersPluginRoot(t *testing.T) {
	t.Setenv("CLAUDE_PLUGIN_ROOT", "/plugins/cache/astgrep-lint")
	t.Setenv("CLAUDE_PLUGINS_RULE_ROOT", "/nix/store/x/share/claude-plugins")

	got, err := hookio.RuleRoot()
	require.NoError(t, err)
	require.Equal(t, "/plugins/cache/astgrep-lint", got,
		"a plugin update must be able to ship rules the installed binary predates")
}

func TestRuleRootFallsBackToPackagedRules(t *testing.T) {
	t.Setenv("CLAUDE_PLUGIN_ROOT", "")
	t.Setenv("CLAUDE_PLUGINS_RULE_ROOT", "/nix/store/x/share/claude-plugins")

	got, err := hookio.RuleRoot()
	require.NoError(t, err)
	require.Equal(t, "/nix/store/x/share/claude-plugins", got,
		"the binaries must work standalone, outside a hook")
}

func TestRuleRootFallsBackToModuleRoot(t *testing.T) {
	t.Setenv("CLAUDE_PLUGIN_ROOT", "")
	t.Setenv("CLAUDE_PLUGINS_RULE_ROOT", "")

	got, err := hookio.RuleRoot()
	require.NoError(t, err)
	require.FileExists(t, filepath.Join(got, "go.mod"))
}

func TestConfigPathJoinsUnderRoot(t *testing.T) {
	t.Setenv("CLAUDE_PLUGIN_ROOT", "/plugins/cache/astgrep-lint")
	t.Setenv("CLAUDE_PLUGINS_RULE_ROOT", "")

	got, err := hookio.ConfigPath("astgrep", "sgconfig.yml")
	require.NoError(t, err)
	require.Equal(t, "/plugins/cache/astgrep-lint/astgrep/sgconfig.yml", got)
}
