package parser

import (
	"testing"
)

// Phase 2 Tests: Enums, Relations, New Types, Schema Diff

func TestParse_InlineEnum(t *testing.T) {
	tests := []struct {
		name    string
		source  string
		wantErr bool
	}{
		{
			name: "inline enum field",
			source: `
				type Article {
					title: String!
					status: "draft" | "published" | "archived"
				}
			`,
			wantErr: false,
		},
		{
			name: "required inline enum",
			source: `
				type Article {
					status: "draft" | "published"!
				}
			`,
			wantErr: false,
		},
		{
			name: "array of inline enums",
			source: `
				type Article {
					tags: ["tech" | "news" | "sports"]
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

func TestParse_NamedEnum(t *testing.T) {
	tests := []struct {
		name    string
		source  string
		wantErr bool
	}{
		{
			name: "named enum definition",
			source: `
				enum Status {
					draft,
					published,
					archived
				}

				type Article {
					title: String!
					status: Status!
				}
			`,
			wantErr: false,
		},
		{
			name: "multiple enums",
			source: `
				enum Status {
					draft
					published
				}

				enum Priority {
					low
					medium
					high
				}

				type Task {
					name: String!
					status: Status!
					priority: Priority
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

func TestParse_Relations(t *testing.T) {
	tests := []struct {
		name    string
		source  string
		wantErr bool
	}{
		{
			name: "simple relation",
			source: `
				type Author {
					name: String!
				}

				type Article {
					title: String!
					author: Author! @relation
				}
			`,
			wantErr: false,
		},
		{
			name: "bidirectional relation",
			source: `
				type Author {
					name: String!
					articles: [Article!]! @relation(inverse: "author")
				}

				type Article {
					title: String!
					author: Author! @relation(inverse: "articles")
				}
			`,
			wantErr: false,
		},
		{
			name: "self-referential relation",
			source: `
				type Category {
					name: String!
					parent: Category @relation
					children: [Category] @relation(inverse: "parent")
				}
			`,
			wantErr: false,
		},
		{
			name: "array relation",
			source: `
				type Tag {
					name: String!
				}

				type Article {
					title: String!
					tags: [Tag!]! @relation
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

func TestParse_NewTypes(t *testing.T) {
	tests := []struct {
		name    string
		source  string
		wantErr bool
	}{
		{
			name: "Date type",
			source: `
				type Event {
					name: String!
					eventDate: Date!
				}
			`,
			wantErr: false,
		},
		{
			name: "RichText type",
			source: `
				type Article {
					title: String!
					content: RichText!
				}
			`,
			wantErr: false,
		},
		{
			name: "Image type",
			source: `
				type Article {
					title: String!
					featuredImage: Image
				}
			`,
			wantErr: false,
		},
		{
			name: "File type",
			source: `
				type Document {
					name: String!
					attachment: File!
				}
			`,
			wantErr: false,
		},
		{
			name: "all new types combined",
			source: `
				type Post {
					title: String!
					publishDate: Date
					content: RichText!
					thumbnail: Image
					downloadable: File
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

func TestParse_NewDecorators(t *testing.T) {
	tests := []struct {
		name    string
		source  string
		wantErr bool
	}{
		{
			name: "blocks decorator for RichText",
			source: `
				type Article {
					content: RichText! @blocks("paragraph", "heading", "image")
				}
			`,
			wantErr: false,
		},
		{
			name: "maxSize decorator for Image",
			source: `
				type Article {
					image: Image! @maxSize(5000000)
				}
			`,
			wantErr: false,
		},
		{
			name: "formats decorator for Image",
			source: `
				type Article {
					image: Image! @formats("jpg", "png", "webp")
				}
			`,
			wantErr: false,
		},
		{
			name: "precision decorator for Float",
			source: `
				type Product {
					price: Float! @precision(2)
				}
			`,
			wantErr: false,
		},
		{
			name: "minItems and maxItems for arrays",
			source: `
				type Article {
					tags: [String!]! @minItems(1) @maxItems(10)
				}
			`,
			wantErr: false,
		},
		{
			name: "hidden decorator",
			source: `
				type User {
					name: String!
					passwordHash: String! @hidden
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

func TestParse_TypeLevelDecorators(t *testing.T) {
	tests := []struct {
		name    string
		source  string
		wantErr bool
	}{
		{
			name: "collection decorator",
			source: `
				@collection("blog_posts")
				type BlogPost {
					title: String!
				}
			`,
			wantErr: false,
		},
		{
			name: "singleton decorator",
			source: `
				@singleton
				type SiteSettings {
					siteName: String!
					tagline: String
				}
			`,
			wantErr: false,
		},
		{
			name: "multiple type decorators",
			source: `
				@collection("settings")
				@singleton
				type GlobalConfig {
					theme: String!
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

func TestValidateData_Date(t *testing.T) {
	source := `
		type Event {
			name: String!
			eventDate: Date!
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
			name: "valid date",
			data: map[string]any{
				"name":      "Conference",
				"eventDate": "2024-01-15",
			},
			wantErrors: 0,
		},
		{
			name: "invalid date format",
			data: map[string]any{
				"name":      "Conference",
				"eventDate": "01/15/2024",
			},
			wantErrors: 1,
		},
		{
			name: "datetime instead of date",
			data: map[string]any{
				"name":      "Conference",
				"eventDate": "2024-01-15T10:00:00Z",
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

func TestValidateData_InlineEnum(t *testing.T) {
	source := `
		type Article {
			title: String!
			status: "draft" | "published" | "archived"!
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
			name: "valid enum value",
			data: map[string]any{
				"title":  "Hello",
				"status": "draft",
			},
			wantErrors: 0,
		},
		{
			name: "another valid enum value",
			data: map[string]any{
				"title":  "Hello",
				"status": "published",
			},
			wantErrors: 0,
		},
		{
			name: "invalid enum value",
			data: map[string]any{
				"title":  "Hello",
				"status": "pending",
			},
			wantErrors: 1,
		},
		{
			name: "wrong type for enum",
			data: map[string]any{
				"title":  "Hello",
				"status": 123,
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

func TestValidateData_RichText(t *testing.T) {
	source := `
		type Article {
			title: String!
			content: RichText!
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
			name: "valid RichText blocks",
			data: map[string]any{
				"title": "Hello",
				"content": []any{
					map[string]any{"type": "paragraph", "children": []any{}},
					map[string]any{"type": "heading", "level": float64(1)},
				},
			},
			wantErrors: 0,
		},
		{
			name: "RichText not an array",
			data: map[string]any{
				"title":   "Hello",
				"content": "plain text",
			},
			wantErrors: 1,
		},
		{
			name: "RichText block without type",
			data: map[string]any{
				"title": "Hello",
				"content": []any{
					map[string]any{"children": []any{}},
				},
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

func TestValidateData_Image(t *testing.T) {
	source := `
		type Article {
			title: String!
			image: Image!
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
			name: "valid Image reference",
			data: map[string]any{
				"title": "Hello",
				"image": map[string]any{
					"url":      "https://example.com/image.jpg",
					"width":    float64(800),
					"height":   float64(600),
					"filename": "image.jpg",
				},
			},
			wantErrors: 0,
		},
		{
			name: "Image without url",
			data: map[string]any{
				"title": "Hello",
				"image": map[string]any{
					"width":  float64(800),
					"height": float64(600),
				},
			},
			wantErrors: 1,
		},
		{
			name: "Image not an object",
			data: map[string]any{
				"title": "Hello",
				"image": "https://example.com/image.jpg",
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

func TestValidateData_ArrayConstraints(t *testing.T) {
	source := `
		type Article {
			title: String!
			tags: [String!]! @minItems(1) @maxItems(5)
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
			name: "valid array size",
			data: map[string]any{
				"title": "Hello",
				"tags":  []any{"tech", "news"},
			},
			wantErrors: 0,
		},
		{
			name: "empty array violates minItems",
			data: map[string]any{
				"title": "Hello",
				"tags":  []any{},
			},
			wantErrors: 2, // One for minItems, one for ArrayReq
		},
		{
			name: "too many items",
			data: map[string]any{
				"title": "Hello",
				"tags":  []any{"a", "b", "c", "d", "e", "f"},
			},
			wantErrors: 1,
		},
		{
			name: "boundary - exactly maxItems",
			data: map[string]any{
				"title": "Hello",
				"tags":  []any{"a", "b", "c", "d", "e"},
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

func TestValidateData_Relation(t *testing.T) {
	source := `
		type Author {
			name: String!
		}

		type Article {
			title: String!
			author: Author! @relation
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
			name: "valid UUID reference",
			data: map[string]any{
				"title":  "Hello",
				"author": "550e8400-e29b-41d4-a716-446655440000",
			},
			wantErrors: 0,
		},
		{
			name: "valid object reference",
			data: map[string]any{
				"title": "Hello",
				"author": map[string]any{
					"id": "550e8400-e29b-41d4-a716-446655440000",
				},
			},
			wantErrors: 0,
		},
		{
			name: "invalid UUID format",
			data: map[string]any{
				"title":  "Hello",
				"author": "not-a-uuid",
			},
			wantErrors: 1,
		},
		{
			name: "object without id",
			data: map[string]any{
				"title": "Hello",
				"author": map[string]any{
					"name": "John",
				},
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

func TestSchemaDiff_Basic(t *testing.T) {
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
			author: String
		}
	`

	compiled1, _ := ParseAndCompile(source1, "Article", "article", false)
	compiled2, _ := ParseAndCompile(source2, "Article", "article", false)

	diff := DiffSchemas(compiled1, compiled2)

	if len(diff.Changes) == 0 {
		t.Error("expected changes in diff")
	}

	// Should have one added field
	foundAddedField := false
	for _, change := range diff.Changes {
		if change.Type == ChangeTypeAdded && change.Kind == ChangeKindField && change.FieldName == "author" {
			foundAddedField = true
		}
	}

	if !foundAddedField {
		t.Error("expected to find added 'author' field")
	}
}

func TestSchemaDiff_BreakingChanges(t *testing.T) {
	source1 := `
		type Article {
			title: String
			body: Text!
		}
	`

	source2 := `
		type Article {
			title: String!
			body: Text!
		}
	`

	compiled1, _ := ParseAndCompile(source1, "Article", "article", false)
	compiled2, _ := ParseAndCompile(source2, "Article", "article", false)

	diff := DiffSchemas(compiled1, compiled2)

	if !diff.HasBreaking {
		t.Error("expected diff to have breaking changes (making field required)")
	}

	// Find the change
	for _, change := range diff.Changes {
		if change.FieldName == "title" && change.Type == ChangeTypeModified {
			if !change.Breaking {
				t.Error("making field required should be marked as breaking")
			}
		}
	}
}

func TestSchemaDiff_RemovedField(t *testing.T) {
	source1 := `
		type Article {
			title: String!
			body: Text!
			author: String
		}
	`

	source2 := `
		type Article {
			title: String!
			body: Text!
		}
	`

	compiled1, _ := ParseAndCompile(source1, "Article", "article", false)
	compiled2, _ := ParseAndCompile(source2, "Article", "article", false)

	diff := DiffSchemas(compiled1, compiled2)

	if !diff.HasBreaking {
		t.Error("expected diff to have breaking changes (removed field)")
	}

	foundRemovedField := false
	for _, change := range diff.Changes {
		if change.Type == ChangeTypeRemoved && change.FieldName == "author" {
			foundRemovedField = true
			if !change.Breaking {
				t.Error("removed field should be marked as breaking")
			}
		}
	}

	if !foundRemovedField {
		t.Error("expected to find removed 'author' field")
	}
}

func TestSchemaDiff_NoChanges(t *testing.T) {
	source := `
		type Article {
			title: String!
			body: Text!
		}
	`

	compiled1, _ := ParseAndCompile(source, "Article", "article", false)
	compiled2, _ := ParseAndCompile(source, "Article", "article", false)

	diff := DiffSchemas(compiled1, compiled2)

	if len(diff.Changes) != 0 {
		t.Errorf("expected no changes, got %d", len(diff.Changes))
	}

	if diff.HasBreaking {
		t.Error("expected no breaking changes")
	}
}

func TestCompile_WithEnums(t *testing.T) {
	source := `
		enum Status {
			draft,
			published,
			archived
		}

		type Article {
			title: String!
			status: Status!
		}
	`

	compiled, err := ParseAndCompile(source, "Article", "article", false)
	if err != nil {
		t.Fatalf("ParseAndCompile() error = %v", err)
	}

	if len(compiled.Enums) != 1 {
		t.Errorf("expected 1 enum, got %d", len(compiled.Enums))
	}

	if compiled.Enums[0].Name != "Status" {
		t.Errorf("expected enum name 'Status', got '%s'", compiled.Enums[0].Name)
	}

	if len(compiled.Enums[0].Values) != 3 {
		t.Errorf("expected 3 enum values, got %d", len(compiled.Enums[0].Values))
	}
}

func TestCompile_WithRelations(t *testing.T) {
	source := `
		type Author {
			name: String!
		}

		type Article {
			title: String!
			author: Author! @relation
			tags: [Tag!]! @relation
		}

		type Tag {
			name: String!
		}
	`

	compiled, err := ParseAndCompile(source, "Article", "article", false)
	if err != nil {
		t.Fatalf("ParseAndCompile() error = %v", err)
	}

	if len(compiled.Relations) != 2 {
		t.Errorf("expected 2 relations, got %d", len(compiled.Relations))
	}

	// Check author relation
	var authorRel *CompiledRelation
	for i := range compiled.Relations {
		if compiled.Relations[i].FieldName == "author" {
			authorRel = &compiled.Relations[i]
			break
		}
	}

	if authorRel == nil {
		t.Fatal("expected to find 'author' relation")
	}

	if authorRel.TargetType != "Author" {
		t.Errorf("expected author relation target 'Author', got '%s'", authorRel.TargetType)
	}

	if authorRel.IsArray {
		t.Error("author relation should not be an array")
	}
}

func TestAutoRelationDetection(t *testing.T) {
	tests := []struct {
		name           string
		source         string
		typeName       string
		expectedRelns  int
		checkField     string
		expectRelation bool
	}{
		{
			name: "auto-detect simple relation without @relation",
			source: `
				type Author {
					name: String!
				}

				type Article {
					title: String!
					author: Author!
				}
			`,
			typeName:       "Article",
			expectedRelns:  1,
			checkField:     "author",
			expectRelation: true,
		},
		{
			name: "auto-detect array relation without @relation",
			source: `
				type Tag {
					name: String!
				}

				type Article {
					title: String!
					tags: [Tag!]!
				}
			`,
			typeName:       "Article",
			expectedRelns:  1,
			checkField:     "tags",
			expectRelation: true,
		},
		{
			name: "self-referential relation",
			source: `
				type Category {
					name: String!
					parent: Category
					children: [Category]
				}
			`,
			typeName:       "Category",
			expectedRelns:  2,
			checkField:     "parent",
			expectRelation: true,
		},
		{
			name: "forward reference relation",
			source: `
				type Article {
					title: String!
					author: Author!
				}

				type Author {
					name: String!
				}
			`,
			typeName:       "Article",
			expectedRelns:  1,
			checkField:     "author",
			expectRelation: true,
		},
		{
			name: "builtin type NOT a relation",
			source: `
				type Article {
					title: String!
					body: Text!
				}
			`,
			typeName:       "Article",
			expectedRelns:  0,
			checkField:     "title",
			expectRelation: false,
		},
		{
			name: "enum type NOT a relation",
			source: `
				enum Status {
					draft
					published
				}

				type Article {
					title: String!
					status: Status!
				}
			`,
			typeName:       "Article",
			expectedRelns:  0,
			checkField:     "status",
			expectRelation: false,
		},
		{
			name: "inline enum NOT a relation",
			source: `
				type Article {
					title: String!
					status: "draft" | "published"
				}
			`,
			typeName:       "Article",
			expectedRelns:  0,
			checkField:     "status",
			expectRelation: false,
		},
		{
			name: "explicit @relation still works",
			source: `
				type Author {
					name: String!
					articles: [Article!]! @relation(inverse: "author")
				}

				type Article {
					title: String!
					author: Author! @relation(inverse: "articles")
				}
			`,
			typeName:       "Article",
			expectedRelns:  1,
			checkField:     "author",
			expectRelation: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			compiled, err := ParseAndCompile(tt.source, tt.typeName, "test", false)
			if err != nil {
				t.Fatalf("ParseAndCompile() error = %v", err)
			}

			if len(compiled.Relations) != tt.expectedRelns {
				t.Errorf("expected %d relations, got %d", tt.expectedRelns, len(compiled.Relations))
			}

			// Check specific field
			var foundField *CompiledField
			for i := range compiled.Fields {
				if compiled.Fields[i].Name == tt.checkField {
					foundField = &compiled.Fields[i]
					break
				}
			}

			if foundField == nil {
				t.Fatalf("field '%s' not found", tt.checkField)
			}

			if foundField.IsRelation != tt.expectRelation {
				t.Errorf("field '%s' IsRelation = %v, want %v", tt.checkField, foundField.IsRelation, tt.expectRelation)
			}
		})
	}
}

func TestAutoRelationDetection_ParsedSchema(t *testing.T) {
	// Test that auto-detection works at the schema validation level
	source := `
		type Author {
			name: String!
		}

		type Article {
			title: String!
			author: Author!
		}
	`

	schema, err := Parse(source)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	// Validate to trigger auto-detection
	err = ValidateSchema(schema)
	if err != nil {
		t.Fatalf("ValidateSchema() error = %v", err)
	}

	// Find the Article type and check the author field
	var articleType *TypeDef
	for i := range schema.Types {
		if schema.Types[i].Name == "Article" {
			articleType = &schema.Types[i]
			break
		}
	}

	if articleType == nil {
		t.Fatal("Article type not found")
	}

	var authorField *FieldDef
	for i := range articleType.Fields {
		if articleType.Fields[i].Name == "author" {
			authorField = &articleType.Fields[i]
			break
		}
	}

	if authorField == nil {
		t.Fatal("author field not found")
	}

	if !authorField.IsRelation {
		t.Error("author field should have IsRelation = true after validation")
	}
}

func TestExternalTypes_CrossSchemaReferences(t *testing.T) {
	// Test that external types work for cross-schema references (like in templates)
	// This simulates a template where Article references Journalist and NewsCategory

	articleFSL := `
		type Article {
			headline: String!
			journalist: Journalist! @relation
			category: NewsCategory! @relation
		}
	`

	// External types from other schemas in the template
	externalTypes := []string{"Journalist", "NewsCategory"}

	// This should succeed because Journalist and NewsCategory are external types
	compiled, err := ParseAndCompileWithExternalTypes(articleFSL, "Article", "article", false, externalTypes)
	if err != nil {
		t.Fatalf("ParseAndCompileWithExternalTypes() error = %v", err)
	}

	// Check that relations are properly detected
	if len(compiled.Relations) != 2 {
		t.Errorf("expected 2 relations, got %d", len(compiled.Relations))
	}

	// Check journalist relation
	var journalistRel *CompiledRelation
	for i := range compiled.Relations {
		if compiled.Relations[i].FieldName == "journalist" {
			journalistRel = &compiled.Relations[i]
			break
		}
	}

	if journalistRel == nil {
		t.Fatal("expected to find 'journalist' relation")
	}

	if journalistRel.TargetType != "Journalist" {
		t.Errorf("expected journalist relation target 'Journalist', got '%s'", journalistRel.TargetType)
	}
}

func TestExternalTypes_FailsWithoutExternalTypes(t *testing.T) {
	// Test that without external types, referencing unknown types fails
	articleFSL := `
		type Article {
			headline: String!
			journalist: Journalist!
		}
	`

	// Without external types, this should fail
	_, err := ParseAndCompile(articleFSL, "Article", "article", false)
	if err == nil {
		t.Fatal("expected error when referencing unknown type without external types")
	}

	// Error should mention unknown type
	if !contains(err.Error(), "unknown type") {
		t.Errorf("expected error to mention 'unknown type', got: %v", err)
	}
}

func TestExternalTypes_AutoDetectRelation(t *testing.T) {
	// Test that auto-relation detection works with external types (without explicit @relation)
	articleFSL := `
		type Article {
			headline: String!
			journalist: Journalist!
			category: NewsCategory
		}
	`

	externalTypes := []string{"Journalist", "NewsCategory"}

	compiled, err := ParseAndCompileWithExternalTypes(articleFSL, "Article", "article", false, externalTypes)
	if err != nil {
		t.Fatalf("ParseAndCompileWithExternalTypes() error = %v", err)
	}

	// Both fields should be auto-detected as relations
	if len(compiled.Relations) != 2 {
		t.Errorf("expected 2 auto-detected relations, got %d", len(compiled.Relations))
	}

	// Check that journalist field is marked as relation
	var journalistField *CompiledField
	for i := range compiled.Fields {
		if compiled.Fields[i].Name == "journalist" {
			journalistField = &compiled.Fields[i]
			break
		}
	}

	if journalistField == nil {
		t.Fatal("journalist field not found")
	}

	if !journalistField.IsRelation {
		t.Error("journalist field should be auto-detected as relation")
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
