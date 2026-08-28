package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"strings"

	"github.com/sourcec0de/claude-plugins/hookio"
)

var (
	ErrAstGrepNotFound = errors.New("ast-grep not found on PATH")
	ErrScratchFile     = errors.New("failed to write scratch file")
	ErrAstGrepJSON     = errors.New("parsing ast-grep json line")
)

// Violation is one shell rule match. Bash commands are single-line in practice,
// so no line number is reported.
type Violation struct {
	RuleID   string
	Text     string
	Message  string
	Severity string
}

type astGrepMatch struct {
	RuleID   string `json:"ruleId"`
	Text     string `json:"text"`
	Message  string `json:"message"`
	Severity string `json:"severity"`
}

// scanParams selects which rules to run and what to run them against.
type scanParams struct {
	RuleTree string
	Cwd      string
	Command  string
}

// sgconfigPath resolves a rule tree's own shell rules. As with astgrep-lint,
// passing --config explicitly keeps ast-grep from walking up from the user's
// working directory and finding their project config instead.
func sgconfigPath(ruleTree string) (string, error) {
	return hookio.ConfigPath(ruleTree, "sgconfig.yml")
}

func scanCommand(p scanParams) ([]Violation, error) {
	binPath, err := exec.LookPath("ast-grep")
	if err != nil {
		return nil, errors.Join(ErrAstGrepNotFound, err)
	}
	configPath, err := sgconfigPath(p.RuleTree)
	if err != nil {
		return nil, err
	}

	tmp, err := os.CreateTemp("", "bashguard-*.sh")
	if err != nil {
		return nil, errors.Join(ErrScratchFile, err)
	}
	defer os.Remove(tmp.Name())

	if _, err := tmp.WriteString(p.Command); err != nil {
		tmp.Close()
		return nil, errors.Join(ErrScratchFile, err)
	}
	tmp.Close()

	cmd := &exec.Cmd{
		Dir:  p.Cwd,
		Path: binPath,
		Args: []string{"ast-grep", "scan", "--config", configPath, "--json=stream", tmp.Name()},
	}
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	// A nonzero exit means error-severity matches were found, which is the
	// expected outcome here rather than a failure. Anything else carries
	// ast-grep's own diagnostic, which is the only clue a rule is malformed.
	if runErr := cmd.Run(); runErr != nil && !isExpectedExit(runErr) {
		return nil, errors.Join(runErr, errors.New(stderr.String()))
	}
	return parseMatches(stdout.Bytes())
}

func parseMatches(data []byte) ([]Violation, error) {
	if len(bytes.TrimSpace(data)) == 0 {
		return nil, nil
	}
	var out []Violation
	for _, line := range bytes.Split(data, []byte("\n")) {
		if len(bytes.TrimSpace(line)) == 0 {
			continue
		}
		var m astGrepMatch
		if err := json.Unmarshal(line, &m); err != nil {
			return nil, errors.Join(ErrAstGrepJSON, err)
		}
		if !isBlocking(m.Severity) {
			continue
		}
		out = append(out, Violation{
			RuleID:   m.RuleID,
			Text:     m.Text,
			Message:  m.Message,
			Severity: m.Severity,
		})
	}
	return out, nil
}

func isBlocking(severity string) bool {
	return severity == "" || strings.EqualFold(severity, "error")
}

func isExpectedExit(err error) bool {
	var exitErr *exec.ExitError
	return errors.As(err, &exitErr)
}

func formatViolations(violations []Violation) string {
	if len(violations) == 0 {
		return ""
	}
	var sb strings.Builder
	for i, v := range violations {
		if i > 0 {
			sb.WriteByte('\n')
		}
		sb.WriteString(v.RuleID)
		sb.WriteString(": ")
		sb.WriteString(firstLine(v.Text))
		sb.WriteByte('\n')
		sb.WriteString("  ")
		sb.WriteString(normalizeMessage(v.Message))
	}
	return sb.String()
}

func firstLine(s string) string {
	if idx := strings.IndexByte(s, '\n'); idx >= 0 {
		return s[:idx]
	}
	return s
}

func normalizeMessage(s string) string {
	return strings.Join(strings.Fields(strings.TrimSpace(s)), " ")
}
