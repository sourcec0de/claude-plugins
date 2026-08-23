package hookio_test

import (
	"path/filepath"
	"testing"

	"github.com/sourcec0de/claude-plugins/hookio"
	"github.com/stretchr/testify/require"
)

func TestPluginRootPrefersEnvironment(t *testing.T) {
	t.Setenv("CLAUDE_PLUGIN_ROOT", "/plugins/cache/astgrep-lint")

	got, err := hookio.PluginRoot()
	require.NoError(t, err)
	require.Equal(t, "/plugins/cache/astgrep-lint", got)
}

func TestPluginRootFallsBackToModuleRoot(t *testing.T) {
	t.Setenv("CLAUDE_PLUGIN_ROOT", "")

	got, err := hookio.PluginRoot()
	require.NoError(t, err)
	require.FileExists(t, filepath.Join(got, "go.mod"))
}

func TestConfigPathJoinsUnderRoot(t *testing.T) {
	t.Setenv("CLAUDE_PLUGIN_ROOT", "/plugins/cache/astgrep-lint")

	got, err := hookio.ConfigPath("astgrep", "sgconfig.yml")
	require.NoError(t, err)
	require.Equal(t, "/plugins/cache/astgrep-lint/astgrep/sgconfig.yml", got)
}
