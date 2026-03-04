package template

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseContent_YAML(t *testing.T) {
	yaml := `name: Blog Post
description: A blog post template
category: content
icon: file-text
fsl: |
  type BlogPost {
    title: String!
    body: RichText
  }
`
	tf, err := ParseContent(yaml, "yaml")
	require.NoError(t, err)
	assert.Equal(t, "Blog Post", tf.Name)
	assert.Equal(t, "content", tf.Category)
	assert.Contains(t, tf.FSL, "BlogPost")
}

func TestParseContent_JSON(t *testing.T) {
	jsonStr := `{
  "name": "Product",
  "category": "commerce",
  "fsl": "type Product {\n  name: String!\n  price: Float!\n}"
}`
	tf, err := ParseContent(jsonStr, "json")
	require.NoError(t, err)
	assert.Equal(t, "Product", tf.Name)
	assert.Equal(t, "commerce", tf.Category)
}

func TestParseContent_AutoDetect(t *testing.T) {
	t.Run("detects JSON", func(t *testing.T) {
		tf, err := ParseContent(`{"name":"Test","fsl":"type T { x: String! }"}`, "")
		require.NoError(t, err)
		assert.Equal(t, "Test", tf.Name)
	})

	t.Run("detects YAML", func(t *testing.T) {
		tf, err := ParseContent("name: Test\nfsl: \"type T { x: String! }\"", "")
		require.NoError(t, err)
		assert.Equal(t, "Test", tf.Name)
	})
}

func TestValidate(t *testing.T) {
	t.Run("valid template", func(t *testing.T) {
		tf := &TemplateFile{Name: "Test", FSL: "type Test { x: String! }"}
		err := Validate(tf)
		assert.NoError(t, err)
	})

	t.Run("missing name", func(t *testing.T) {
		tf := &TemplateFile{FSL: "type X { a: String! }"}
		err := Validate(tf)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "name")
	})

	t.Run("missing FSL", func(t *testing.T) {
		tf := &TemplateFile{Name: "Test"}
		err := Validate(tf)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "FSL")
	})

	t.Run("invalid category", func(t *testing.T) {
		tf := &TemplateFile{Name: "Test", FSL: "type T { x: String! }", Category: "invalid"}
		err := Validate(tf)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "category")
	})

	t.Run("valid categories", func(t *testing.T) {
		for _, cat := range []string{"content", "commerce", "marketing", "system", "custom"} {
			tf := &TemplateFile{Name: "Test", FSL: "type T { x: String! }", Category: cat}
			assert.NoError(t, Validate(tf), "category %q should be valid", cat)
		}
	})
}

func TestIsValidCategory(t *testing.T) {
	assert.True(t, IsValidCategory("content"))
	assert.True(t, IsValidCategory("commerce"))
	assert.True(t, IsValidCategory("marketing"))
	assert.True(t, IsValidCategory("system"))
	assert.True(t, IsValidCategory("custom"))
	assert.False(t, IsValidCategory("invalid"))
	assert.False(t, IsValidCategory(""))
}

func TestGenerateSlug(t *testing.T) {
	assert.Equal(t, "blog-post", GenerateSlug("Blog Post"))
	assert.Equal(t, "hello-world", GenerateSlug("Hello World"))
	assert.Equal(t, "test-123", GenerateSlug("Test 123"))
	assert.Equal(t, "my-template", GenerateSlug("my_template"))
	assert.NotEmpty(t, GenerateSlug("AB")) // short input gets padded
}

func TestRoundTrip_YAML(t *testing.T) {
	tf := &TemplateFile{
		Name:        "Test",
		Description: "A test",
		Category:    "content",
		FSL:         "type Test { x: String! }",
	}

	yaml, err := ToYAML(tf)
	require.NoError(t, err)
	assert.Contains(t, yaml, "name: Test")

	parsed, err := ParseContent(yaml, "yaml")
	require.NoError(t, err)
	assert.Equal(t, tf.Name, parsed.Name)
}

func TestRoundTrip_JSON(t *testing.T) {
	tf := &TemplateFile{
		Name:     "Product",
		Category: "commerce",
		FSL:      "type Product { name: String! }",
	}

	jsonStr, err := ToJSON(tf)
	require.NoError(t, err)
	assert.Contains(t, jsonStr, `"name": "Product"`)

	parsed, err := ParseContent(jsonStr, "json")
	require.NoError(t, err)
	assert.Equal(t, tf.Name, parsed.Name)
}

func TestWriteFile_AndParseBack(t *testing.T) {
	tf := &TemplateFile{
		Name:     "WriteTest",
		Category: "content",
		FSL:      "type WriteTest { v: String! }",
	}

	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "test.yaml")

	err := WriteFile(tf, path, "yaml")
	require.NoError(t, err)

	// Verify file exists and can be parsed back
	_, err = os.Stat(path)
	require.NoError(t, err)

	parsed, err := ParseFile(path)
	require.NoError(t, err)
	assert.Equal(t, "WriteTest", parsed.Name)
}
