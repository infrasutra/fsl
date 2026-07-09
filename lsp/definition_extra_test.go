package lsp

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFindEnumDefinition(t *testing.T) {
	doc := newDoc("enum Status {\n  draft\n  published\n}")

	t.Run("finds definition", func(t *testing.T) {
		loc := findEnumDefinition(doc, "Status")
		require.NotNil(t, loc)
		assert.Equal(t, doc.URI, loc.URI)
		assert.Equal(t, 0, loc.Range.Start.Line)
	})

	t.Run("returns nil for unknown enum", func(t *testing.T) {
		assert.Nil(t, findEnumDefinition(doc, "Missing"))
	})
}

func TestGetDefinition_EmptyWordReturnsNil(t *testing.T) {
	doc := newDoc("type Post {\n\n}")
	def := GetDefinition(doc, Position{Line: 1, Character: 0})
	assert.Nil(t, def)
}

func TestGetDefinition_InvalidSchemaReturnsNil(t *testing.T) {
	doc := newDoc("type Post { title: ")
	def := GetDefinition(doc, Position{Line: 0, Character: 6})
	assert.Nil(t, def)
}

func TestGetDefinition_EnumReference(t *testing.T) {
	doc := newDoc("enum Status {\n  draft\n  published\n}\n\ntype Post {\n  status: Status\n}")
	def := GetDefinition(doc, Position{Line: 6, Character: 10}) // "Status" in field type position
	require.NotNil(t, def)
	assert.Equal(t, 0, def.Range.Start.Line)
}

func TestGetReferences_EmptyWordReturnsNil(t *testing.T) {
	doc := newDoc("type Post {\n\n}")
	refs := GetReferences(doc, Position{Line: 1, Character: 0}, true)
	assert.Nil(t, refs)
}

func TestGetReferences_NonTypeWordReturnsEmpty(t *testing.T) {
	doc := newDoc("type Post { title: String! }")
	refs := GetReferences(doc, Position{Line: 0, Character: 15}, true)
	assert.Empty(t, refs)
}

func TestGetReferences_ExcludesDeclarationWhenRequested(t *testing.T) {
	doc := newDoc("type Post {\n  title: String!\n  author: Post\n}")
	withDecl := GetReferences(doc, Position{Line: 0, Character: 6}, true)
	withoutDecl := GetReferences(doc, Position{Line: 0, Character: 6}, false)
	assert.Greater(t, len(withDecl), len(withoutDecl))
}

func TestIsWholeWord_BoundaryChecks(t *testing.T) {
	assert.True(t, isWholeWord("Post", 0, 4))
	assert.False(t, isWholeWord("PostList", 0, 4), "adjacent word char after should fail")
	assert.False(t, isWholeWord("SuperPost", 5, 4), "adjacent word char before should fail")
}

func TestIsTypeDeclaration_EdgeCases(t *testing.T) {
	assert.True(t, isTypeDeclaration("enum Status {", "Status"))
	assert.False(t, isTypeDeclaration("", "Status"))
	assert.False(t, isTypeDeclaration("field: Status", "Status"))
}
