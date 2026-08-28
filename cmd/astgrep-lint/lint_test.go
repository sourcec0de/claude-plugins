package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sourcec0de/claude-plugins/hookio"
	"github.com/stretchr/testify/require"
)

type decideCase struct {
	InitialContent string
	FileName       string
	OldString      string
	NewString      string
	WriteContent   string
}

func runDecide(t *testing.T, tc decideCase) hookio.Decision {
	t.Helper()
	dir := t.TempDir()
	fullPath := filepath.Join(dir, tc.FileName)
	if tc.InitialContent != "" {
		require.NoError(t, os.WriteFile(fullPath, []byte(tc.InitialContent), 0o644))
	}
	return decide(hookio.Event{
		HookEventName: "PreToolUse",
		ToolInput: hookio.ToolInput{
			FilePath:  fullPath,
			OldString: tc.OldString,
			NewString: tc.NewString,
			Content:   tc.WriteContent,
		},
	})
}

func TestScanViolationsCleanGoContent(t *testing.T) {
	t.Parallel()

	content := `package testdata

func Clean() string {
	return "ok"
}
`
	got, err := scanViolations(scanParams{FilePath: "clean.go", Content: content})
	require.NoError(t, err)
	require.Empty(t, got)
}

func TestScanViolationsDetectsFmtPrintln(t *testing.T) {
	t.Parallel()

	content := `package testdata

import "fmt"

func Leak() {
	fmt.Println("debug")
}
`
	got, err := scanViolations(scanParams{FilePath: "leak.go", Content: content})
	require.NoError(t, err)
	require.Len(t, got, 1)
	require.Equal(t, "no-fmt-println", got[0].RuleID)
	require.Contains(t, got[0].Text, "fmt.Println")
	require.Equal(t, 6, got[0].Line)
}

func TestScanViolationsPreservesDuplicates(t *testing.T) {
	t.Parallel()

	content := `package testdata

import "fmt"

func Twice() {
	fmt.Println("a")
	fmt.Println("a")
}
`
	got, err := scanViolations(scanParams{FilePath: "twice.go", Content: content})
	require.NoError(t, err)
	require.Len(t, got, 2)
	for _, v := range got {
		require.Equal(t, "no-fmt-println", v.RuleID)
	}
}

func TestScanViolationsDetectsTypeScriptRules(t *testing.T) {
	t.Parallel()

	content := `export async function loads(page: any) {
  await page.getByRole('heading').waitFor()
}
`
	got, err := scanViolations(scanParams{FilePath: "sample.test.ts", Content: content})
	require.NoError(t, err)

	sawWaitFor := false
	for _, v := range got {
		if v.RuleID == "ts-no-waitfor" {
			sawWaitFor = true
		}
	}
	require.True(t, sawWaitFor, "expected ts-no-waitfor violation in: %+v", got)
}

func TestScanViolationsExemptsStoriesInBodyComment(t *testing.T) {
	t.Parallel()

	content := `function StoryDemo(): JSX.Element {
  // narration that documents the story scenario
  const x = build();
  return render(x);
}
`
	got, err := scanViolations(scanParams{FilePath: "Demo.stories.tsx", Content: content})
	require.NoError(t, err)

	for _, v := range got {
		require.NotEqual(t, "ts-no-comment-in-body", v.RuleID,
			"in-body comments in .stories files must be exempt: %+v", got)
	}
}

func TestScanViolationsCatchesInBodyCommentOutsideStories(t *testing.T) {
	t.Parallel()

	content := `function Demo(): JSX.Element {
  // narration that should be caught in a normal component
  const x = build();
  return render(x);
}
`
	got, err := scanViolations(scanParams{FilePath: "Demo.tsx", Content: content})
	require.NoError(t, err)

	sawComment := false
	for _, v := range got {
		if v.RuleID == "ts-no-comment-in-body" {
			sawComment = true
		}
	}
	require.True(t, sawComment, "in-body comment in a non-stories .tsx must still be caught: %+v", got)
}

func TestScanViolationsExemptsLintIgnoreInBodyComment(t *testing.T) {
	t.Parallel()

	content := `package sample

func Old() {
	//lint:ignore SA1019 keeping deprecated call until callers migrate
	legacy()
}
`
	got, err := scanViolations(scanParams{FilePath: "sample.go", Content: content})
	require.NoError(t, err)

	for _, v := range got {
		require.NotEqual(t, "no-comment-in-body", v.RuleID,
			"//lint:ignore directives in a body must be exempt: %+v", got)
	}
}

func TestScanViolationsCatchesPlainInBodyGoComment(t *testing.T) {
	t.Parallel()

	content := `package sample

func Old() {
	// narration that should be caught
	legacy()
}
`
	got, err := scanViolations(scanParams{FilePath: "sample.go", Content: content})
	require.NoError(t, err)

	sawComment := false
	for _, v := range got {
		if v.RuleID == "no-comment-in-body" {
			sawComment = true
		}
	}
	require.True(t, sawComment, "plain in-body Go comment must still be caught: %+v", got)
}

