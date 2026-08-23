package main

import (
	"fmt"
	"os"
)

// runCLI scans the named files and reports every violation found, not just the
// newly introduced ones. This is the mode reached by invoking astgrep-lint as a
// bare command, which lets work be checked before it is written.
func runCLI(paths []string) int {
	// The SessionStart hook passes --warm purely to trigger the wrapper's
	// build-if-stale step, so the first real edit of the session is not slowed
	// down by a compile. Reaching this point means the build already happened.
	if len(paths) == 1 && paths[0] == "--warm" {
		return 0
	}
	failed := false
	for _, path := range paths {
		violations, err := scanFile(path)
		if err != nil {
			fmt.Fprintf(os.Stderr, "astgrep-lint: %s: %v\n", path, err)
			failed = true
			continue
		}
		if len(violations) == 0 {
			continue
		}
		fmt.Fprintln(os.Stderr, formatViolations(violations))
		failed = true
	}
	if failed {
		return 1
	}
	return 0
}

func scanFile(path string) ([]Violation, error) {
	if !isLintable(path) {
		return nil, nil
	}
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	violations, err := scanViolations(scanParams{FilePath: path, Content: string(content)})
	if err != nil {
		return nil, err
	}
	return filterSuppressed(violations, string(content)), nil
}
