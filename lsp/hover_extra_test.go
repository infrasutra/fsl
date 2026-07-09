package lsp

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetFieldHover(t *testing.T) {
	doc := newDoc("type Post {\n  title: String!\n  author: Post @relation\n}")

	t.Run("resolves a field within the enclosing type", func(t *testing.T) {
		hover := getFieldHover(doc, Position{Line: 1, Character: 3}, "title")
		require.NotNil(t, hover)
		assert.Contains(t, hover.Contents.Value, "Type: `String`")
		assert.Contains(t, hover.Contents.Value, "Required: yes")
	})

	t.Run("relation field lists decorators", func(t *testing.T) {
		hover := getFieldHover(doc, Position{Line: 2, Character: 3}, "author")
		require.NotNil(t, hover)
		assert.Contains(t, hover.Contents.Value, "Relation: yes")
		assert.Contains(t, hover.Contents.Value, "Decorators:")
	})

	t.Run("empty line returns nil", func(t *testing.T) {
		hover := getFieldHover(newDoc(""), Position{Line: 0, Character: 0}, "title")
		assert.Nil(t, hover)
	})

	t.Run("no colon on line returns nil", func(t *testing.T) {
		hover := getFieldHover(doc, Position{Line: 0, Character: 0}, "type")
		assert.Nil(t, hover)
	})

	t.Run("word does not match field name returns nil", func(t *testing.T) {
		hover := getFieldHover(doc, Position{Line: 1, Character: 3}, "notAField")
		assert.Nil(t, hover)
	})

	t.Run("nil schema returns nil", func(t *testing.T) {
		invalidDoc := newDoc("type Post { title: ")
		hover := getFieldHover(invalidDoc, Position{Line: 0, Character: 12}, "title")
		assert.Nil(t, hover)
	})
}

func TestFindEnclosingTypeName(t *testing.T) {
	t.Run("single line type", func(t *testing.T) {
		doc := newDoc("type Post {\n  title: String!\n}")
		assert.Equal(t, "Post", findEnclosingTypeName(doc, 1))
	})

	t.Run("type name on separate line from brace is a known limitation", func(t *testing.T) {
		// The pending-type tracking resets before the opening brace is seen when
		// the brace isn't on the same line as "type Name", so the enclosing type
		// is not recognized in this layout.
		doc := newDoc("type Post\n{\n  title: String!\n}")
		assert.Equal(t, "", findEnclosingTypeName(doc, 2))
	})

	t.Run("outside any type returns empty", func(t *testing.T) {
		doc := newDoc("type Post {\n  title: String!\n}\n\ntype Author {\n  name: String!\n}")
		assert.Equal(t, "", findEnclosingTypeName(doc, 3))
	})

	t.Run("line beyond document clamps to last line", func(t *testing.T) {
		doc := newDoc("type Post {\n  title: String!\n}")
		assert.Equal(t, "", findEnclosingTypeName(doc, 99))
	})
}

func TestParseTypeName(t *testing.T) {
	assert.Equal(t, "Post", parseTypeName("type Post {"))
	assert.Equal(t, "Post", parseTypeName("type Post"))
	assert.Equal(t, "", parseTypeName(""))
}

func TestFormatDecorator(t *testing.T) {
	assert.Equal(t, "@required", formatDecorator("required", nil))
	assert.Equal(t, "@required", formatDecorator("required", true))
	assert.Equal(t, `@default("draft")`, formatDecorator("default", "draft"))
	assert.Equal(t, "@min(0)", formatDecorator("min", 0))
	assert.Equal(t, "@precision(2.5)", formatDecorator("precision", 2.5))
	assert.Equal(t, `@formats(["jpg", "png"])`, formatDecorator("formats", []string{"jpg", "png"}))
	assert.Equal(t, `@slices([1, "two"])`, formatDecorator("slices", []any{1, "two"}))
	assert.Equal(t, `@relation(onDelete: "cascade")`, formatDecorator("relation", map[string]any{"onDelete": "cascade"}))
}

func TestFormatAnySlice(t *testing.T) {
	assert.Equal(t, "[]", formatAnySlice(nil))
	assert.Equal(t, `[1, "two", true]`, formatAnySlice([]any{1, "two", true}))
}

func TestFormatNamedArgs(t *testing.T) {
	result := formatNamedArgs(map[string]any{"b": 2, "a": "x"})
	assert.Equal(t, `a: "x", b: 2`, result)
}

func TestFormatDecoratorArg_AllBranches(t *testing.T) {
	assert.Equal(t, `"hello"`, formatDecoratorArg("hello"))
	assert.Equal(t, "42", formatDecoratorArg(42))
	assert.Equal(t, `["a", "b"]`, formatDecoratorArg([]string{"a", "b"}))
	assert.Equal(t, `[1, 2]`, formatDecoratorArg([]any{1, 2}))
	assert.Equal(t, `{k: "v"}`, formatDecoratorArg(map[string]any{"k": "v"}))
}

func TestGetEnumHover(t *testing.T) {
	doc := newDoc("enum Status {\n  draft\n  published\n}")
	hover := GetHover(doc, Position{Line: 0, Character: 6})
	require.NotNil(t, hover)
	assert.Contains(t, hover.Contents.Value, "Status (enum)")
	assert.Contains(t, hover.Contents.Value, "draft")
	assert.Contains(t, hover.Contents.Value, "published")
}

func TestGetHover_TypeDescription(t *testing.T) {
	doc := newDoc(blogSchema)
	hover := GetHover(doc, Position{Line: 3, Character: 6})
	require.NotNil(t, hover)
	assert.Contains(t, hover.Contents.Value, "Article")
}

func TestGetHover_EnumKeyword(t *testing.T) {
	doc := newDoc("enum Status {\n  draft\n}")
	hover := GetHover(doc, Position{Line: 0, Character: 1})
	require.NotNil(t, hover)
	assert.Contains(t, hover.Contents.Value, "## enum")
}