func TestScanViolationsExemptsNolintInBodyComment(t *testing.T) {
	t.Parallel()

	content := `package sample

func Old() {
	//nolint:errcheck // intentional fire-and-forget
	legacy()
}
`
	got, err := scanViolations(scanParams{FilePath: "sample.go", Content: content})
	require.NoError(t, err)

	for _, v := range got {
		require.NotEqual(t, "no-comment-in-body", v.RuleID,
			"//nolint directives in a body must remain exempt: %+v", got)
	}
}

func TestDiffViolations(t *testing.T) {
	t.Parallel()

	v := func(rule, text string) Violation {
		return Violation{RuleID: rule, Text: text}
	}

	cases := []struct {
		name   string
		before []Violation
		after  []Violation
		want   []Violation
	}{
		{
			name:   "both empty",
			before: nil,
			after:  nil,
			want:   nil,
		},
		{
			name:   "add one to empty",
			before: nil,
			after:  []Violation{v("r1", "t1")},
			want:   []Violation{v("r1", "t1")},
		},
		{
			name:   "removal only",
			before: []Violation{v("r1", "t1")},
			after:  nil,
			want:   nil,
		},
		{
			name:   "no change",
			before: []Violation{v("r1", "t1")},
			after:  []Violation{v("r1", "t1")},
			want:   nil,
		},
		{
			name:   "line shift (same rule + text) is not new",
			before: []Violation{{RuleID: "r1", Text: "t1", Line: 10}},
			after:  []Violation{{RuleID: "r1", Text: "t1", Line: 42}},
			want:   nil,
		},
		{
			name:   "swap one for another",
			before: []Violation{v("r1", "t1")},
			after:  []Violation{v("r2", "t2")},
			want:   []Violation{v("r2", "t2")},
		},
		{
			name:   "pre-existing left alone, one new added",
			before: []Violation{v("r1", "t1")},
			after:  []Violation{v("r1", "t1"), v("r2", "t2")},
			want:   []Violation{v("r2", "t2")},
		},
		{
			name:   "duplicate added beyond before count",
			before: []Violation{v("r1", "t1"), v("r1", "t1")},
			after:  []Violation{v("r1", "t1"), v("r1", "t1"), v("r1", "t1")},
			want:   []Violation{v("r1", "t1")},
		},
		{
			name:   "order preserved from after",
			before: nil,
			after:  []Violation{v("r3", "c"), v("r1", "a"), v("r2", "b")},
			want:   []Violation{v("r3", "c"), v("r1", "a"), v("r2", "b")},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := diffViolations(violationDelta{Before: tc.before, After: tc.after})
			require.Len(t, got, len(tc.want))
			for i := range got {
				require.Equal(t, tc.want[i].RuleID, got[i].RuleID, "index %d RuleID", i)
				require.Equal(t, tc.want[i].Text, got[i].Text, "index %d Text", i)
			}
		})
	}
}

func TestFormatViolationsEmpty(t *testing.T) {
	t.Parallel()

	require.Empty(t, formatViolations(nil))
}

func TestFormatViolationsSingleCompact(t *testing.T) {
	t.Parallel()

	v := Violation{
		RuleID:  "no-fmt-println",
		Text:    `fmt.Println("x")`,
		Line:    7,
		File:    "hello.go",
		Message: "fmt print functions are banned.",
	}
	got := formatViolations([]Violation{v})
	want := "no-fmt-println: fmt.Println(\"x\") @ hello.go:7\n  fmt print functions are banned."
	require.Equal(t, want, got)
}

func TestFormatViolationsMultiple(t *testing.T) {
	t.Parallel()

	vs := []Violation{
		{RuleID: "no-fmt-println", Text: `fmt.Println("x")`, Line: 3, File: "a.go", Message: "msg-1"},
		{RuleID: "no-nested-if", Text: `if x { if y {} }`, Line: 9, File: "a.go", Message: "msg-2"},
	}
	got := formatViolations(vs)
	require.Contains(t, got, "no-fmt-println: ")
	require.Contains(t, got, "no-nested-if: ")
	require.GreaterOrEqual(t, strings.Count(got, "\n"), 3)
}

func TestFormatViolationsTruncatesMultilineText(t *testing.T) {
	t.Parallel()

	multiline := "if x > 0 {\n\tif y > 0 {\n\t\treturn true\n\t}\n}"
	v := Violation{
		RuleID:  "no-nested-if",
		Text:    multiline,
		Line:    4,
		File:    "a.go",
		Message: "nested if banned",
	}
	got := formatViolations([]Violation{v})
	require.NotContains(t, got, "\treturn true")
	require.Contains(t, got, "if x > 0 {")
}

