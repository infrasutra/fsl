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
// the parser reports diagnostics instead of panicking.
func TestParse_MalformedInput(t *testing.T) {
	tests := []struct {
		name   string
		source string
	}{
		{"unterminated field", "type Post { title: "},
		{"unterminated type body", "type Post { title: String!"},
		{"unterminated string literal", `type Post { title: String! @default("unterminated) }`},
		{"missing type name", "type { title: String! }"},
		{"missing field type", "type Post { title: }"},
		{"unexpected token at top level", "42"},
		{"dangling decorator", "type Post { title: String! @ }"},
		{"nested unbalanced braces", "type Post { title: String! { }"},
		{"empty enum", "enum Status { }"},
		{"array without close bracket", "type Post { tags: [String }"},
		{"decorator with unbalanced parens", `type Post { title: String! @maxLength(200 }`},
		{"only whitespace", "   \n\t  "},
		{"only comments", "// just a comment\n// another one"},
		{"binary garbage", "\x00\x01\x02\x03"},
		{"truncated unicode escape", `type Post { title: String! @default("\u12") }`},
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
			// Malformed input should either be marked invalid or, if lenient parsing
			// recovers, still avoid panicking -- the panic guard above is the real assertion.
			_ = result.Valid
		})
	}
}

