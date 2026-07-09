package template

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseContent_FSLFrontmatter(t *testing.T) {
	content := `---
name: Blog Post
category: content
---
type BlogPost {
  title: String!
}`
	tf, err := ParseContent(content, "fsl")
	require.NoError(t, err)
	assert.Equal(t, "Blog Post", tf.Name)
	assert.Contains(t, tf.FSL, "BlogPost")
}

func TestParseContent_FSLAutoDetect(t *testing.T) {
	content := `---
name: Blog Post
category: content
---
type BlogPost {
  title: String!
}`
	tf, err := ParseContent(content, "")
	require.NoError(t, err)
	assert.Equal(t, "Blog Post", tf.Name)
}

func TestParseContent_FSLMissingFrontmatterErrors(t *testing.T) {
	_, err := ParseContent("type BlogPost { title: String! }", "fsl")
	assert.ErrorContains(t, err, "YAML frontmatter")
}

func TestParseContent_FSLInvalidFrontmatterErrors(t *testing.T) {
	content := "---\nname: [unterminated\n---\ntype X { a: String! }"
	_, err := ParseContent(content, "fsl")
	assert.ErrorContains(t, err, "invalid YAML frontmatter")
}

func TestParseContent_UnsupportedFormatErrors(t *testing.T) {
	_, err := ParseContent("anything", "xml")
	assert.ErrorContains(t, err, "unsupported format")
}

func TestParseContent_InvalidYAMLErrors(t *testing.T) {
	_, err := ParseContent("not: [valid: yaml", "yaml")
	assert.ErrorContains(t, err, "invalid YAML")
}

func TestParseContent_InvalidFSLBodyFailsValidation(t *testing.T) {
	content := "---\nname: Broken\ncategory: content\n---\ntype Broken { title: "
	_, err := ParseContent(content, "fsl")
	assert.ErrorContains(t, err, "FSL validation failed")
}

func TestParseFile_DetectsFormatByExtension(t *testing.T) {
	dir := t.TempDir()

	t.Run("yaml extension", func(t *testing.T) {
		p := filepath.Join(dir, "t.yaml")
		require.NoError(t, os.WriteFile(p, []byte("name: Y\ncategory: content\nfsl: |\n  type Y { x: String! }\n"), 0o644))
		tf, err := ParseFile(p)
		require.NoError(t, err)
		assert.Equal(t, "Y", tf.Name)
	})

	t.Run("yml extension", func(t *testing.T) {
		p := filepath.Join(dir, "t.yml")
		require.NoError(t, os.WriteFile(p, []byte("name: YML\ncategory: content\nfsl: |\n  type YML { x: String! }\n"), 0o644))
		tf, err := ParseFile(p)
		require.NoError(t, err)
		assert.Equal(t, "YML", tf.Name)
	})

	t.Run("json extension", func(t *testing.T) {
		p := filepath.Join(dir, "t.json")
		require.NoError(t, os.WriteFile(p, []byte(`{"name":"J","fsl":"type J { x: String! }"}`), 0o644))
		tf, err := ParseFile(p)
		require.NoError(t, err)
		assert.Equal(t, "J", tf.Name)
	})

	t.Run("fsl extension with frontmatter", func(t *testing.T) {
		p := filepath.Join(dir, "t.fsl")
		require.NoError(t, os.WriteFile(p, []byte("---\nname: F\ncategory: content\n---\ntype F { x: String! }\n"), 0o644))
		tf, err := ParseFile(p)
		require.NoError(t, err)
		assert.Equal(t, "F", tf.Name)
	})

	t.Run("unknown extension auto-detects", func(t *testing.T) {
		p := filepath.Join(dir, "t.txt")
		require.NoError(t, os.WriteFile(p, []byte(`{"name":"Auto","fsl":"type Auto { x: String! }"}`), 0o644))
		tf, err := ParseFile(p)
		require.NoError(t, err)
		assert.Equal(t, "Auto", tf.Name)
	})

	t.Run("missing file errors", func(t *testing.T) {
		_, err := ParseFile(filepath.Join(dir, "missing.yaml"))
		assert.ErrorContains(t, err, "failed to read file")
	})
}

func TestValidate_InvalidFSLSyntax(t *testing.T) {
	tf := &TemplateFile{Name: "Bad", FSL: "type Bad { title: "}
	err := Validate(tf)
	assert.ErrorContains(t, err, "FSL validation failed")
}

func TestToFSL(t *testing.T) {
	tf := &TemplateFile{
		Name:        "Blog Post",
		Description: "A blog post",
		Category:    "content",
		Icon:        "file-text",
		IsSingleton: true,
		Tags:        []string{"blog", "content"},
		FSL:         "type BlogPost {\n  title: String!\n}",
	}

	out, err := ToFSL(tf)
	require.NoError(t, err)

	assert.True(t, strings.HasPrefix(out, "---\n"))
	assert.Contains(t, out, "name: Blog Post")
	assert.Contains(t, out, "category: content")
	assert.Contains(t, out, "is_singleton: true")
	assert.Contains(t, out, "type BlogPost")

	// The FSL body must not leak into the YAML frontmatter block.
	parts := strings.SplitN(out, "---\n", 3)
	require.Len(t, parts, 3)
	assert.NotContains(t, parts[1], "type BlogPost")
}

func TestToFSL_RoundTrip(t *testing.T) {
	tf := &TemplateFile{
		Name:     "Roundtrip",
		Category: "content",
		FSL:      "type Roundtrip { x: String! }",
	}

	out, err := ToFSL(tf)
	require.NoError(t, err)

	parsed, err := ParseContent(out, "fsl")
	require.NoError(t, err)
	assert.Equal(t, tf.Name, parsed.Name)
	assert.Equal(t, tf.FSL, parsed.FSL)
}

func TestWriteFile_FormatByExtension(t *testing.T) {
	dir := t.TempDir()
	tf := &TemplateFile{Name: "WF", Category: "content", FSL: "type WF { x: String! }"}

	t.Run("json extension", func(t *testing.T) {
		p := filepath.Join(dir, "out.json")
		require.NoError(t, WriteFile(tf, p, ""))
		content, err := os.ReadFile(p)
		require.NoError(t, err)
		assert.Contains(t, string(content), `"name": "WF"`)
	})

	t.Run("fsl extension", func(t *testing.T) {
		p := filepath.Join(dir, "out.fsl")
		require.NoError(t, WriteFile(tf, p, ""))
		content, err := os.ReadFile(p)
		require.NoError(t, err)
		assert.True(t, strings.HasPrefix(string(content), "---\n"))
	})

	t.Run("unrecognized extension defaults to yaml", func(t *testing.T) {
		p := filepath.Join(dir, "out.unknown")
		require.NoError(t, WriteFile(tf, p, ""))
		content, err := os.ReadFile(p)
		require.NoError(t, err)
		assert.Contains(t, string(content), "name: WF")
	})

	t.Run("explicit format overrides extension", func(t *testing.T) {
		p := filepath.Join(dir, "out.yaml")
		require.NoError(t, WriteFile(tf, p, "json"))
		content, err := os.ReadFile(p)
		require.NoError(t, err)
		assert.Contains(t, string(content), `"name": "WF"`)
	})

	t.Run("unwritable path errors", func(t *testing.T) {
		err := WriteFile(tf, filepath.Join(dir, "no-such-dir", "out.yaml"), "yaml")
		assert.Error(t, err)
	})
}