func TestFormatViolationsNormalizesMessageNewlines(t *testing.T) {
	t.Parallel()

	v := Violation{
		RuleID:  "r",
		Text:    "t",
		Line:    1,
		File:    "a.go",
		Message: "first line.\nsecond line.\n",
	}
	got := formatViolations([]Violation{v})
	require.NotContains(t, got, "\nsecond line.")
	require.Contains(t, got, "first line. second line.")
}

const dirtyGo = `package dirty

import "fmt"

func Legacy() {
	fmt.Println("legacy")
}

func Target() string {
	return "original"
}
`

func TestFilterSuppressedValidDirective(t *testing.T) {
	t.Parallel()

	content := `line 1
// astgrep-allow: ts-no-type-assertion -- bridges createElement generic to DOM attrs
const x = y as T
line 4
`
	violations := []Violation{
		{RuleID: "ts-no-type-assertion", Text: "y as T", Line: 3},
	}
	got := filterSuppressed(violations, content)
	require.Empty(t, got, "valid directive with substantive justification must suppress")
}

func TestFilterSuppressedWrongRuleID(t *testing.T) {
	t.Parallel()

	content := `// astgrep-allow: some-other-rule -- this justification is long enough
const x = y as T
`
	violations := []Violation{
		{RuleID: "ts-no-type-assertion", Text: "y as T", Line: 2},
	}
	got := filterSuppressed(violations, content)
	require.Len(t, got, 1, "directive for a different rule must not suppress")
}

func TestFilterSuppressedShortJustification(t *testing.T) {
	t.Parallel()

	content := `// astgrep-allow: ts-no-type-assertion -- too short
const x = y as T
`
	violations := []Violation{
		{RuleID: "ts-no-type-assertion", Text: "y as T", Line: 2},
	}
	got := filterSuppressed(violations, content)
	require.Len(t, got, 1, "justification under 10 non-whitespace chars must not suppress")
}

func TestFilterSuppressedMissingDashes(t *testing.T) {
	t.Parallel()

	content := `// astgrep-allow: ts-no-type-assertion bridges createElement generics
const x = y as T
`
	violations := []Violation{
		{RuleID: "ts-no-type-assertion", Text: "y as T", Line: 2},
	}
	got := filterSuppressed(violations, content)
	require.Len(t, got, 1, "directive missing -- justification separator must not suppress")
}

func TestFilterSuppressedNotOnPrecedingLine(t *testing.T) {
	t.Parallel()

	content := `// astgrep-allow: ts-no-type-assertion -- bridges createElement generic to DOM attrs

const x = y as T
`
	violations := []Violation{
		{RuleID: "ts-no-type-assertion", Text: "y as T", Line: 3},
	}
	got := filterSuppressed(violations, content)
	require.Len(t, got, 1, "directive two lines above violation must not suppress")
}

func TestFilterSuppressedHashComment(t *testing.T) {
	t.Parallel()

	content := `# astgrep-allow: ts-no-type-assertion -- bridges createElement generic to DOM attrs
const x = y as T
`
	violations := []Violation{
		{RuleID: "ts-no-type-assertion", Text: "y as T", Line: 2},
	}
	got := filterSuppressed(violations, content)
	require.Empty(t, got, "# style directive must suppress")
}

func TestFilterSuppressedEmptyViolations(t *testing.T) {
	t.Parallel()

	got := filterSuppressed(nil, "anything")
	require.Empty(t, got)
}

func TestDecideSuppressedViolationIsAllowed(t *testing.T) {
	t.Parallel()

	newContent := `package fresh

import "fmt"

func Leak() {
	// astgrep-allow: no-fmt-println -- intentional diagnostic output in a CLI tool
	fmt.Println("x")
}
`
	out := runDecide(t, decideCase{
		FileName:     "fresh.go",
		WriteContent: newContent,
	})
	require.True(t, out.IsNoop(), "suppressed violation with valid justification must be allowed")
}

func TestDecideSuppressedWithShortJustificationBlocks(t *testing.T) {
	t.Parallel()

	newContent := `package fresh

import "fmt"

func Leak() {
	// astgrep-allow: no-fmt-println -- fine
	fmt.Println("x")
}
`
	out := runDecide(t, decideCase{
		FileName:     "fresh.go",
		WriteContent: newContent,
	})
	require.False(t, out.IsNoop(), "short justification must not suppress")
	require.Equal(t, hookio.PermissionDeny, out.PermissionDecision())
}

func TestDecideCleanFileCleanEdit(t *testing.T) {
	t.Parallel()

	clean := `package clean

func Ok() string {
	return "ok"
}
`
	out := runDecide(t, decideCase{
		InitialContent: clean,
		FileName:       "clean.go",
		OldString:      `return "ok"`,
		NewString:      `return "ok!"`,
	})
	require.True(t, out.IsNoop())
}

