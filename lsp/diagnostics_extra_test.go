package lsp

import (
	"testing"

	"github.com/infrasutra/fsl/parser"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExtractFirstQuoted(t *testing.T) {
	assert.Equal(t, "Post", extractFirstQuoted("type 'Post' has a problem"))
	assert.Equal(t, "", extractFirstQuoted("no quotes here"))
	assert.Equal(t, "", extractFirstQuoted("only one quote '"))
}

func TestMapSeverity_AllBranches(t *testing.T) {
	assert.Equal(t, SeverityError, mapSeverity(parser.SeverityError))
	assert.Equal(t, SeverityWarning, mapSeverity(parser.SeverityWarning))
	assert.Equal(t, SeverityInformation, mapSeverity(parser.SeverityInfo))
	assert.Equal(t, SeverityHint, mapSeverity(parser.SeverityHint))
	assert.Equal(t, SeverityError, mapSeverity(parser.DiagnosticSeverity(99)))
}

func TestMapLintSeverity_AllBranches(t *testing.T) {
	assert.Equal(t, SeverityWarning, mapLintSeverity(parser.LintWarning))
	assert.Equal(t, SeverityHint, mapLintSeverity(parser.LintHint))
	assert.Equal(t, SeverityWarning, mapLintSeverity(parser.LintSeverity(99)))
}

func TestGetDiagnostics_LintFindingsIncluded(t *testing.T) {
	doc := newDoc("type lowercase_type {\n  x: String!\n}")
	diags := GetDiagnostics(doc, []*Document{doc})

	require.NotEmpty(t, diags)
	found := false
	for _, d := range diags {
		if d.Source == "fsl-lint" {
			found = true
			assert.NotEmpty(t, d.Code)
		}
	}
	assert.True(t, found, "expected at least one fsl-lint diagnostic, got: %v", diags)
}

func TestFindLintNameRange_FieldNameBranch(t *testing.T) {
	doc := newDoc("type Post {\n  title: String!\n}")
	rng := findLintNameRange(doc, parser.LintResult{TypeName: "Post", FieldName: "title", Message: "field issue"})
	assert.Equal(t, 1, rng.Start.Line)
}

func TestFindLintNameRange_TypeNameOnlyBranch(t *testing.T) {
	doc := newDoc("type Post {\n  title: String!\n}")
	rng := findLintNameRange(doc, parser.LintResult{TypeName: "Post", Message: "type issue"})
	assert.Equal(t, 0, rng.Start.Line)
}

func TestFindLintNameRange_QuotedFallbackBranch(t *testing.T) {
	doc := newDoc("type Post {\n  title: String!\n}")
	rng := findLintNameRange(doc, parser.LintResult{Message: "field 'title' has an issue"})
	assert.Equal(t, 1, rng.Start.Line)
}

func TestFindLintNameRange_NoMatchReturnsZeroRange(t *testing.T) {
	doc := newDoc("type Post {\n  title: String!\n}")
	rng := findLintNameRange(doc, parser.LintResult{Message: "no quotes here"})
	assert.Equal(t, Range{}, rng)
}

func TestFindEnumNameRange(t *testing.T) {
	doc := newDoc("enum Status {\n  draft\n}")

	t.Run("finds range", func(t *testing.T) {
		rng := findEnumNameRange(doc, "Status")
		assert.Equal(t, 0, rng.Start.Line)
	})

	t.Run("missing enum returns zero position", func(t *testing.T) {
		rng := findEnumNameRange(doc, "Missing")
		assert.Equal(t, Position{Line: 0, Character: 0}, rng.Start)
	})
}

func TestGetDiagnostics_NilResultReturnsEmpty(t *testing.T) {
	doc := newDoc("")
	diags := GetDiagnostics(doc, nil)
	assert.NotNil(t, diags)
}
