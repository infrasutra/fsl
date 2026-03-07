package lsp

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetPrepareRename_TypeName(t *testing.T) {
	doc := newDoc("type Article {\n  title: String!\n}")

	result := GetPrepareRename(doc, Position{Line: 0, Character: 6})
	require.NotNil(t, result, "should allow renaming type name")
	assert.Equal(t, "Article", result.Placeholder)
	assert.Equal(t, 0, result.Range.Start.Line)
}

func TestGetPrepareRename_EnumName(t *testing.T) {
	doc := newDoc("enum Status {\n  draft\n  published\n}")

	result := GetPrepareRename(doc, Position{Line: 0, Character: 6})
	require.NotNil(t, result, "should allow renaming enum name")
	assert.Equal(t, "Status", result.Placeholder)
}

func TestGetPrepareRename_FieldName_NotRenameable(t *testing.T) {
	doc := newDoc("type Post {\n  title: String!\n}")

	result := GetPrepareRename(doc, Position{Line: 1, Character: 3})
	assert.Nil(t, result, "field names should not be renameable")
}

func TestGetPrepareRename_EmptyPosition(t *testing.T) {
	doc := newDoc("type Post {\n  title: String!\n}")

	result := GetPrepareRename(doc, Position{Line: 2, Character: 0})
	assert.Nil(t, result, "closing brace position should not be renameable")
}

func TestGetPrepareRename_Keyword(t *testing.T) {
	doc := newDoc("type Post {\n  title: String!\n}")

	result := GetPrepareRename(doc, Position{Line: 0, Character: 1})
	assert.Nil(t, result, "keyword 'type' should not be renameable")
}

func TestGetRename_Type(t *testing.T) {
	doc := newDoc("type Article {\n  title: String!\n  author: Article\n}")

	edit := GetRename(doc, Position{Line: 0, Character: 6}, "BlogPost")
	require.NotNil(t, edit, "rename should produce edits")
	require.Contains(t, edit.Changes, doc.URI)

	edits := edit.Changes[doc.URI]
	assert.GreaterOrEqual(t, len(edits), 2, "should rename at least the definition and usage")

	// Verify all edits replace with new name
	for _, e := range edits {
		assert.Equal(t, "BlogPost", e.NewText)
	}
}

func TestGetRename_Enum(t *testing.T) {
	doc := newDoc("enum Status {\n  draft\n  published\n}\n\ntype Post {\n  status: Status\n}")

	edit := GetRename(doc, Position{Line: 0, Character: 6}, "PostStatus")
	require.NotNil(t, edit)

	edits := edit.Changes[doc.URI]
	assert.GreaterOrEqual(t, len(edits), 2, "should rename enum definition and field type reference")

	for _, e := range edits {
		assert.Equal(t, "PostStatus", e.NewText)
	}
}

func TestGetRename_NonRenameable(t *testing.T) {
	doc := newDoc("type Post {\n  title: String!\n}")

	edit := GetRename(doc, Position{Line: 1, Character: 3}, "heading")
	assert.Nil(t, edit, "field names should not produce rename edits")
}

func TestGetRename_NilDoc(t *testing.T) {
	doc := newDoc("type Post {\n  title: String!\n}")

	// Position on keyword "type"
	edit := GetRename(doc, Position{Line: 0, Character: 1}, "NewName")
	assert.Nil(t, edit, "keyword should not produce rename edits")
}

func TestFindRenameEdits_WholeWord(t *testing.T) {
	doc := newDoc("type Post {\n  title: String!\n  postId: Int\n}")

	edits := findRenameEdits(doc, "Post", "Article")

	// Should match "Post" on line 0 but NOT "postId" on line 2 (different case, but let's verify whole word)
	for _, e := range edits {
		// Each edit range should be exactly 4 characters (length of "Post")
		assert.Equal(t, e.Range.End.Character-e.Range.Start.Character, len("Post"))
		assert.Equal(t, "Article", e.NewText)
	}
}
