package parser

import (
	"strings"
	"testing"
)

func mustParseForLint(t *testing.T, src string) *Schema {
	t.Helper()
	lexer := NewLexer(src)
	p := NewParser(lexer)
	schema, err := p.ParseSchema()
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	v := NewValidator(schema)
	v.Validate()
	return schema
}

func TestLint_NamingConvention_TypePascalCase(t *testing.T) {
	schema := mustParseForLint(t, `
		type article {
			title: String!
		}
	`)
	cfg := LinterConfig{NamingConvention: true}
	results := Lint(schema, cfg)
	found := false
	for _, r := range results {
		if r.Rule.Name == "naming-convention" && strings.Contains(r.Message, "article") {
			found = true
		}
	}
	if !found {
		t.Errorf("expected naming-convention warning for lowercase type name")
	}
}

func TestLint_NamingConvention_FieldCamelCase(t *testing.T) {
	schema := mustParseForLint(t, `
		type Article {
			Title: String!
		}
	`)
	cfg := LinterConfig{NamingConvention: true}
	results := Lint(schema, cfg)
	found := false
	for _, r := range results {
		if r.Rule.Name == "naming-convention" && strings.Contains(r.Message, "Title") {
			found = true
		}
	}
	if !found {
		t.Errorf("expected naming-convention warning for field 'Title'")
	}
}

func TestLint_NamingConvention_Valid(t *testing.T) {
	schema := mustParseForLint(t, `
		type Article {
			title: String!
			publishedAt: DateTime
		}
	`)
	cfg := LinterConfig{NamingConvention: true}
	results := Lint(schema, cfg)
	for _, r := range results {
		if r.Rule.Name == "naming-convention" {
			t.Errorf("unexpected naming-convention warning: %s", r.Message)
		}
	}
}

func TestLint_UnusedTypes_Detected(t *testing.T) {
	schema := mustParseForLint(t, `
		type Author {
			name: String!
		}
		type Article {
			title: String!
		}
	`)
	cfg := LinterConfig{UnusedTypes: true}
	results := Lint(schema, cfg)
	if len(results) == 0 {
		t.Fatal("expected unused-types warnings")
	}
}

func TestLint_UnusedTypes_Referenced(t *testing.T) {
	schema := mustParseForLint(t, `
		type Author {
			name: String!
		}
		type Article {
			title: String!
			author: Author
		}
	`)
	cfg := LinterConfig{UnusedTypes: true}
	results := Lint(schema, cfg)
	for _, r := range results {
		if r.Rule.Name == "unused-types" && strings.Contains(r.Message, "Author") {
			t.Errorf("Author is referenced, should not be flagged")
		}
	}
}

func TestLint_UnusedTypes_SingleType_Skipped(t *testing.T) {
	schema := mustParseForLint(t, `
		type Article {
			title: String!
		}
	`)
	cfg := LinterConfig{UnusedTypes: true}
	results := Lint(schema, cfg)
	for _, r := range results {
		if r.Rule.Name == "unused-types" {
			t.Errorf("single-type schema should not trigger unused-types")
		}
	}
}

func TestLint_RequiredFieldOrdering_Violation(t *testing.T) {
	schema := mustParseForLint(t, `
		type Article {
			subtitle: String
			title: String!
		}
	`)
	cfg := LinterConfig{RequiredFieldOrdering: true}
	results := Lint(schema, cfg)
	found := false
	for _, r := range results {
		if r.Rule.Name == "required-field-ordering" && strings.Contains(r.Message, "title") {
			found = true
		}
	}
	if !found {
		t.Errorf("expected required-field-ordering warning for 'title'")
	}
}

func TestLint_RequiredFieldOrdering_Valid(t *testing.T) {
	schema := mustParseForLint(t, `
		type Article {
			title: String!
			body: Text!
			subtitle: String
		}
	`)
	cfg := LinterConfig{RequiredFieldOrdering: true}
	results := Lint(schema, cfg)
	for _, r := range results {
		if r.Rule.Name == "required-field-ordering" {
			t.Errorf("unexpected required-field-ordering warning: %s", r.Message)
		}
	}
}

func TestLint_RelationCardinality_SingularRelation(t *testing.T) {
	schema := mustParseForLint(t, `
		type Author {
			name: String!
		}
		type Article {
			title: String!
			author: Author
		}
	`)
	cfg := LinterConfig{RelationCardinality: true}
	results := Lint(schema, cfg)
	found := false
	for _, r := range results {
		if r.Rule.Name == "relation-cardinality" && strings.Contains(r.Message, "author") {
			found = true
			if r.Rule.Severity != LintHint {
				t.Errorf("relation-cardinality should be LintHint")
			}
		}
	}
	if !found {
		t.Errorf("expected relation-cardinality hint for 'author'")
	}
}

func TestLint_MaxFieldCount_Exceeded(t *testing.T) {
	src := "type Article {\n"
	for i := 0; i < 5; i++ {
		src += "  field" + string(rune('a'+i)) + ": String\n"
	}
	src += "}"
	schema := mustParseForLint(t, src)
	cfg := LinterConfig{MaxFieldCount: 3}
	results := Lint(schema, cfg)
	found := false
	for _, r := range results {
		if r.Rule.Name == "max-field-count" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected max-field-count warning")
	}
}

func TestDefaultLinterConfig(t *testing.T) {
	cfg := DefaultLinterConfig()
	if !cfg.NamingConvention || !cfg.UnusedTypes || !cfg.RequiredFieldOrdering || !cfg.RelationCardinality {
		t.Error("all rules should default to true")
	}
	if cfg.MaxFieldCount != 30 {
		t.Errorf("MaxFieldCount should default to 30, got %d", cfg.MaxFieldCount)
	}
}

func TestLintResultsToString_Empty(t *testing.T) {
	if s := LintResultsToString(nil); s != "" {
		t.Errorf("expected empty string for nil results")
	}
}
