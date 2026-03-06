package lsp

import (
	"encoding/json"
	"testing"

	"github.com/infrasutra/fsl/parser"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// helpers
// ---------------------------------------------------------------------------

func newDoc(content string) *Document {
	doc := NewDocument("file:///test.fsl", content, 1)
	doc.parse()
	return doc
}

const blogSchema = `
@icon("newspaper")
@description("Blog articles")
type Article {
  title: String! @maxLength(200)
  slug: String! @pattern("^[a-z0-9-]+$") @unique
  body: RichText
  views: Int @min(0)
  featured: Boolean @default(false)
  publishedAt: DateTime
  category: Category @relation
}

type Category {
  name: String!
  slug: String! @unique
}

enum Status {
  draft
  published
  archived
}
`

// ===========================================================================
// Document
// ===========================================================================

func TestNewDocument(t *testing.T) {
	doc := NewDocument("file:///test.fsl", "type Post { title: String! }", 1)
	assert.Equal(t, "file:///test.fsl", doc.URI)
	assert.Equal(t, "type Post { title: String! }", doc.Content)
	assert.Equal(t, 1, doc.Version)
}

func TestDocument_Parse(t *testing.T) {
	doc := newDoc("type Post { title: String! }")
	assert.NotNil(t, doc.GetSchema())
	assert.NotNil(t, doc.GetParseResult())
	assert.True(t, doc.GetParseResult().Valid)
}

func TestDocument_ParseInvalid(t *testing.T) {
	doc := newDoc("type Post { title: ")
	result := doc.GetParseResult()
	assert.NotNil(t, result)
	assert.False(t, result.Valid)
}

func TestDocument_GetWordAt(t *testing.T) {
	doc := newDoc("type Article {\n  title: String!\n}")

	t.Run("word on type name", func(t *testing.T) {
		word := doc.GetWordAt(Position{Line: 0, Character: 6})
		assert.Equal(t, "Article", word)
	})

	t.Run("word on field type", func(t *testing.T) {
		word := doc.GetWordAt(Position{Line: 1, Character: 10})
		assert.Equal(t, "String", word)
	})

	t.Run("empty position", func(t *testing.T) {
		word := doc.GetWordAt(Position{Line: 0, Character: 4})
		assert.Equal(t, "type", word)
	})
}

func TestDocument_GetLine(t *testing.T) {
	doc := newDoc("line0\nline1\nline2")
	assert.Equal(t, "line0", doc.GetLine(0))
	assert.Equal(t, "line1", doc.GetLine(1))
	assert.Equal(t, "line2", doc.GetLine(2))
	assert.Equal(t, "", doc.GetLine(99))
}

// ===========================================================================
// DocumentStore
// ===========================================================================

func TestDocumentStore(t *testing.T) {
	store := NewDocumentStore()

	t.Run("open and get", func(t *testing.T) {
		store.Open("file:///a.fsl", "type A { x: String! }", 1)
		doc := store.Get("file:///a.fsl")
		require.NotNil(t, doc)
		assert.Equal(t, "type A { x: String! }", doc.Content)
	})

	t.Run("update replaces content", func(t *testing.T) {
		store.Update("file:///a.fsl", "type A { y: Int! }", 2)
		doc := store.Get("file:///a.fsl")
		require.NotNil(t, doc)
		assert.Equal(t, "type A { y: Int! }", doc.Content)
		assert.Equal(t, 2, doc.Version)
	})

	t.Run("close removes document", func(t *testing.T) {
		store.Close("file:///a.fsl")
		doc := store.Get("file:///a.fsl")
		assert.Nil(t, doc)
	})

	t.Run("get nonexistent returns nil", func(t *testing.T) {
		assert.Nil(t, store.Get("file:///nonexistent.fsl"))
	})

	t.Run("all returns all open docs", func(t *testing.T) {
		store.Open("file:///x.fsl", "type X {}", 1)
		store.Open("file:///y.fsl", "type Y {}", 1)
		all := store.All()
		assert.Len(t, all, 2)
	})
}

// ===========================================================================
// Diagnostics
// ===========================================================================

func TestGetDiagnostics_ValidSchema(t *testing.T) {
	doc := newDoc("type Post { title: String! }")
	diags := GetDiagnostics(doc, []*Document{doc})
	assert.Empty(t, diags)
}

func TestGetDiagnostics_SyntaxError(t *testing.T) {
	doc := newDoc("type Post { title: ")
	diags := GetDiagnostics(doc, []*Document{doc})
	assert.NotEmpty(t, diags)
	assert.Equal(t, SeverityError, diags[0].Severity)
}

func TestGetDiagnostics_UnknownType(t *testing.T) {
	doc := newDoc("type Post { author: UnknownType! }")
	diags := GetDiagnostics(doc, []*Document{doc})
	assert.NotEmpty(t, diags)
}

func TestGetDiagnostics_EmptyDocument(t *testing.T) {
	doc := newDoc("")
	diags := GetDiagnostics(doc, []*Document{doc})
	assert.Empty(t, diags)
}

func TestGetDiagnostics_MultipleTypes(t *testing.T) {
	doc := newDoc(blogSchema)
	diags := GetDiagnostics(doc, []*Document{doc})
	// Unreferenced enum produces a warning (severity 2), not an error
	for _, d := range diags {
		assert.NotEqual(t, SeverityError, d.Severity,
			"should not have errors, got: %v", d)
	}
}

func TestGetDiagnostics_CrossFileTypeReference(t *testing.T) {
	articleDoc := NewDocument("file:///article.fsl", "type Article {\n  author: Author! @relation\n}", 1)
	authorDoc := NewDocument("file:///author.fsl", "type Author {\n  name: String!\n}", 1)

	diags := GetDiagnostics(articleDoc, []*Document{articleDoc, authorDoc})
	for _, d := range diags {
		assert.NotEqual(t, SeverityError, d.Severity, "should not have unknown type error: %v", d)
	}
}

func TestConvertDiagnostic(t *testing.T) {
	t.Run("maps all fields", func(t *testing.T) {
		d := ConvertDiagnostic(fslDiag(1, 1, 5, 1, 10, "test error"))
		assert.Equal(t, SeverityError, d.Severity)
		assert.Equal(t, "test error", d.Message)
		// parser uses 1-based lines; LSP uses 0-based
		assert.Equal(t, 0, d.Range.Start.Line)      // 1-1=0
		assert.Equal(t, 4, d.Range.Start.Character) // startCol-1
	})
}

// ===========================================================================
// Completions
// ===========================================================================

func TestGetCompletions_FieldType(t *testing.T) {
	doc := newDoc("type Post {\n  title: \n}")
	completions := GetCompletions(doc, Position{Line: 1, Character: 9})
	require.NotNil(t, completions)
	assert.NotEmpty(t, completions.Items, "should suggest types after colon")
}

func TestGetCompletions_TopLevel(t *testing.T) {
	doc := newDoc("type Post {\n  title: String!\n}\n")
	completions := GetCompletions(doc, Position{Line: 3, Character: 0})
	require.NotNil(t, completions)
	// Should suggest 'type' and 'enum' keywords at top level
	hasType := false
	for _, item := range completions.Items {
		if item.Label == "type" {
			hasType = true
		}
	}
	assert.True(t, hasType, "should suggest 'type' keyword at top level")
}

func TestGetCompletions_Decorator(t *testing.T) {
	doc := newDoc("type Post {\n  title: String! @\n}")
	completions := GetCompletions(doc, Position{Line: 1, Character: 18})
	require.NotNil(t, completions)
	assert.NotEmpty(t, completions.Items, "should suggest decorators after @")

	hasSlices := false
	for _, item := range completions.Items {
		if item.Label == "@slices" {
			hasSlices = true
			break
		}
	}
	assert.True(t, hasSlices, "should suggest @slices decorator")
}

func TestGetBuiltinTypes(t *testing.T) {
	types := GetBuiltinTypes()
	assert.NotEmpty(t, types)
	assert.Contains(t, types, "String")
	assert.Contains(t, types, "Int")
	assert.Contains(t, types, "Float")
	assert.Contains(t, types, "Boolean")
	assert.Contains(t, types, "DateTime")
	assert.Contains(t, types, "RichText")
	assert.Contains(t, types, "JSON")
}

// ===========================================================================
// Hover
// ===========================================================================

func TestGetHover_TypeKeyword(t *testing.T) {
	doc := newDoc(blogSchema)
	hover := GetHover(doc, Position{Line: 3, Character: 6}) // "Article"
	if hover != nil {
		assert.NotEmpty(t, hover.Contents.Value)
	}
}

func TestGetHover_FieldType(t *testing.T) {
	doc := newDoc("type Post {\n  title: String!\n}")
	hover := GetHover(doc, Position{Line: 1, Character: 10})
	if hover != nil {
		assert.Contains(t, hover.Contents.Value, "String")
	}
}

func TestGetHover_SlicesDecorator(t *testing.T) {
	doc := newDoc("type Page {\n  slices: JSON! @slices(hero: HeroSlice)\n}\n\ntype HeroSlice {\n  heading: String!\n}")
	hover := GetHover(doc, Position{Line: 1, Character: 17}) // @slices
	require.NotNil(t, hover)
	assert.Contains(t, hover.Contents.Value, "@slices")
	assert.Contains(t, hover.Contents.Value, "typed slice-zone")
}

func TestGetHover_NilOnWhitespace(t *testing.T) {
	doc := newDoc("type Post {\n\n}")
	hover := GetHover(doc, Position{Line: 1, Character: 0})
	// Empty line should return nil or empty hover
	if hover != nil {
		// That's fine, just testing it doesn't panic
		t.Logf("hover on empty line: %v", hover.Contents.Value)
	}
}

// ===========================================================================
// Symbols
// ===========================================================================

func TestGetDocumentSymbols_Types(t *testing.T) {
	doc := newDoc(blogSchema)
	symbols := GetDocumentSymbols(doc)
	require.NotEmpty(t, symbols)

	names := make(map[string]bool)
	for _, s := range symbols {
		names[s.Name] = true
	}
	assert.True(t, names["Article"], "should have Article symbol")
	assert.True(t, names["Category"], "should have Category symbol")
	assert.True(t, names["Status"], "should have Status enum symbol")
}

func TestGetDocumentSymbols_Fields(t *testing.T) {
	doc := newDoc("type Post {\n  title: String!\n  slug: String!\n}")
	symbols := GetDocumentSymbols(doc)
	require.Len(t, symbols, 1) // Post type

	// Post should have children (fields)
	assert.NotEmpty(t, symbols[0].Children, "type should have field children")

	fieldNames := make(map[string]bool)
	for _, child := range symbols[0].Children {
		fieldNames[child.Name] = true
	}
	assert.True(t, fieldNames["title"])
	assert.True(t, fieldNames["slug"])
}

func TestGetDocumentSymbols_EnumValues(t *testing.T) {
	doc := newDoc("enum Status {\n  draft\n  published\n  archived\n}")
	symbols := GetDocumentSymbols(doc)
	require.Len(t, symbols, 1)
	assert.Equal(t, "Status", symbols[0].Name)
	assert.NotEmpty(t, symbols[0].Children, "enum should have value children")
}

func TestGetDocumentSymbols_EmptyDocument(t *testing.T) {
	doc := newDoc("")
	symbols := GetDocumentSymbols(doc)
	assert.Empty(t, symbols)
}

func TestHandler_HandleInitialize_EnablesWorkspaceSymbolProvider(t *testing.T) {
	server := &Server{documents: NewDocumentStore()}
	handler := NewHandler(server)

	result, err := handler.handleInitialize(nil)
	require.NoError(t, err)
	assert.Equal(t, true, result.Capabilities.WorkspaceSymbolProvider)
	assert.Equal(t, true, result.Capabilities.DocumentFormattingProvider)
}

func TestHandler_HandleFormatting_ReturnsTextEdit(t *testing.T) {
	server := &Server{documents: NewDocumentStore()}
	handler := NewHandler(server)

	server.GetDocuments().Open("file:///models/article.fsl", `type Article{title:String!@maxLength(200)@minLength(1)}`, 1)

	params, err := json.Marshal(DocumentFormattingParams{
		TextDocument: TextDocumentIdentifier{URI: "file:///models/article.fsl"},
		Options:      FormattingOptions{TabSize: 2, InsertSpaces: true},
	})
	require.NoError(t, err)

	result, err := handler.handleFormatting(params)
	require.NoError(t, err)

	edits, ok := result.([]TextEdit)
	require.True(t, ok)
	require.Len(t, edits, 1)
	assert.Equal(t, "type Article {\n  title: String! @minLength(1) @maxLength(200)\n}\n", edits[0].NewText)
}

func TestHandler_HandleFormatting_InvalidSchemaReturnsNoEdits(t *testing.T) {
	server := &Server{documents: NewDocumentStore()}
	handler := NewHandler(server)

	server.GetDocuments().Open("file:///models/article.fsl", `type Article { title:`, 1)

	params, err := json.Marshal(DocumentFormattingParams{
		TextDocument: TextDocumentIdentifier{URI: "file:///models/article.fsl"},
		Options:      FormattingOptions{TabSize: 2, InsertSpaces: true},
	})
	require.NoError(t, err)

	result, err := handler.handleFormatting(params)
	require.NoError(t, err)

	edits, ok := result.([]TextEdit)
	require.True(t, ok)
	assert.Empty(t, edits)
}

func TestHandler_HandleWorkspaceSymbol_QueryAcrossDocuments(t *testing.T) {
	server := &Server{documents: NewDocumentStore()}
	handler := NewHandler(server)

	server.GetDocuments().Open("file:///models/article.fsl", `
type Article {
  title: String!
}

enum Status {
  draft
  published
}
`, 1)

	server.GetDocuments().Open("file:///models/category.fsl", `
type Category {
  name: String!
}
`, 1)

	params, err := json.Marshal(WorkspaceSymbolParams{Query: "category"})
	require.NoError(t, err)

	result, err := handler.handleRequest("workspace/symbol", params)
	require.NoError(t, err)

	symbols, ok := result.([]SymbolInformation)
	require.True(t, ok)
	require.Len(t, symbols, 1)

	assert.Equal(t, "Category", symbols[0].Name)
	assert.Equal(t, SymbolKindClass, symbols[0].Kind)
	assert.Equal(t, "file:///models/category.fsl", symbols[0].Location.URI)
}

func TestHandler_HandleWorkspaceSymbol_EmptyQueryReturnsAllTopLevelSymbols(t *testing.T) {
	server := &Server{documents: NewDocumentStore()}
	handler := NewHandler(server)

	server.GetDocuments().Open("file:///models/article.fsl", `
type Article {
  title: String!
}

enum Status {
  draft
  published
}
`, 1)

	server.GetDocuments().Open("file:///models/category.fsl", `
type Category {
  name: String!
}
`, 1)

	params, err := json.Marshal(WorkspaceSymbolParams{Query: ""})
	require.NoError(t, err)

	result, err := handler.handleWorkspaceSymbol(params)
	require.NoError(t, err)

	symbols, ok := result.([]SymbolInformation)
	require.True(t, ok)
	require.NotEmpty(t, symbols)

	names := make(map[string]bool)
	for _, symbol := range symbols {
		names[symbol.Name] = true
	}

	assert.True(t, names["Article"])
	assert.True(t, names["Category"])
	assert.True(t, names["Status"])
}

func TestWorkspaceSymbols(t *testing.T) {
	server := &Server{documents: NewDocumentStore()}
	handler := NewHandler(server)

	// Normal formatting
	server.GetDocuments().Open("file:///workspace/article.fsl", `
type Article {
	title: String!
	name: String
}

enum Status {
	draft
	published
}
`, 1)

	// Weird formatting (double spaces, etc.)
	server.GetDocuments().Open("file:///workspace/category.fsl", `
type  Category  {
	name  :  String!
}
`, 1)

	t.Run("find category with weird formatting", func(t *testing.T) {
		params, err := json.Marshal(WorkspaceSymbolParams{Query: "category"})
		require.NoError(t, err)

		result, err := handler.handleWorkspaceSymbol(params)
		require.NoError(t, err)

		symbols, ok := result.([]SymbolInformation)
		require.True(t, ok)
		require.Len(t, symbols, 1, "Should find Category even with weird formatting")
		assert.Equal(t, "Category", symbols[0].Name)
	})

	t.Run("find field with weird formatting", func(t *testing.T) {
		params, err := json.Marshal(WorkspaceSymbolParams{Query: "name"})
		require.NoError(t, err)

		result, err := handler.handleWorkspaceSymbol(params)
		require.NoError(t, err)

		symbols, ok := result.([]SymbolInformation)
		require.True(t, ok)
		require.Len(t, symbols, 2, "Should find both name fields")

		names := make(map[string]string)
		for _, s := range symbols {
			names[s.ContainerName] = s.Name
		}
		assert.Equal(t, "name", names["Article"])
		assert.Equal(t, "name", names["Category"])
	})

	t.Run("empty query returns all", func(t *testing.T) {
		params, err := json.Marshal(WorkspaceSymbolParams{Query: ""})
		require.NoError(t, err)

		result, err := handler.handleWorkspaceSymbol(params)
		require.NoError(t, err)

		symbols, ok := result.([]SymbolInformation)
		require.True(t, ok)
		// Article, title, name, Status, draft, published, Category, name = 8 symbols
		assert.Len(t, symbols, 8)
	})
}

// ===========================================================================
// Definition
// ===========================================================================

func TestGetDefinition_TypeReference(t *testing.T) {
	doc := newDoc(blogSchema)
	// "Category" in "category: Category @relation" — line index depends on exact content
	// Find the line with "category: Category"
	for i := range 20 {
		line := doc.GetLine(i)
		if line == "" {
			continue
		}
		if contains(line, "Category") && contains(line, "@relation") {
			def := GetDefinition(doc, Position{Line: i, Character: indexOf(line, "Category")})
			if def != nil {
				assert.Equal(t, "file:///test.fsl", def.URI)
				t.Logf("definition found at line %d", def.Range.Start.Line)
			}
			return
		}
	}
}

func TestGetReferences_TypeName(t *testing.T) {
	doc := newDoc(blogSchema)
	// Find "Category" type declaration line
	for i := range 20 {
		line := doc.GetLine(i)
		if isTypeDeclaration(line, "Category") {
			refs := GetReferences(doc, Position{Line: i, Character: indexOf(line, "Category")}, true)
			assert.GreaterOrEqual(t, len(refs), 1, "Category should have at least 1 reference")
			return
		}
	}
}

// ===========================================================================
// Helper formatters
// ===========================================================================

func TestFormatFieldType(t *testing.T) {
	assert.Equal(t, "String", formatFieldType("String", false, false))
	assert.Equal(t, "String!", formatFieldType("String", false, true))
	assert.Equal(t, "[String]", formatFieldType("String", true, false))
	assert.Equal(t, "[String]!", formatFieldType("String", true, true))
}

func TestFormatStringSlice(t *testing.T) {
	assert.Equal(t, `["a", "b"]`, formatStringSlice([]string{"a", "b"}))
	assert.Equal(t, `["x"]`, formatStringSlice([]string{"x"}))
	assert.Equal(t, "[]", formatStringSlice([]string{}))
}

func TestFormatDecoratorArg(t *testing.T) {
	assert.Equal(t, `"hello"`, formatDecoratorArg("hello"))
	assert.Equal(t, "42", formatDecoratorArg(42))
	assert.Equal(t, "42", formatDecoratorArg(float64(42)))
	assert.Equal(t, "3.14", formatDecoratorArg(3.14))
	assert.Equal(t, "true", formatDecoratorArg(true))
}

func TestIsWholeWord(t *testing.T) {
	assert.True(t, isWholeWord("type Article {", 5, 7))
	assert.True(t, isWholeWord("Article", 0, 7))
	assert.False(t, isWholeWord("ArticleList", 0, 7))
}

func TestIsTypeDeclaration(t *testing.T) {
	assert.True(t, isTypeDeclaration("type Article {", "Article"))
	assert.True(t, isTypeDeclaration("  type Article {", "Article"))
	assert.False(t, isTypeDeclaration("  author: Article @relation", "Article"))
}

// ===========================================================================
// Helpers
// ===========================================================================

func contains(s, substr string) bool {
	return len(s) >= len(substr) && indexOf(s, substr) >= 0
}

func indexOf(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}

// fslDiag creates a parser.Diagnostic for testing
func fslDiag(severity, startLine, startCol, endLine, endCol int, msg string) parser.Diagnostic {
	return parser.Diagnostic{
		Severity:    parser.DiagnosticSeverity(severity),
		Message:     msg,
		StartLine:   startLine,
		StartColumn: startCol,
		EndLine:     endLine,
		EndColumn:   endCol,
		Source:      "fsl",
	}
}
