package lsp

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestOffsetToPosition(t *testing.T) {
	doc := newDoc("line0\nline1\nline2")

	assert.Equal(t, Position{Line: 0, Character: 0}, doc.OffsetToPosition(0))
	assert.Equal(t, Position{Line: 0, Character: 5}, doc.OffsetToPosition(5))
	assert.Equal(t, Position{Line: 1, Character: 0}, doc.OffsetToPosition(6))
	assert.Equal(t, Position{Line: 2, Character: 2}, doc.OffsetToPosition(14))
	assert.Equal(t, Position{Line: 2, Character: 5}, doc.OffsetToPosition(999), "offset beyond content clamps at end")
}

func TestPositionToOffset(t *testing.T) {
	doc := newDoc("line0\nline1\nline2")

	assert.Equal(t, 0, doc.PositionToOffset(Position{Line: 0, Character: 0}))
	assert.Equal(t, 5, doc.PositionToOffset(Position{Line: 0, Character: 5}))
	assert.Equal(t, 6, doc.PositionToOffset(Position{Line: 1, Character: 0}))
	assert.Equal(t, 14, doc.PositionToOffset(Position{Line: 2, Character: 2}))
}

func TestOffsetPosition_RoundTrip(t *testing.T) {
	doc := newDoc("type Post {\n  title: String!\n  body: RichText\n}")

	for offset := 0; offset < len(doc.Content); offset += 7 {
		pos := doc.OffsetToPosition(offset)
		back := doc.PositionToOffset(pos)
		assert.Equal(t, offset, back, "round trip failed for offset %d", offset)
	}
}

func TestGetWordRange(t *testing.T) {
	doc := newDoc("type Article {\n  title: String!\n}")

	t.Run("word range on type name", func(t *testing.T) {
		r := doc.GetWordRange(Position{Line: 0, Character: 6})
		assert.Equal(t, 5, r.Start.Character)
		assert.Equal(t, 12, r.End.Character)
	})

	t.Run("empty line returns collapsed range", func(t *testing.T) {
		empty := newDoc("")
		r := empty.GetWordRange(Position{Line: 0, Character: 0})
		assert.Equal(t, r.Start, r.End)
	})
}
