// Command autofmt formats a file after the model writes it.
//
// It runs as a PostToolUse hook. Formatting is a convenience rather than a
// gate, so a missing formatter is skipped silently; only a formatter that runs
// and fails is reported back to the model.
package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/sourcec0de/claude-plugins/hookio"
)

func main() {
	hookio.Run(decide)
}

func decide(event hookio.Event) hookio.Decision {
	filePath := targetFile(event)
	if filePath == "" {
		return hookio.Noop()
	}

	var problems []string
	for _, f := range formattersFor(event.Cwd, filePath) {
		if err := f.run(event.Cwd, filePath); err != nil {
			problems = append(problems, err.Error())
		}
	}
	if len(problems) == 0 {
		return hookio.Noop()
	}
	return hookio.Context(event.HookEventName, fmt.Sprintf(
		"Formatting %s reported:\n%s", filePath, strings.Join(problems, "\n")))
}

// targetFile prefers the path the tool reported writing, falling back to the
// requested path when the response carries none.
func targetFile(event hookio.Event) string {
	if event.ToolResponse.FilePath != "" {
		return event.ToolResponse.FilePath
	}
	return event.ToolInput.FilePath
}

// formatter is one formatting command, described so that availability can be
// resolved separately from execution.
type formatter struct {
	Name string
	Bin  string
	Args []string
}

func (f formatter) run(cwd, filePath string) error {
	binPath, err := exec.LookPath(f.Bin)
	if err != nil {
		return nil
	}
	cmd := &exec.Cmd{
		Dir:  cwd,
		Path: binPath,
		Args: append([]string{f.Bin}, append(f.Args, filePath)...),
	}
	output, runErr := cmd.CombinedOutput()
	if runErr == nil {
		return nil
	}
	return fmt.Errorf("%s: %v\n%s", f.Name, runErr, strings.TrimSpace(string(output)))
}

func formattersFor(cwd, filePath string) []formatter {
	switch {
	case goExtMatch.MatchString(filePath):
		return []formatter{{Name: "gofmt", Bin: "gofmt", Args: []string{"-w"}}}
	case tsExtMatch.MatchString(filePath):
		return nodeFormatters(cwd)
	default:
		return nil
	}
}

// nodeFormatters resolves the project's package runner rather than assuming
// one. The original implementation hardcoded pnpm, which is correct only in the
// repository it came from.
//
// ESLint and prettier load configuration and plugins from the project being
// formatted, which means running them executes that project's JavaScript. The
// gates below keep that from happening in a directory the user merely has
// checked out: the runner's lockfile and the tool itself must both be present
// in the directory the edit landed in.
func nodeFormatters(cwd string) []formatter {
	runner := detectRunner(cwd)
	if runner == "" {
		return nil
	}
	var out []formatter
	for _, tool := range []struct{ name, flag string }{
		{"eslint", "--fix"},
		{"prettier", "--write"},
	} {
		if !installedLocally(cwd, tool.name) {
			continue
		}
		out = append(out, formatter{
			Name: tool.name,
			Bin:  runner,
			Args: runnerArgs(runner, tool.name, tool.flag),
		})
	}
	return out
}

func detectRunner(cwd string) string {
	for _, candidate := range []string{"pnpm", "yarn", "npm", "bun"} {
		if _, err := exec.LookPath(candidate); err != nil {
			continue
		}
		if hasLockfile(cwd, candidate) {
			return candidate
		}
	}
	return ""
}

// hasLockfile keeps autofmt from running a package runner in a directory that
// is not actually a Node project of that flavour. The path is resolved against
// the directory the edit happened in, not the hook process's own working
// directory, which are not always the same.
func hasLockfile(cwd, runner string) bool {
	lockfiles := map[string]string{
		"pnpm": "pnpm-lock.yaml",
		"yarn": "yarn.lock",
		"npm":  "package-lock.json",
		"bun":  "bun.lockb",
	}
	name, ok := lockfiles[runner]
	if !ok {
		return false
	}
	_, err := os.Stat(filepath.Join(cwd, name))
	return err == nil
}

// installedLocally reports whether the project has actually installed the tool,
// so a bare lockfile in an uninstalled checkout does not cause the runner to
// fetch and execute one.
func installedLocally(cwd, tool string) bool {
	_, err := os.Stat(filepath.Join(cwd, "node_modules", ".bin", tool))
	return err == nil
}

func runnerArgs(runner, tool, flag string) []string {
	if runner == "npm" {
		return []string{"exec", "--", tool, flag}
	}
	return []string{"exec", tool, flag}
}
