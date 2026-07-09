package parser

import (
	"strconv"
	"strings"
	"testing"
)

// TestParse_Unicode exercises unicode content in strings, identifiers, and decorators.
func TestParse_Unicode(t *testing.T) {
	tests := []struct {
		name    string
		source  string
		wantErr bool
	}{
		{
			name: "unicode default string value",
			source: `type Article {
  title: String! @default("héllo wörld 日本語")
}`,
			wantErr: false,
		},
		{
			name: "unicode in description decorator",
			source: `@description("Ceci est en français 🎉")
type Article {
  title: String!
}`,
			wantErr: false,
		},
		{
			name: "unicode pattern value",
			source: `type Article {
  slug: String! @pattern("^[\\p{L}0-9-]+$")
}`,
			wantErr: false,
		},
		{
			name: "emoji in string literal",
			source: `type Article {
  title: String! @default("🚀🎉✨")
}`,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ParseWithDiagnostics(tt.source)
			if tt.wantErr && result.Valid {
				t.Fatalf("expected invalid schema, got valid: %s", tt.source)
			}
			if !tt.wantErr && !result.Valid {
				t.Fatalf("expected valid schema, got errors: %v", result.Diagnostics)
			}
		})
	}
}

// TestParse_DeeplyNestedTypes exercises many chained types and a type with a large number
// of fields, to ensure the parser and validator don't choke on depth or width.
func TestParse_DeeplyNestedTypes(t *testing.T) {
	t.Run("long chain of relations", func(t *testing.T) {
		const depth = 50
		var sb strings.Builder
		for i := 0; i < depth; i++ {
			sb.WriteString("type Node")
			sb.WriteString(strconv.Itoa(i))
			sb.WriteString(" {\n  name: String!\n")
			if i > 0 {
				sb.WriteString("  parent: Node")
				sb.WriteString(strconv.Itoa(i - 1))
				sb.WriteString(" @relation\n")
			}
			sb.WriteString("}\n\n")
		}

		result := ParseWithDiagnostics(sb.String())
		if !result.Valid {
			t.Fatalf("expected deeply chained relations to parse, got errors: %v", result.Diagnostics)
		}
		if len(result.Schema.Types) != depth {
			t.Fatalf("expected %d types, got %d", depth, len(result.Schema.Types))
		}
	})

	t.Run("type with a very large number of fields", func(t *testing.T) {
		const fieldCount = 200
		var sb strings.Builder
		sb.WriteString("type Wide {\n")
		for i := 0; i < fieldCount; i++ {
			sb.WriteString("  field")
			sb.WriteString(strconv.Itoa(i))
			sb.WriteString(": String\n")
		}
		sb.WriteString("}\n")

		result := ParseWithDiagnostics(sb.String())
		if !result.Valid {
			t.Fatalf("expected wide type to parse, got errors: %v", result.Diagnostics)
		}
		if len(result.Schema.Types[0].Fields) != fieldCount {
			t.Fatalf("expected %d fields, got %d", fieldCount, len(result.Schema.Types[0].Fields))
		}
	})

	t.Run("deeply nested slice components", func(t *testing.T) {
		source := `type Page {
  slices: JSON! @slices(hero: HeroSlice, cta: CtaSlice)
}

type HeroSlice {
  heading: String!
  sub: SubBlock @relation
}

type SubBlock {
  detail: String!
}

type CtaSlice {
  label: String!
  target: String!
}`
		result := ParseWithDiagnostics(source)
		if !result.Valid {
			t.Fatalf("expected nested slice schema to parse, got errors: %v", result.Diagnostics)
		}
	})
}

