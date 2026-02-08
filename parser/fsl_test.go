package parser

import (
	"testing"
)

func TestParse_BasicTypes(t *testing.T) {
	tests := []struct {
		name    string
		source  string
		wantErr bool
	}{
		{
			name: "simple field with String type",
			source: `
				type Article {
					title: String!
				}
			`,
			wantErr: false,
		},
		{
			name: "multiple fields with different types",
			source: `
				type Article {
					title: String!
					body: Text!
					views: Int
					rating: Float
					published: Boolean!
					metadata: JSON
				}
			`,
			wantErr: false,
		},
		{
			name: "array types",
			source: `
				type Article {
					tags: [String]
					scores: [Int]!
				}
			`,
			wantErr: false,
		},
		{
			name: "optional fields without bang",
			source: `
				type Article {
					title: String!
					subtitle: String
				}
			`,
			wantErr: false,
		},
		{
			name: "required array with required elements",
			source: `
				type Article {
					tags: [String!]!
				}
			`,
			wantErr: false,
		},
		{
			name: "invalid syntax - missing brace",
			source: `
				type Article {
					title: String!
			`,
			wantErr: true,
		},
		{
			name: "invalid syntax - missing colon",
			source: `
				type Article {
					title String
				}
			`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := Parse(tt.source)
			if (err != nil) != tt.wantErr {
				t.Errorf("Parse() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestParse_Decorators(t *testing.T) {
	tests := []struct {
		name    string
		source  string
		wantErr bool
	}{
		{
			name: "single decorator after field",
			source: `
				type Article {
					title: String! @maxLength(100)
				}
			`,
			wantErr: false,
		},
		{
			name: "multiple decorators",
			source: `
				type Article {
					title: String! @minLength(1) @maxLength(100)
				}
			`,
			wantErr: false,
		},
		{
			name: "decorator with pattern",
			source: `
				type Article {
					slug: String! @pattern("^[a-z0-9-]+$")
				}
			`,
			wantErr: false,
		},
		{
			name: "numeric decorators",
			source: `
				type Article {
					rating: Float @min(0) @max(5)
				}
			`,
			wantErr: false,
		},
		{
			name: "default value decorator",
			source: `
				type Article {
					status: String @default("draft")
				}
			`,
			wantErr: false,
		},
		{
			name: "unique decorator",
			source: `
				type Article {
					slug: String! @unique
				}
			`,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := Parse(tt.source)
			if (err != nil) != tt.wantErr {
				t.Errorf("Parse() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestParseAndCompile(t *testing.T) {
	source := `
		type Article {
			title: String! @minLength(1) @maxLength(200)
			body: Text! @minLength(10)
			views: Int
			tags: [String]
			published: Boolean!
		}
	`

	compiled, err := ParseAndCompile(source, "Article", "article", false)
	if err != nil {
		t.Fatalf("ParseAndCompile() error = %v", err)
	}

	if compiled.Name != "Article" {
		t.Errorf("expected Name 'Article', got '%s'", compiled.Name)
	}

	if compiled.ApiID != "article" {
		t.Errorf("expected ApiID 'article', got '%s'", compiled.ApiID)
	}

	if compiled.Singleton {
		t.Errorf("expected Singleton false, got true")
	}

	if compiled.Checksum == "" {
		t.Error("expected non-empty checksum")
	}

	// Check fields
	expectedFields := map[string]struct {
		fieldType string
		required  bool
		isArray   bool
	}{
		"title":     {"String", true, false},
		"body":      {"Text", true, false},
		"views":     {"Int", false, false},
		"tags":      {"String", false, true},
		"published": {"Boolean", true, false},
	}

	if len(compiled.Fields) != len(expectedFields) {
		t.Errorf("expected %d fields, got %d", len(expectedFields), len(compiled.Fields))
	}

	for _, field := range compiled.Fields {
		expected, ok := expectedFields[field.Name]
		if !ok {
			t.Errorf("unexpected field: %s", field.Name)
			continue
		}

		if field.Type != expected.fieldType {
			t.Errorf("field %s: expected type '%s', got '%s'", field.Name, expected.fieldType, field.Type)
		}

		if field.Required != expected.required {
			t.Errorf("field %s: expected required=%v, got %v", field.Name, expected.required, field.Required)
		}

		if field.Array != expected.isArray {
			t.Errorf("field %s: expected array=%v, got %v", field.Name, expected.isArray, field.Array)
		}
	}
}

func TestValidateData(t *testing.T) {
	source := `
		type Article {
			title: String! @minLength(1) @maxLength(50)
			views: Int
			rating: Float
			published: Boolean!
			tags: [String]
		}
	`

	compiled, err := ParseAndCompile(source, "Article", "article", false)
	if err != nil {
		t.Fatalf("ParseAndCompile() error = %v", err)
	}

	tests := []struct {
		name       string
		data       map[string]any
		wantErrors int
	}{
		{
			name: "valid data",
			data: map[string]any{
				"title":     "Hello World",
				"published": true,
			},
			wantErrors: 0,
		},
		{
			name: "valid data with optional fields",
			data: map[string]any{
				"title":     "Hello World",
				"views":     float64(100),
				"rating":    float64(4.5),
				"published": false,
				"tags":      []any{"tech", "news"},
			},
			wantErrors: 0,
		},
		{
			name: "missing required field",
			data: map[string]any{
				"published": true,
			},
			wantErrors: 1, // missing title
		},
		{
			name: "wrong type for field",
			data: map[string]any{
				"title":     123, // should be string
				"published": true,
			},
			wantErrors: 1,
		},
		{
			name: "string too long",
			data: map[string]any{
				"title":     "This is a very long title that exceeds the maximum length of 50 characters allowed for this field",
				"published": true,
			},
			wantErrors: 1,
		},
		{
			name: "string too short",
			data: map[string]any{
				"title":     "",
				"published": true,
			},
			wantErrors: 1,
		},
		{
			name: "integer must be whole number",
			data: map[string]any{
				"title":     "Hello",
				"views":     float64(100.5), // should be integer
				"published": true,
			},
			wantErrors: 1,
		},
		{
			name: "unexpected field",
			data: map[string]any{
				"title":     "Hello",
				"published": true,
				"unknown":   "field",
			},
			wantErrors: 1,
		},
		{
			name: "array type validation",
			data: map[string]any{
				"title":     "Hello",
				"published": true,
				"tags":      "not-an-array", // should be array
			},
			wantErrors: 1,
		},
		{
			name: "array with wrong element types",
			data: map[string]any{
				"title":     "Hello",
				"published": true,
				"tags":      []any{"valid", 123, "also-valid"}, // 123 should be string
			},
			wantErrors: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errors := ValidateData(tt.data, compiled)
			if len(errors) != tt.wantErrors {
				t.Errorf("ValidateData() got %d errors, want %d errors", len(errors), tt.wantErrors)
				for _, e := range errors {
					t.Logf("  - %s: %s", e.Field, e.Message)
				}
			}
		})
	}
}

func TestValidateData_DateTime(t *testing.T) {
	source := `
		type Event {
			name: String!
			startTime: DateTime!
		}
	`

	compiled, err := ParseAndCompile(source, "Event", "event", false)
	if err != nil {
		t.Fatalf("ParseAndCompile() error = %v", err)
	}

	tests := []struct {
		name       string
		data       map[string]any
		wantErrors int
	}{
		{
			name: "valid ISO 8601 datetime",
			data: map[string]any{
				"name":      "Conference",
				"startTime": "2024-01-15T10:30:00Z",
			},
			wantErrors: 0,
		},
		{
			name: "valid datetime with timezone offset",
			data: map[string]any{
				"name":      "Meeting",
				"startTime": "2024-01-15T10:30:00+05:30",
			},
			wantErrors: 0,
		},
		{
			name: "invalid datetime format",
			data: map[string]any{
				"name":      "Event",
				"startTime": "2024-01-15", // missing time component
			},
			wantErrors: 1,
		},
		{
			name: "datetime not a string",
			data: map[string]any{
				"name":      "Event",
				"startTime": 1705315800, // unix timestamp
			},
			wantErrors: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errors := ValidateData(tt.data, compiled)
			if len(errors) != tt.wantErrors {
				t.Errorf("ValidateData() got %d errors, want %d", len(errors), tt.wantErrors)
				for _, e := range errors {
					t.Logf("  - %s: %s", e.Field, e.Message)
				}
			}
		})
	}
}

func TestValidateData_Pattern(t *testing.T) {
	source := `
		type Article {
			slug: String! @pattern("^[a-z0-9-]+$")
			email: String @pattern("^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$")
		}
	`

	compiled, err := ParseAndCompile(source, "Article", "article", false)
	if err != nil {
		t.Fatalf("ParseAndCompile() error = %v", err)
	}

	tests := []struct {
		name       string
		data       map[string]any
		wantErrors int
	}{
		{
			name: "valid slug",
			data: map[string]any{
				"slug": "hello-world-123",
			},
			wantErrors: 0,
		},
		{
			name: "invalid slug with uppercase",
			data: map[string]any{
				"slug": "Hello-World",
			},
			wantErrors: 1,
		},
		{
			name: "invalid slug with spaces",
			data: map[string]any{
				"slug": "hello world",
			},
			wantErrors: 1,
		},
		{
			name: "valid email",
			data: map[string]any{
				"slug":  "hello",
				"email": "user@example.com",
			},
			wantErrors: 0,
		},
		{
			name: "invalid email",
			data: map[string]any{
				"slug":  "hello",
				"email": "not-an-email",
			},
			wantErrors: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errors := ValidateData(tt.data, compiled)
			if len(errors) != tt.wantErrors {
				t.Errorf("ValidateData() got %d errors, want %d", len(errors), tt.wantErrors)
				for _, e := range errors {
					t.Logf("  - %s: %s", e.Field, e.Message)
				}
			}
		})
	}
}

func TestValidateData_NumericConstraints(t *testing.T) {
	source := `
		type Product {
			name: String!
			price: Float! @min(0) @max(1000)
			quantity: Int! @min(0)
		}
	`

	compiled, err := ParseAndCompile(source, "Product", "product", false)
	if err != nil {
		t.Fatalf("ParseAndCompile() error = %v", err)
	}

	tests := []struct {
		name       string
		data       map[string]any
		wantErrors int
	}{
		{
			name: "valid values",
			data: map[string]any{
				"name":     "Widget",
				"price":    float64(99.99),
				"quantity": float64(10),
			},
			wantErrors: 0,
		},
		{
			name: "negative price",
			data: map[string]any{
				"name":     "Widget",
				"price":    float64(-10),
				"quantity": float64(10),
			},
			wantErrors: 1,
		},
		{
			name: "price exceeds max",
			data: map[string]any{
				"name":     "Widget",
				"price":    float64(1500),
				"quantity": float64(10),
			},
			wantErrors: 1,
		},
		{
			name: "negative quantity",
			data: map[string]any{
				"name":     "Widget",
				"price":    float64(10),
				"quantity": float64(-5),
			},
			wantErrors: 1,
		},
		{
			name: "boundary values",
			data: map[string]any{
				"name":     "Widget",
				"price":    float64(0),
				"quantity": float64(0),
			},
			wantErrors: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errors := ValidateData(tt.data, compiled)
			if len(errors) != tt.wantErrors {
				t.Errorf("ValidateData() got %d errors, want %d", len(errors), tt.wantErrors)
				for _, e := range errors {
					t.Logf("  - %s: %s", e.Field, e.Message)
				}
			}
		})
	}
}

func TestValidateData_JSON(t *testing.T) {
	source := `
		type Config {
			name: String!
			settings: JSON!
		}
	`

	compiled, err := ParseAndCompile(source, "Config", "config", false)
	if err != nil {
		t.Fatalf("ParseAndCompile() error = %v", err)
	}

	tests := []struct {
		name       string
		data       map[string]any
		wantErrors int
	}{
		{
			name: "json object",
			data: map[string]any{
				"name": "MyConfig",
				"settings": map[string]any{
					"theme":    "dark",
					"fontSize": float64(14),
				},
			},
			wantErrors: 0,
		},
		{
			name: "json array",
			data: map[string]any{
				"name":     "MyConfig",
				"settings": []any{"one", float64(2), true},
			},
			wantErrors: 0,
		},
		{
			name: "nested json",
			data: map[string]any{
				"name": "MyConfig",
				"settings": map[string]any{
					"nested": map[string]any{
						"deep": []any{float64(1), float64(2), float64(3)},
					},
				},
			},
			wantErrors: 0,
		},
		{
			name: "json primitive",
			data: map[string]any{
				"name":     "MyConfig",
				"settings": "just a string",
			},
			wantErrors: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errors := ValidateData(tt.data, compiled)
			if len(errors) != tt.wantErrors {
				t.Errorf("ValidateData() got %d errors, want %d", len(errors), tt.wantErrors)
				for _, e := range errors {
					t.Logf("  - %s: %s", e.Field, e.Message)
				}
			}
		})
	}
}

func TestCompile_Checksum(t *testing.T) {
	source1 := `
		type Article {
			title: String!
			body: Text!
		}
	`

	source2 := `
		type Article {
			title: String!
			body: Text!
		}
	`

	source3 := `
		type Article {
			title: String!
			content: Text!
		}
	`

	compiled1, err := ParseAndCompile(source1, "Article", "article", false)
	if err != nil {
		t.Fatalf("ParseAndCompile() error = %v", err)
	}

	compiled2, err := ParseAndCompile(source2, "Article", "article", false)
	if err != nil {
		t.Fatalf("ParseAndCompile() error = %v", err)
	}

	compiled3, err := ParseAndCompile(source3, "Article", "article", false)
	if err != nil {
		t.Fatalf("ParseAndCompile() error = %v", err)
	}

	// Same source should produce same checksum
	if compiled1.Checksum != compiled2.Checksum {
		t.Error("identical sources should produce identical checksums")
	}

	// Different source should produce different checksum
	if compiled1.Checksum == compiled3.Checksum {
		t.Error("different sources should produce different checksums")
	}
}

func TestLexer_Comments(t *testing.T) {
	source := `
		// This is a comment
		type Article {
			// Field comment
			title: String!

			/* Block comment */
			body: Text!
		}
	`

	_, err := Parse(source)
	if err != nil {
		t.Errorf("Parse() with comments failed: %v", err)
	}
}

func TestParse_MultipleTypes(t *testing.T) {
	source := `
		type Author {
			name: String!
			email: String!
		}

		type Article {
			title: String!
			author: Author!
		}
	`

	schema, err := Parse(source)
	if err != nil {
		t.Errorf("Parse() multiple types failed: %v", err)
		return
	}

	if len(schema.Types) != 2 {
		t.Errorf("expected 2 types, got %d", len(schema.Types))
	}
}

func TestValidateData_TextNewlines(t *testing.T) {
	source := `
		type Post {
			title: String!
			content: Text!
		}
	`

	compiled, err := ParseAndCompile(source, "Post", "post", false)
	if err != nil {
		t.Fatalf("ParseAndCompile() error = %v", err)
	}

	tests := []struct {
		name       string
		data       map[string]any
		wantErrors int
	}{
		{
			name: "Text allows newlines",
			data: map[string]any{
				"title":   "My Post",
				"content": "Line 1\nLine 2\nLine 3",
			},
			wantErrors: 0,
		},
		{
			name: "String rejects newlines",
			data: map[string]any{
				"title":   "My\nPost",
				"content": "Content here",
			},
			wantErrors: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errors := ValidateData(tt.data, compiled)
			if len(errors) != tt.wantErrors {
				t.Errorf("ValidateData() got %d errors, want %d", len(errors), tt.wantErrors)
				for _, e := range errors {
					t.Logf("  - %s: %s", e.Field, e.Message)
				}
			}
		})
	}
}

func TestValidateData_RequiredArrays(t *testing.T) {
	source := `
		type Article {
			title: String!
			tags: [String]!
			categories: [String!]!
		}
	`

	compiled, err := ParseAndCompile(source, "Article", "article", false)
	if err != nil {
		t.Fatalf("ParseAndCompile() error = %v", err)
	}

	tests := []struct {
		name       string
		data       map[string]any
		wantErrors int
	}{
		{
			name: "valid with non-empty arrays",
			data: map[string]any{
				"title":      "Hello",
				"tags":       []any{"tech"},
				"categories": []any{"news"},
			},
			wantErrors: 0,
		},
		{
			name: "missing required array",
			data: map[string]any{
				"title":      "Hello",
				"categories": []any{"news"},
			},
			wantErrors: 1, // missing tags
		},
		{
			name: "empty required array with required elements",
			data: map[string]any{
				"title":      "Hello",
				"tags":       []any{},
				"categories": []any{}, // Both fail - empty arrays with required elements
			},
			wantErrors: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errors := ValidateData(tt.data, compiled)
			if len(errors) != tt.wantErrors {
				t.Errorf("ValidateData() got %d errors, want %d", len(errors), tt.wantErrors)
				for _, e := range errors {
					t.Logf("  - %s: %s", e.Field, e.Message)
				}
			}
		})
	}
}
