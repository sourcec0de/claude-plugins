package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/sourcec0de/claude-plugins/hookio"
	"github.com/stretchr/testify/require"
)

func TestTargetFilePrefersToolResponse(t *testing.T) {
	t.Parallel()

	got := targetFile(hookio.Event{
		ToolInput:    hookio.ToolInput{FilePath: "/requested.go"},
		ToolResponse: hookio.ToolResponse{FilePath: "/written.go"},
	})
	require.Equal(t, "/written.go", got)
}

func TestTargetFileFallsBackToToolInput(t *testing.T) {
	t.Parallel()

	got := targetFile(hookio.Event{ToolInput: hookio.ToolInput{FilePath: "/requested.go"}})
	require.Equal(t, "/requested.go", got)
}

func TestFormattersForByExtension(t *testing.T) {
	t.Parallel()

	require.Len(t, formattersFor("", "a.go"), 1)
	require.Equal(t, "gofmt", formattersFor("", "a.go")[0].Name)
	require.Nil(t, formattersFor("", "README.md"))
	require.Nil(t, formattersFor("", ""))
}

// The Node formatters execute the edited project's own eslint/prettier config,
// so they must key off the directory the edit landed in rather than wherever
// the hook process happens to be running.

func TestNodeFormattersSkipDirectoryWithoutLockfile(t *testing.T) {
	t.Parallel()

	require.Nil(t, nodeFormatters(t.TempDir()),
		"a directory that is not a Node project must not invoke a package runner")
}

func TestNodeFormattersSkipUninstalledProject(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "pnpm-lock.yaml"), []byte("lockfileVersion: 9\n"), 0o644))

	require.Empty(t, nodeFormatters(dir),
		"a lockfile alone must not cause the runner to fetch and execute a tool")
}

func TestHasLockfileResolvesAgainstEditDirectory(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	require.False(t, hasLockfile(dir, "pnpm"))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "pnpm-lock.yaml"), []byte(""), 0o644))
	require.True(t, hasLockfile(dir, "pnpm"))
	require.False(t, hasLockfile(dir, "yarn"))
	require.False(t, hasLockfile(dir, "unknown-runner"))
}

func TestInstalledLocally(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	require.False(t, installedLocally(dir, "eslint"))
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "node_modules", ".bin"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "node_modules", ".bin", "eslint"), []byte("#!/bin/sh\n"), 0o755))
	require.True(t, installedLocally(dir, "eslint"))
	require.False(t, installedLocally(dir, "prettier"))
}

func TestRunnerArgsNpmUsesDoubleDash(t *testing.T) {
	t.Parallel()

	require.Equal(t, []string{"exec", "--", "prettier", "--write"}, runnerArgs("npm", "prettier", "--write"))
	require.Equal(t, []string{"exec", "prettier", "--write"}, runnerArgs("pnpm", "prettier", "--write"))
}

func TestDecideIgnoresUnknownExtension(t *testing.T) {
	t.Parallel()

	got := decide(hookio.Event{
		HookEventName: "PostToolUse",
		ToolInput:     hookio.ToolInput{FilePath: "notes.md"},
	})
	require.True(t, got.IsNoop())
}

func TestDecideFormatsGoFileInPlace(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "messy.go")
	require.NoError(t, os.WriteFile(path, []byte("package m\nfunc  A( ) {\n}\n"), 0o644))

	got := decide(hookio.Event{
		HookEventName: "PostToolUse",
		Cwd:           dir,
		ToolInput:     hookio.ToolInput{FilePath: path},
	})
	require.True(t, got.IsNoop(), "a clean format must not report anything: %s", got.Reason())

	formatted, err := os.ReadFile(path)
	require.NoError(t, err)
	require.Equal(t, "package m\n\nfunc A() {\n}\n", string(formatted))
}

func TestDecideReportsFormatterFailure(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "broken.go")
	require.NoError(t, os.WriteFile(path, []byte("package m\nfunc ( {{{\n"), 0o644))

	got := decide(hookio.Event{
		HookEventName: "PostToolUse",
		Cwd:           dir,
		ToolInput:     hookio.ToolInput{FilePath: path},
	})
	require.False(t, got.IsNoop(), "a failing formatter must be reported")
	require.Contains(t, got.Reason(), "gofmt")
	require.Empty(t, got.PermissionDecision(), "autofmt must never deny a tool call")
}

func TestMissingFormatterIsSkippedSilently(t *testing.T) {
	t.Setenv("PATH", "/nonexistent")

	dir := t.TempDir()
	path := filepath.Join(dir, "x.go")
	require.NoError(t, os.WriteFile(path, []byte("package m\n"), 0o644))

	got := decide(hookio.Event{
		HookEventName: "PostToolUse",
		Cwd:           dir,
		ToolInput:     hookio.ToolInput{FilePath: path},
	})
	require.True(t, got.IsNoop(), "an absent formatter is not an error")
}