// TestParse_MalformedInput exercises a variety of syntactically broken schemas to ensure
// the parser reports precise diagnostics (or, for lenient cases, the expected valid empty
// schema) instead of panicking or silently swallowing errors.
func TestParse_MalformedInput(t *testing.T) {
	tests := []struct {
		name       string
		source     string
		wantValid  bool
		wantSource string // diagnostic Source to require when wantValid is false
		wantMsg    string // substring the diagnostic Message must contain
		wantLine   int    // exact StartLine to require (0 = don't check)
		wantColumn int    // exact StartColumn to require (0 = don't check)
		wantTypes  int    // when wantValid, exact len(Schema.Types) to require
		wantEnums  int    // when wantValid, exact len(Schema.Enums) to require
	}{
		{
			name: "unterminated field", source: "type Post { title: ",
			wantValid: false, wantSource: "parser", wantMsg: "expected type name, got EOF", wantLine: 1, wantColumn: 20,
		},
		{
			name: "unterminated type body", source: "type Post { title: String!",
			wantValid: false, wantSource: "parser", wantMsg: "expected '}', got EOF", wantLine: 1, wantColumn: 27,
		},
		{
			name: "unterminated string literal", source: `type Post { title: String! @default("unterminated) }`,
			wantValid: false, wantSource: "parser", wantMsg: "expected ')', got EOF", wantLine: 1, wantColumn: 54,
		},
		{
			name: "missing type name", source: "type { title: String! }",
			wantValid: false, wantSource: "parser", wantMsg: "expected type name, got {", wantLine: 1, wantColumn: 6,
		},
		{
			name: "missing field type", source: "type Post { title: }",
			wantValid: false, wantSource: "parser", wantMsg: "expected type name, got }", wantLine: 1, wantColumn: 20,
		},
		{
			name: "unexpected token at top level", source: "42",
			wantValid: false, wantSource: "parser", wantMsg: "expected 'type', 'enum', or '@'", wantLine: 1, wantColumn: 1,
		},
		{
			name: "dangling decorator", source: "type Post { title: String! @ }",
			wantValid: false, wantSource: "parser", wantMsg: "expected decorator name, got }", wantLine: 1, wantColumn: 30,
		},
		{
			name: "nested unbalanced braces", source: "type Post { title: String! { }",
			wantValid: false, wantSource: "parser", wantMsg: "expected field name, got {", wantLine: 1, wantColumn: 28,
		},
		{
			name: "empty enum", source: "enum Status { }",
			wantValid: false, wantSource: "validator", wantMsg: "enum must have at least one value", wantLine: 1, wantColumn: 1,
		},
		{
			name: "array without close bracket", source: "type Post { tags: [String }",
			wantValid: false, wantSource: "parser", wantMsg: "expected ']', got }", wantLine: 1, wantColumn: 27,
		},
		{
			name: "decorator with unbalanced parens", source: `type Post { title: String! @maxLength(200 }`,
			wantValid: false, wantSource: "parser", wantMsg: "unexpected token in decorator argument", wantLine: 1, wantColumn: 43,
		},
		{
			name: "only whitespace", source: "   \n\t  ",
			wantValid: true, wantTypes: 0, wantEnums: 0,
		},
		{
			name: "only comments", source: "// just a comment\n// another one",
			wantValid: true, wantTypes: 0, wantEnums: 0,
		},
		{
			name: "binary garbage", source: "\x00\x01\x02\x03",
			// The lexer treats unrecognized control bytes as ignorable whitespace-like
			// noise rather than raising a lexer error, so this parses as an empty schema.
			wantValid: true, wantTypes: 0, wantEnums: 0,
		},
		{
			name: "truncated unicode escape", source: `type Post { title: String! @default("\u12") }`,
			// The decorator argument string isn't validated as a real unicode escape
			// sequence, so this is accepted as a literal string default.
			wantValid: true, wantTypes: 1, wantEnums: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			defer func() {
				if r := recover(); r != nil {
					t.Fatalf("ParseWithDiagnostics panicked on malformed input %q: %v", tt.source, r)
				}
			}()

			result := ParseWithDiagnostics(tt.source)
			if result == nil {
				t.Fatal("expected a non-nil diagnostics result even for malformed input")
			}

			if result.Valid != tt.wantValid {
				t.Fatalf("Valid = %v, want %v (diagnostics: %v)", result.Valid, tt.wantValid, result.Diagnostics)
			}

			if tt.wantValid {
				if result.Schema == nil {
					t.Fatal("expected a non-nil schema for valid lenient input")
				}
				if len(result.Schema.Types) != tt.wantTypes {
					t.Errorf("len(Schema.Types) = %d, want %d", len(result.Schema.Types), tt.wantTypes)
				}
				if len(result.Schema.Enums) != tt.wantEnums {
					t.Errorf("len(Schema.Enums) = %d, want %d", len(result.Schema.Enums), tt.wantEnums)
				}
				if len(result.Diagnostics) != 0 {
					t.Errorf("expected no diagnostics for valid lenient input, got: %v", result.Diagnostics)
				}
				return
			}

			if len(result.Diagnostics) == 0 {
				t.Fatal("expected at least one diagnostic for invalid input")
			}
			d := result.Diagnostics[0]
			if d.Severity != SeverityError {
				t.Errorf("Severity = %v, want SeverityError", d.Severity)
			}
			if d.Source != tt.wantSource {
				t.Errorf("Source = %q, want %q", d.Source, tt.wantSource)
			}
			if !strings.Contains(d.Message, tt.wantMsg) {
				t.Errorf("Message = %q, want substring %q", d.Message, tt.wantMsg)
			}
			if tt.wantLine != 0 && d.StartLine != tt.wantLine {
				t.Errorf("StartLine = %d, want %d", d.StartLine, tt.wantLine)
			}
			if tt.wantColumn != 0 && d.StartColumn != tt.wantColumn {
				t.Errorf("StartColumn = %d, want %d", d.StartColumn, tt.wantColumn)
			}
		})
	}
}