func TestDecideDirtyFileEditFarFromViolation(t *testing.T) {
	t.Parallel()

	out := runDecide(t, decideCase{
		InitialContent: dirtyGo,
		FileName:       "dirty.go",
		OldString:      `return "original"`,
		NewString:      `return "changed"`,
	})
	require.True(t, out.IsNoop(), "pre-existing fmt.Println must not block unrelated edit")
}

func TestDecideCleanFileEditAddsNewViolation(t *testing.T) {
	t.Parallel()

	clean := `package clean

import "fmt"

func Ok() {
	_ = fmt.Sprint("ok")
}
`
	out := runDecide(t, decideCase{
		InitialContent: clean,
		FileName:       "clean.go",
		OldString:      `_ = fmt.Sprint("ok")`,
		NewString:      `fmt.Println("ok")`,
	})
	require.False(t, out.IsNoop())
	require.Equal(t, hookio.PermissionDeny, out.PermissionDecision())
	require.Contains(t, out.Reason(), "no-fmt-println")
}

func TestDecideDirtyFileEditAddsDifferentNewViolation(t *testing.T) {
	t.Parallel()

	out := runDecide(t, decideCase{
		InitialContent: dirtyGo,
		FileName:       "dirty.go",
		OldString: `func Target() string {
	return "original"
}`,
		NewString: `func Target() string {
	// TODO remove this
	return "original"
}`,
	})
	require.False(t, out.IsNoop())
	require.Equal(t, hookio.PermissionDeny, out.PermissionDecision())
	reason := out.Reason()
	require.Contains(t, reason, "no-todo-comments")
	require.NotContains(t, reason, "no-fmt-println",
		"pre-existing fmt.Println must not appear in denial of a different-rule edit")
}

func TestDecideDirtyFileEditAddsDuplicateOfExistingViolation(t *testing.T) {
	t.Parallel()

	out := runDecide(t, decideCase{
		InitialContent: dirtyGo,
		FileName:       "dirty.go",
		OldString:      `fmt.Println("legacy")`,
		NewString: `fmt.Println("legacy")
	fmt.Println("legacy")`,
	})
	require.False(t, out.IsNoop())
	require.Equal(t, hookio.PermissionDeny, out.PermissionDecision())
	require.Contains(t, out.Reason(), "no-fmt-println")
	require.Equal(t, 1, strings.Count(out.Reason(), "no-fmt-println:"),
		"only the added duplicate must be cited, not the pre-existing one")
}

func TestDecideDirtyFileEditRemovesViolation(t *testing.T) {
	t.Parallel()

	out := runDecide(t, decideCase{
		InitialContent: dirtyGo,
		FileName:       "dirty.go",
		OldString: `func Legacy() {
	fmt.Println("legacy")
}`,
		NewString: `func Legacy() {}`,
	})
	require.True(t, out.IsNoop())
}

func TestDecideLineShiftPreservesExistingViolationIdentity(t *testing.T) {
	t.Parallel()

	out := runDecide(t, decideCase{
		InitialContent: dirtyGo,
		FileName:       "dirty.go",
		OldString:      `package dirty`,
		NewString: `package dirty

func Inserted() string { return "hi" }
`,
	})
	require.True(t, out.IsNoop(),
		"line shift of an unchanged pre-existing violation must not be treated as new")
}

func TestDecideWriteBrandNewFileWithViolation(t *testing.T) {
	t.Parallel()

	newContent := `package fresh

import "fmt"

func Leak() {
	fmt.Println("x")
}
`
	out := runDecide(t, decideCase{
		FileName:     "fresh.go",
		WriteContent: newContent,
	})
	require.False(t, out.IsNoop())
	require.Equal(t, hookio.PermissionDeny, out.PermissionDecision())
	require.Contains(t, out.Reason(), "no-fmt-println")
}

func TestDecideFailsClosedWhenAstGrepMissing(t *testing.T) {
	t.Setenv("PATH", "/nonexistent")

	clean := `package clean

func Ok() {}
`
	out := runDecide(t, decideCase{
		InitialContent: clean,
		FileName:       "clean.go",
		OldString:      "func Ok() {}",
		NewString:      `func Ok() string { return "hi" }`,
	})
	require.False(t, out.IsNoop())
	require.Equal(t, hookio.PermissionDeny, out.PermissionDecision())
	require.Contains(t, out.Reason(), "fail-closed")
}

func TestDecideNoOpForNonGoOrTsFile(t *testing.T) {
	t.Parallel()

	out := runDecide(t, decideCase{
		InitialContent: "some markdown\n",
		FileName:       "README.md",
		OldString:      "some",
		NewString:      "changed",
	})
	require.True(t, out.IsNoop())
}
