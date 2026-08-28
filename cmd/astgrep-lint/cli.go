package main

import (
	"fmt"
	"os"
)

// runCLI scans the named files and reports every violation found, not just the
// newly introduced ones. This is the mode reached by invoking astgrep-lint as a
// bare command, which lets work be checked before it is written. Files of a
// language with no rules still go through the text rules.
func runCLI(paths []string) int {
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
