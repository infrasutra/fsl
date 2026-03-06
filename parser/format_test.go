package parser

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFormat_CanonicalizesSchema(t *testing.T) {
	input := `@description("Blog")
@icon("newspaper")
type Article{
title:String! @maxLength(200) @minLength(1)
slug:[String!]!@maxItems(10)@minItems(1)
slices:JSON! @slices(cta: CtaSlice,hero: HeroSlice)
status:"draft"|"published"!
}

enum Status{draft,published,archived}`

	formatted, err := Format(input)
	require.NoError(t, err)

	expected := `@icon("newspaper")
@description("Blog")
type Article {
  title: String! @minLength(1) @maxLength(200)
  slug: [String!]! @minItems(1) @maxItems(10)
  slices: JSON! @slices(cta: "CtaSlice", hero: "HeroSlice")
  status: "draft" | "published"!
}

enum Status {
  draft,
  published,
  archived
}
`

	assert.Equal(t, expected, formatted)
}

func TestFormat_InvalidSchemaReturnsError(t *testing.T) {
	_, err := Format(`type Post { title:`)
	assert.Error(t, err)
}

func TestFormatSchemaNil(t *testing.T) {
	assert.Equal(t, "", FormatSchema(nil))
}
