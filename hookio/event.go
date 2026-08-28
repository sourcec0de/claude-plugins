// Package hookio marshals Claude Code hook payloads in and hook decisions out.
//
// The wire contract carries one documented ambiguity: whether the runtime reads
// stdout JSON on every exit code, or only on exit 0. Decision resolves this by
// making the two outcomes mutually exclusive, so neither reading changes
// observable behaviour. See decision.go.
package hookio

import (
	"encoding/json"
	"fmt"
	"io"
)

// Event is the JSON object Claude Code writes to a hook's stdin.
type Event struct {
	SessionID      string       `json:"session_id"`
	TranscriptPath string       `json:"transcript_path"`
	Cwd            string       `json:"cwd"`
	PermissionMode string       `json:"permission_mode"`
	HookEventName  string       `json:"hook_event_name"`
	AgentID        string       `json:"agent_id,omitempty"`
	AgentType      string       `json:"agent_type,omitempty"`
	ToolName       string       `json:"tool_name,omitempty"`
	ToolInput      ToolInput    `json:"tool_input,omitempty"`
	ToolResponse   ToolResponse `json:"tool_response,omitempty"`
	ToolUseID      string       `json:"tool_use_id,omitempty"`
}

// ToolInput holds the arguments of the tool call that fired the hook. The
// fields are the union across Write, Edit and Bash; only those belonging to the
// firing tool are populated.
type ToolInput struct {
	FilePath   string `json:"file_path,omitempty"`
	Content    string `json:"content,omitempty"`
	OldString  string `json:"old_string,omitempty"`
	NewString  string `json:"new_string,omitempty"`
	ReplaceAll bool   `json:"replace_all,omitempty"`
	Command    string `json:"command,omitempty"`
}

// ToolResponse holds the tool result. It is populated on PostToolUse only.
type ToolResponse struct {
	FilePath string `json:"filePath,omitempty"`
	Success  bool   `json:"success,omitempty"`
}

// DecodeEvent parses a hook payload read from r.
func DecodeEvent(r io.Reader) (Event, error) {
	raw, err := io.ReadAll(r)
	if err != nil {
		return Event{}, fmt.Errorf("reading hook input: %w", err)
	}
	var e Event
	if err := json.Unmarshal(raw, &e); err != nil {
		return Event{}, fmt.Errorf("parsing hook input: %w", err)
	}
	return e, nil
}
