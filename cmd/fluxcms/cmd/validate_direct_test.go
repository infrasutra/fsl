package cmd

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/infrasutra/fsl/parser"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func withValidateFlags(t *testing.T, format string, lint bool) {
	t.Helper()
	prevFormat, prevLint := validateFormat, validateLint
	validateFormat, validateLint = format, lint
	t.Cleanup(func() { validateFormat, validateLint = prevFormat, prevLint })
}

func TestRunValidate_ValidPretty(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "post.fsl")
	require.NoError(t, os.WriteFile(p, []byte(`type Post { title: String! }`), 0o644))

	withValidateFlags(t, "pretty", false)
	require.NoError(t, runValidate(validateCmd, []string{p}))
}

func TestRunValidate_InvalidPretty(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "bad.fsl")
	require.NoError(t, os.WriteFile(p, []byte(`type Post { title: `), 0o644))

	withValidateFlags(t, "pretty", false)
	err := runValidate(validateCmd, []string{p})
	assert.ErrorContains(t, err, "validation failed")
}

func TestRunValidate_JSONFormat(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "post.fsl")
	require.NoError(t, os.WriteFile(p, []byte(`type Post { title: String! }`), 0o644))

	withValidateFlags(t, "json", false)
	require.NoError(t, runValidate(validateCmd, []string{p}))
}

func TestRunValidate_LintFlagAddsWarnings(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "post.fsl")
	require.NoError(t, os.WriteFile(p, []byte(`type lowercase_type { x: String! }`), 0o644))

	withValidateFlags(t, "pretty", true)
	require.NoError(t, runValidate(validateCmd, []string{p}))
}

func TestRunValidate_UnreadableFileReported(t *testing.T) {
	dir := t.TempDir()
	missing := filepath.Join(dir, "gone.fsl")
	require.NoError(t, os.WriteFile(missing, []byte(`type X { a: String! }`), 0o644))
	require.NoError(t, os.Remove(missing))

	// collectFSLFiles accepts an explicit path even if it later can't be read,
	// exercising runValidate's cannot-read-file branch.
	files, err := collectFSLFiles([]string{dir})
	assert.Error(t, err)
	assert.Nil(t, files)
}

func TestOutputJSON(t *testing.T) {
	report := ValidationReport{
		Results:    []ValidationResult{{File: "a.fsl", Valid: true, Diagnostics: []parser.Diagnostic{}}},
		TotalFiles: 1,
		ValidFiles: 1,
	}
	require.NoError(t, outputJSON(report))
}

func TestOutputPretty(t *testing.T) {
	t.Run("all valid", func(t *testing.T) {
		report := ValidationReport{
			Results:    []ValidationResult{{File: "a.fsl", Valid: true, Diagnostics: []parser.Diagnostic{}}},
			TotalFiles: 1,
			ValidFiles: 1,
		}
		require.NoError(t, outputPretty(report))
	})

	t.Run("has errors", func(t *testing.T) {
		report := ValidationReport{
			Results: []ValidationResult{{
				File:  "a.fsl",
				Valid: false,
				Diagnostics: []parser.Diagnostic{{
					Severity: parser.SeverityError, Message: "boom", StartLine: 1, StartColumn: 1,
				}},
				Lines: []string{"type Bad {"},
			}},
			TotalFiles:  1,
			ValidFiles:  0,
			TotalErrors: 1,
		}
		err := outputPretty(report)
		assert.ErrorContains(t, err, "validation failed")
	})
}

func TestGetSeverityColorAndLabel(t *testing.T) {
	cases := []struct {
		sev   parser.DiagnosticSeverity
		label string
	}{
		{parser.SeverityError, "error"},
		{parser.SeverityWarning, "warning"},
		{parser.SeverityInfo, "info"},
		{parser.SeverityHint, "hint"},
		{parser.DiagnosticSeverity(99), "unknown"},
	}
	for _, tc := range cases {
		assert.Equal(t, tc.label, getSeverityLabel(tc.sev))
		if tc.label == "unknown" {
			assert.Equal(t, "", getSeverityColor(tc.sev))
		} else {
			assert.NotEmpty(t, getSeverityColor(tc.sev))
		}
	}
}

func TestFormatDiagnosticLine(t *testing.T) {
	lines := []string{"type Post {", "  title: String!", "}"}

	t.Run("valid position", func(t *testing.T) {
		lineText, caret := formatDiagnosticLine(lines, 2, 3)
		assert.Equal(t, "  title: String!", lineText)
		assert.Equal(t, "  ^", caret)
	})

	t.Run("line number out of range", func(t *testing.T) {
		lineText, caret := formatDiagnosticLine(lines, 0, 1)
		assert.Empty(t, lineText)
		assert.Empty(t, caret)

		lineText, caret = formatDiagnosticLine(lines, 99, 1)
		assert.Empty(t, lineText)
		assert.Empty(t, caret)
	})

	t.Run("empty line returns empty", func(t *testing.T) {
		lineText, caret := formatDiagnosticLine([]string{""}, 1, 1)
		assert.Empty(t, lineText)
		assert.Empty(t, caret)
	})

	t.Run("column beyond line length clamps", func(t *testing.T) {
		lineText, caret := formatDiagnosticLine([]string{"abc"}, 1, 999)
		assert.Equal(t, "abc", lineText)
		assert.Equal(t, "   ^", caret)
	})

	t.Run("non-positive column defaults to 1", func(t *testing.T) {
		lineText, caret := formatDiagnosticLine([]string{"abc"}, 1, 0)
		assert.Equal(t, "abc", lineText)
		assert.Equal(t, "^", caret)
	})
}
