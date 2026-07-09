package lsp

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIsAtTypeLevel(t *testing.T) {
	t.Run("top level before any braces", func(t *testing.T) {
		doc := newDoc("@")
		assert.True(t, isAtTypeLevel(doc, Position{Line: 0, Character: 1}))
	})

	t.Run("inside a type body", func(t *testing.T) {
		doc := newDoc("type Post {\n  @\n}")
		assert.False(t, isAtTypeLevel(doc, Position{Line: 1, Character: 3}))
	})

	t.Run("after closing brace returns to type level", func(t *testing.T) {
		doc := newDoc("type Post {\n  title: String!\n}\n@")
		assert.True(t, isAtTypeLevel(doc, Position{Line: 3, Character: 1}))
	})
}

func TestGetCustomTypeCompletions(t *testing.T) {
	t.Run("nil schema returns nil", func(t *testing.T) {
		doc := newDoc("type Post { title: ")
		assert.Nil(t, getCustomTypeCompletions(doc))
	})

	t.Run("returns types and enums", func(t *testing.T) {
		doc := newDoc(blogSchema)
		items := getCustomTypeCompletions(doc)
		require.NotEmpty(t, items)

		labels := map[string]bool{}
		for _, item := range items {
			labels[item.Label] = true
		}
		assert.True(t, labels["Article"])
		assert.True(t, labels["Category"])
		assert.True(t, labels["Status"])
	})
}

func TestIsInsideRelationDecorator_MoreCases(t *testing.T) {
	t.Run("cursor beyond line length", func(t *testing.T) {
		assert.False(t, isInsideRelationDecorator("short", Position{Line: 0, Character: 100}))
	})

	t.Run("closed parens before cursor is outside", func(t *testing.T) {
		line := `author: Author @relation(inverse: "posts")`
		assert.False(t, isInsideRelationDecorator(line, Position{Line: 0, Character: len(line)}))
	})

	t.Run("cursor inside open relation args", func(t *testing.T) {
		line := `author: Author @relation(inverse: "posts"`
		assert.True(t, isInsideRelationDecorator(line, Position{Line: 0, Character: len(line)}))
	})

	t.Run("no relation decorator on line", func(t *testing.T) {
		assert.False(t, isInsideRelationDecorator("title: String!", Position{Line: 0, Character: 5}))
	})
}

func TestGetCompletions_ContextBranches(t *testing.T) {
	t.Run("empty document suggests keywords", func(t *testing.T) {
		doc := newDoc("")
		completions := GetCompletions(doc, Position{Line: 0, Character: 0})
		require.NotEmpty(t, completions.Items)
	})

	t.Run("inside relation decorator suggests relation args", func(t *testing.T) {
		doc := newDoc("type Post {\n  author: Author @relation(\n}")
		line := "  author: Author @relation("
		completions := GetCompletions(doc, Position{Line: 1, Character: len(line)})
		require.NotEmpty(t, completions.Items)
		assert.Equal(t, "inverse", completions.Items[0].Label)
	})

	t.Run("inside type body suggests field decorators", func(t *testing.T) {
		doc := newDoc("type Post {\n  title: String! {\n}")
		completions := GetCompletions(doc, Position{Line: 1, Character: 18})
		require.NotEmpty(t, completions.Items)
	})

	t.Run("default branch provides broad completions", func(t *testing.T) {
		doc := newDoc("type Post {\n  titl\n}")
		completions := GetCompletions(doc, Position{Line: 1, Character: 6})
		require.NotEmpty(t, completions.Items)
	})
}
