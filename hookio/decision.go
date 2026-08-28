package hookio

import (
	"encoding/json"
	"fmt"
	"io"
)

// PermissionDecision is the verdict a PreToolUse hook returns to the runtime.
type PermissionDecision string

const (
	PermissionAllow PermissionDecision = "allow"
	PermissionDeny  PermissionDecision = "deny"
)

type outcome int

const (
	outcomeNoop outcome = iota
	outcomeStructured
	outcomeBlock
)

// Decision is the result of a hook handler.
//
// Exactly one of two wire forms is emitted, never both:
//
//	noop, allow, deny, context -> exit 0, JSON on stdout (nothing on stderr)
//	block                      -> exit 2, plain reason on stderr (nothing on stdout)
//
// The official reference says stdout JSON is read on every exit code, with
// exit 2's block being the one outcome JSON cannot override; third-party
// writeups claim JSON is parsed only on exit 0. Keeping the forms disjoint
// makes the behaviour identical under either reading.
type Decision struct {
	outcome outcome
	reason  string
	output  *hookOutput
}

type hookOutput struct {
	HookSpecificOutput *hookSpecificOutput `json:"hookSpecificOutput,omitempty"`
}

type hookSpecificOutput struct {
	HookEventName            string             `json:"hookEventName"`
	PermissionDecision       PermissionDecision `json:"permissionDecision,omitempty"`
	PermissionDecisionReason string             `json:"permissionDecisionReason,omitempty"`
	AdditionalContext        string             `json:"additionalContext,omitempty"`
}

// Noop lets the tool call proceed without emitting any output.
func Noop() Decision {
	return Decision{outcome: outcomeNoop}
}

// Allow explicitly permits the tool call, bypassing the permission prompt.
func Allow(event, reason string) Decision {
	return structured(&hookSpecificOutput{
		HookEventName:            event,
		PermissionDecision:       PermissionAllow,
		PermissionDecisionReason: reason,
	})
}

// Deny rejects the tool call before it runs and shows reason to the model.
// This is the PreToolUse rejection path: a structured verdict on exit 0.
func Deny(event, reason string) Decision {
	return structured(&hookSpecificOutput{
		HookEventName:            event,
		PermissionDecision:       PermissionDeny,
		PermissionDecisionReason: reason,
	})
}

// Context feeds an advisory message back to the model without changing whether
// the tool call proceeds.
func Context(event, message string) Decision {
	return structured(&hookSpecificOutput{
		HookEventName:     event,
		AdditionalContext: message,
	})
}

// Block halts the tool call via exit 2 with reason on stderr. It is the
// PostToolUse rejection path, where no structured verdict field exists.
func Block(reason string) Decision {
	return Decision{outcome: outcomeBlock, reason: reason}
}

func structured(hs *hookSpecificOutput) Decision {
	return Decision{outcome: outcomeStructured, output: &hookOutput{HookSpecificOutput: hs}}
}

// Emit writes the decision to the given streams and returns the exit code.
// It upholds the disjointness invariant: a block writes only to stderr, every
// other outcome writes only to stdout.
func (d Decision) Emit(stdout, stderr io.Writer) int {
	if d.outcome == outcomeBlock {
		fmt.Fprintln(stderr, d.reason)
		return 2
	}
	if d.outcome == outcomeNoop {
		return 0
	}
	if err := json.NewEncoder(stdout).Encode(d.output); err != nil {
		fmt.Fprintf(stderr, "hookio: encoding decision: %v\n", err)
		return 1
	}
	return 0
}

// IsNoop reports whether the decision leaves the tool call untouched.
func (d Decision) IsNoop() bool {
	return d.outcome == outcomeNoop
}

// PermissionDecision returns the verdict a structured decision carries, or the
// empty string for a noop, a context message or a block.
func (d Decision) PermissionDecision() PermissionDecision {
	if d.outcome != outcomeStructured || d.output.HookSpecificOutput == nil {
		return ""
	}
	return d.output.HookSpecificOutput.PermissionDecision
}

// Reason returns the human-readable explanation attached to the decision: the
// permission reason for a structured verdict, the stderr text for a block, and
// the empty string for a noop.
func (d Decision) Reason() string {
	if d.outcome == outcomeBlock {
		return d.reason
	}
	if d.outcome != outcomeStructured || d.output.HookSpecificOutput == nil {
		return ""
	}
	hs := d.output.HookSpecificOutput
	if hs.PermissionDecisionReason != "" {
		return hs.PermissionDecisionReason
	}
	return hs.AdditionalContext
}

// ExitCode returns the status the decision exits with.
func (d Decision) ExitCode() int {
	if d.outcome == outcomeBlock {
		return 2
	}
	return 0
}
