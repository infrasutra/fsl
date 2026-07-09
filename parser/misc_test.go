package parser

import (
	"strings"
	"testing"
)

func TestLexer_Error(t *testing.T) {
	lexer := NewLexer(`type Post { title: `)
	err := lexer.Error("unexpected end of input")
	if err == nil {
		t.Fatal("expected non-nil error")
	}
	if !strings.Contains(err.Error(), "lexer error at line") {
		t.Fatalf("expected error to mention lexer position, got: %v", err)
	}
	if !strings.Contains(err.Error(), "unexpected end of input") {
		t.Fatalf("expected error to include the message, got: %v", err)
	}
}

func TestTokenType_String(t *testing.T) {
	tests := []struct {
		tok  TokenType
		want string
	}{
		{TokenEOF, "EOF"},
		{TokenIdent, "IDENT"},
		{TokenString, "STRING"},
		{TokenNumber, "NUMBER"},
		{TokenBool, "BOOL"},
		{TokenLBrace, "{"},
		{TokenRBrace, "}"},
		{TokenLBracket, "["},
		{TokenRBracket, "]"},
		{TokenLParen, "("},
		{TokenRParen, ")"},
	}
	for _, tt := range tests {
		if got := tt.tok.String(); got != tt.want {
			t.Errorf("TokenType(%d).String() = %q, want %q", tt.tok, got, tt.want)
		}
	}
}

func TestLintResultsToString(t *testing.T) {
	t.Run("empty results", func(t *testing.T) {
		if got := LintResultsToString(nil); got != "" {
			t.Fatalf("expected empty string, got %q", got)
		}
	})

	t.Run("formats warning with position", func(t *testing.T) {
		results := []LintResult{
			{
				Rule:    LintRule{Name: "naming-convention", Severity: LintWarning},
				Message: "type name should be PascalCase",
				Line:    3,
				Column:  1,
			},
		}
		out := LintResultsToString(results)
		if !strings.Contains(out, "warning:3:1 [naming-convention]") {
			t.Fatalf("unexpected format: %q", out)
		}
	})

	t.Run("formats hint without position", func(t *testing.T) {
		results := []LintResult{
			{
				Rule:    LintRule{Name: "some-hint", Severity: LintHint},
				Message: "consider renaming",
			},
		}
		out := LintResultsToString(results)
		if !strings.Contains(out, "hint [some-hint] consider renaming") {
			t.Fatalf("unexpected format: %q", out)
		}
	})
}

func TestExtractQuotedValue(t *testing.T) {
	if got := extractQuotedValue("unknown type: Foo"); got != "Foo" {
		t.Errorf("got %q, want %q", got, "Foo")
	}
	if got := extractQuotedValue("no colon here"); got != "" {
		t.Errorf("got %q, want empty string", got)
	}
}

func TestIsWhitespace(t *testing.T) {
	for _, c := range []byte{' ', '\t', '\n', '\r'} {
		if !isWhitespace(c) {
			t.Errorf("expected %q to be whitespace", c)
		}
	}
	if isWhitespace('a') {
		t.Error("expected 'a' to not be whitespace")
	}
}

func TestGetFileExtension(t *testing.T) {
	cases := map[string]string{
		"schema.fsl":     "fsl",
		"schema.FSL":     "fsl",
		"a.b.json":       "json",
		"noextension":    "",
		"dir/schema.yml": "yml",
	}
	for input, want := range cases {
		if got := getFileExtension(input); got != want {
			t.Errorf("getFileExtension(%q) = %q, want %q", input, got, want)
		}
	}
}

func TestParseErrorToDiagnostic(t *testing.T) {
	result := ParseWithDiagnostics(`type Post { title: `)
	if result.Valid {
		t.Fatal("expected invalid schema")
	}
	found := false
	for _, d := range result.Diagnostics {
		if d.Severity == SeverityError && d.Source == "parser" {
			found = true
			if d.StartLine < 1 || d.StartColumn < 1 {
				t.Errorf("expected positive line/column, got line=%d col=%d", d.StartLine, d.StartColumn)
			}
		}
	}
	if !found {
		t.Fatalf("expected at least one parser-sourced error diagnostic, got: %v", result.Diagnostics)
	}
}

func TestValidationErrorToDiagnostic(t *testing.T) {
	result := ParseWithDiagnostics(`type Post { author: UnknownType! }`)
	if result.Valid {
		t.Fatal("expected invalid schema due to unknown type reference")
	}
	found := false
	for _, d := range result.Diagnostics {
		if d.Source == "validator" {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected a validator-sourced diagnostic, got: %v", result.Diagnostics)
	}
}
