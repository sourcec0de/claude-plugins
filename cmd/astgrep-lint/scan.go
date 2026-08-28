package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/sourcec0de/claude-plugins/hookio"
)

var (
	goExtMatch = regexp.MustCompile(`\.go$`)
	tsExtMatch = regexp.MustCompile(`\.tsx?$|\.jsx?$`)

	ErrAstGrepNotFound  = errors.New("ast-grep not found on PATH")
	ErrCreateTempFile   = errors.New("failed to create temp file")
	ErrWriteTempFile    = errors.New("failed to write temp file")
	ErrAstGrepRun       = errors.New("running ast-grep")
	ErrAstGrepJSONParse = errors.New("parsing ast-grep json line")
)

// Violation is one rule match, with the file path mapped back from the scratch
// file ast-grep actually read to the path the model is editing.
type Violation struct {
	RuleID   string
	Text     string
	Line     int
	File     string
	Message  string
	Severity string
}

type astGrepMatch struct {
	RuleID   string `json:"ruleId"`
	Text     string `json:"text"`
	File     string `json:"file"`
	Message  string `json:"message"`
	Severity string `json:"severity"`
	Range    struct {
		Start struct {
			Line int `json:"line"`
		} `json:"start"`
	} `json:"range"`
}

type scanParams struct {
	Cwd      string
	FilePath string
	Content  string
}

type violationDelta struct {
	Before []Violation
	After  []Violation
}

func isLintable(filePath string) bool {
	return goExtMatch.MatchString(filePath) || tsExtMatch.MatchString(filePath)
}

// sgconfigPath resolves the plugin's own sgconfig.yml.
//
// Passing --config explicitly is mandatory. Left to its own devices ast-grep
// searches the working directory and walks upward, which in a hook means it
// would silently lint the model's edits against the user's project rules
// instead of the plugin's.
func sgconfigPath() (string, error) {
	return hookio.ConfigPath("astgrep", "sgconfig.yml")
}

func scanExisting(event hookio.Event) ([]Violation, string, error) {
	existing, err := os.ReadFile(event.ToolInput.FilePath)
	if errors.Is(err, os.ErrNotExist) {
		return nil, "", nil
	}
	if err != nil {
		return nil, "", err
	}
	if len(existing) == 0 {
		return nil, "", nil
	}
	content := string(existing)
	violations, err := scanViolations(scanParams{
		Cwd:      event.Cwd,
		FilePath: event.ToolInput.FilePath,
		Content:  content,
	})
	return violations, content, err
}

// scanViolations runs every rule that applies to the file. The text rules
// apply to all of them, so a Markdown or YAML edit is still checked for
// attribution; the ast-grep rules only apply to the languages that have them.
func scanViolations(p scanParams) ([]Violation, error) {
	violations := scanText(p)
	if !isLintable(p.FilePath) {
		return violations, nil
	}
	parsed, err := scanAstGrep(p)
	if err != nil {
		return nil, err
	}
	return append(violations, parsed...), nil
}

// scanAstGrep runs the plugin's rules over content. The content is written
// to a scratch file that keeps the original extension, so ast-grep selects the
// same parser it would for the real file.
func scanAstGrep(p scanParams) ([]Violation, error) {
	binPath, err := exec.LookPath("ast-grep")
	if err != nil {
		return nil, errors.Join(ErrAstGrepNotFound, err)
	}
	configPath, err := sgconfigPath()
	if err != nil {
		return nil, err
	}

	tmp, err := os.CreateTemp("", fmt.Sprintf("astgrep-*-%s", filepath.Base(p.FilePath)))
	if err != nil {
		return nil, errors.Join(ErrCreateTempFile, err)
	}
	defer os.Remove(tmp.Name())

	if _, err := tmp.WriteString(p.Content); err != nil {
		tmp.Close()
		return nil, errors.Join(ErrWriteTempFile, err)
	}
	tmp.Close()

	stdout, err := runAstGrep(binPath, configPath, p.Cwd, tmp.Name())
	if err != nil {
		return nil, err
	}
	return parseMatches(stdout, tmp.Name(), p.FilePath)
}

func runAstGrep(binPath, configPath, cwd, target string) ([]byte, error) {
	cmd := &exec.Cmd{
		Dir:  cwd,
		Path: binPath,
		Args: []string{"ast-grep", "scan", "--config", configPath, "--json=stream", target},
	}
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	// A nonzero exit is how ast-grep reports that it found error-severity
	// matches, so only a failure to start is a real error.
	if runErr := cmd.Run(); runErr != nil && !isExpectedExit(runErr) {
		return nil, errors.Join(ErrAstGrepRun, runErr, errors.New(stderr.String()))
	}
	return stdout.Bytes(), nil
}

func parseMatches(data []byte, scratchPath, realPath string) ([]Violation, error) {
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
			return nil, errors.Join(ErrAstGrepJSONParse, err)
		}
		if !isBlocking(m.Severity) {
			continue
		}
		out = append(out, Violation{
			RuleID:   m.RuleID,
			Text:     m.Text,
			Line:     m.Range.Start.Line + 1,
			File:     resolveFile(m.File, scratchPath, realPath),
			Message:  m.Message,
			Severity: m.Severity,
		})
	}
	return out, nil
}

// isBlocking keeps only error-severity matches. Warnings stay informational so
// new rules can be rolled out without a wave of false rejections.
func isBlocking(severity string) bool {
	return severity == "" || strings.EqualFold(severity, "error")
}

func resolveFile(reported, scratchPath, realPath string) string {
	if reported == scratchPath {
		return realPath
	}
	return reported
}

func isExpectedExit(err error) bool {
	var exitErr *exec.ExitError
	return errors.As(err, &exitErr)
}

func diffViolations(d violationDelta) []Violation {
	beforeCounts := make(map[string]int, len(d.Before))
	for _, v := range d.Before {
		beforeCounts[violationKey(v)]++
	}
	var out []Violation
	afterSeen := make(map[string]int, len(d.After))
	for _, v := range d.After {
		key := violationKey(v)
		afterSeen[key]++
		if afterSeen[key] > beforeCounts[key] {
			out = append(out, v)
		}
	}
	return out
}

// violationKey identifies a violation by rule and matched text rather than by
// line, so that shifting a pre-existing violation up or down the file does not
// make it look newly introduced.
func violationKey(v Violation) string {
	return fmt.Sprintf("%s\x00%s", v.RuleID, v.Text)
}
