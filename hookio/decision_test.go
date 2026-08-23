package hookio_test

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/sourcec0de/claude-plugins/hookio"
	"github.com/stretchr/testify/require"
)

type emitted struct {
	Code   int
	Stdout string
	Stderr string
}

func emit(d hookio.Decision) emitted {
	var stdout, stderr bytes.Buffer
	code := d.Emit(&stdout, &stderr)
	return emitted{Code: code, Stdout: stdout.String(), Stderr: stderr.String()}
}

func TestNoopWritesNothing(t *testing.T) {
	t.Parallel()

	got := emit(hookio.Noop())
	require.Equal(t, 0, got.Code)
	require.Empty(t, got.Stdout)
	require.Empty(t, got.Stderr)
}

func TestDenyIsStructuredOnExitZero(t *testing.T) {
	t.Parallel()

	got := emit(hookio.Deny("PreToolUse", "no-fmt-println: banned"))
	require.Equal(t, 0, got.Code)
	require.Empty(t, got.Stderr, "a structured decision must not write to stderr")

	var payload struct {
		HookSpecificOutput struct {
			HookEventName            string `json:"hookEventName"`
			PermissionDecision       string `json:"permissionDecision"`
			PermissionDecisionReason string `json:"permissionDecisionReason"`
		} `json:"hookSpecificOutput"`
	}
	require.NoError(t, json.Unmarshal([]byte(got.Stdout), &payload))
	require.Equal(t, "PreToolUse", payload.HookSpecificOutput.HookEventName)
	require.Equal(t, "deny", payload.HookSpecificOutput.PermissionDecision)
	require.Equal(t, "no-fmt-println: banned", payload.HookSpecificOutput.PermissionDecisionReason)
}

func TestAllowIsStructuredOnExitZero(t *testing.T) {
	t.Parallel()

	got := emit(hookio.Allow("PreToolUse", "clean"))
	require.Equal(t, 0, got.Code)
	require.Empty(t, got.Stderr)
	require.Contains(t, got.Stdout, `"permissionDecision":"allow"`)
}

func TestContextCarriesAdditionalContext(t *testing.T) {
	t.Parallel()

	got := emit(hookio.Context("PostToolUse", "gofmt rewrote the file"))
	require.Equal(t, 0, got.Code)
	require.Empty(t, got.Stderr)
	require.Contains(t, got.Stdout, `"additionalContext":"gofmt rewrote the file"`)
	require.NotContains(t, got.Stdout, "permissionDecision")
}

func TestBlockIsExitTwoOnStderrOnly(t *testing.T) {
	t.Parallel()

	got := emit(hookio.Block("rejected: three violations"))
	require.Equal(t, 2, got.Code)
	require.Empty(t, got.Stdout, "a block must leave stdout empty so JSON can never contradict it")
	require.Equal(t, "rejected: three violations\n", got.Stderr)
}

func TestOutcomesAreDisjoint(t *testing.T) {
	t.Parallel()

	cases := map[string]hookio.Decision{
		"noop":    hookio.Noop(),
		"allow":   hookio.Allow("PreToolUse", "ok"),
		"deny":    hookio.Deny("PreToolUse", "nope"),
		"context": hookio.Context("PostToolUse", "fyi"),
		"block":   hookio.Block("nope"),
	}
	for name, decision := range cases {
		t.Run(name, func(t *testing.T) {
			got := emit(decision)
			require.False(t, got.Stdout != "" && got.Stderr != "",
				"a decision must never write to both streams")
			require.False(t, got.Code == 2 && got.Stdout != "",
				"exit 2 must never be accompanied by stdout JSON")
		})
	}
}

func TestDecodeEvent(t *testing.T) {
	t.Parallel()

	raw := `{
	  "session_id": "s1",
	  "cwd": "/repo",
	  "hook_event_name": "PreToolUse",
	  "tool_name": "Edit",
	  "tool_input": {"file_path": "/repo/a.go", "old_string": "x", "new_string": "y"}
	}`
	event, err := hookio.DecodeEvent(strings.NewReader(raw))
	require.NoError(t, err)
	require.Equal(t, "PreToolUse", event.HookEventName)
	require.Equal(t, "Edit", event.ToolName)
	require.Equal(t, "/repo/a.go", event.ToolInput.FilePath)
	require.Equal(t, "y", event.ToolInput.NewString)
}

func TestDecodeEventRejectsGarbage(t *testing.T) {
	t.Parallel()

	_, err := hookio.DecodeEvent(strings.NewReader("not json"))
	require.Error(t, err)
}
