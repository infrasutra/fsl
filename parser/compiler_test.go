package parser

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCompileMultiple_MultipleTypes(t *testing.T) {
	schema := mustParse(t, `
type Post {
  title: String!
  body: RichText
}

type Author {
  name: String!
  email: String! @unique
}
`)

	compiled, err := CompileMultiple(schema, "blog", false)
	require.NoError(t, err)
	assert.Len(t, compiled, 2)

	assert.Equal(t, "Post", compiled[0].Name)
	assert.Equal(t, "blog_0", compiled[0].ApiID)
	assert.Len(t, compiled[0].Fields, 2)
	assert.Equal(t, "title", compiled[0].Fields[0].Name)
	assert.Equal(t, "String", compiled[0].Fields[0].Type)
	assert.True(t, compiled[0].Fields[0].Required)
	assert.Equal(t, "body", compiled[0].Fields[1].Name)
	assert.Equal(t, "RichText", compiled[0].Fields[1].Type)
	assert.False(t, compiled[0].Fields[1].Required)

	assert.Equal(t, "Author", compiled[1].Name)
	assert.Equal(t, "blog_1", compiled[1].ApiID)
	assert.Len(t, compiled[1].Fields, 2)
}

func TestCompileMultiple_EmptyBaseApiID(t *testing.T) {
	schema := mustParse(t, `
type Post {
  title: String!
}
`)

	compiled, err := CompileMultiple(schema, "", false)
	require.NoError(t, err)
	require.Len(t, compiled, 1)
	assert.Equal(t, "type_0", compiled[0].ApiID)
}

func TestCompileMultiple_SingleType(t *testing.T) {
	schema := mustParse(t, `
type Settings {
  siteName: String!
  logo: Image
}
`)

	compiled, err := CompileMultiple(schema, "settings", true)
	require.NoError(t, err)
	require.Len(t, compiled, 1)
	assert.Equal(t, "Settings", compiled[0].Name)
	assert.True(t, compiled[0].Singleton)
}

func TestCompileMultiple_EmptySchema(t *testing.T) {
	schema := &Schema{Types: []TypeDef{}}
	_, err := CompileMultiple(schema, "test", false)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "at least one type")
}

func TestCompileMultiple_Checksums(t *testing.T) {
	schema := mustParse(t, `
type Post {
  title: String!
}

type Tag {
  name: String!
}
`)

	compiled, err := CompileMultiple(schema, "app", false)
	require.NoError(t, err)
	require.Len(t, compiled, 2)

	assert.NotEmpty(t, compiled[0].Checksum)
	assert.NotEmpty(t, compiled[1].Checksum)
	assert.NotEqual(t, compiled[0].Checksum, compiled[1].Checksum, "different types should have different checksums")
}

func TestCompile_FieldTypes(t *testing.T) {
	schema := mustParse(t, `
type Product {
  name: String! @maxLength(100)
  price: Float!
  quantity: Int @min(0)
  available: Boolean @default(true)
  releaseDate: DateTime
  tags: [String]
}
`)

	compiled, err := Compile(schema, "Product", "product", false)
	require.NoError(t, err)
	assert.Equal(t, "Product", compiled.Name)
	assert.Equal(t, "product", compiled.ApiID)
	assert.Len(t, compiled.Fields, 6)

	nameField := compiled.Fields[0]
	assert.Equal(t, "String", nameField.Type)
	assert.True(t, nameField.Required)

	priceField := compiled.Fields[1]
	assert.Equal(t, "Float", priceField.Type)
	assert.True(t, priceField.Required)

	tagsField := compiled.Fields[5]
	assert.Equal(t, "String", tagsField.Type)
	assert.True(t, tagsField.Array)
}

func TestCompile_Decorators(t *testing.T) {
	schema := mustParse(t, `
type Post {
  title: String! @maxLength(200)
  slug: String! @unique @pattern("^[a-z0-9-]+$")
}
`)

	compiled, err := Compile(schema, "Post", "post", false)
	require.NoError(t, err)

	titleField := compiled.Fields[0]
	assert.NotNil(t, titleField.Decorators)
	maxLen, ok := titleField.Decorators["maxLength"]
	assert.True(t, ok)
	assert.Equal(t, int64(200), maxLen)

	slugField := compiled.Fields[1]
	_, hasUnique := slugField.Decorators["unique"]
	assert.True(t, hasUnique)
}

// mustParse is a test helper that parses FSL and returns the schema, failing the test on error.
func mustParse(t *testing.T, input string) *Schema {
	t.Helper()
	result := ParseWithDiagnostics(input)
	require.True(t, result.Valid, "parse failed: %v", result.Diagnostics)
	require.NotNil(t, result.Schema)
	return result.Schema
}
