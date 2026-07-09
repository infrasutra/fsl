package cmd

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCollectFSLFiles(t *testing.T) {
	t.Run("nonexistent path errors", func(t *testing.T) {
		_, err := collectFSLFiles([]string{filepath.Join(t.TempDir(), "missing")})
		assert.Error(t, err)
	})

	t.Run("directory with no fsl files errors", func(t *testing.T) {
		dir := t.TempDir()
		require.NoError(t, os.WriteFile(filepath.Join(dir, "readme.md"), []byte("x"), 0o644))

		_, err := collectFSLFiles([]string{dir})
		assert.ErrorContains(t, err, "no .fsl files found")
	})

	t.Run("directory walk collects nested fsl files", func(t *testing.T) {
		dir := t.TempDir()
		nested := filepath.Join(dir, "nested")
		require.NoError(t, os.MkdirAll(nested, 0o755))
		require.NoError(t, os.WriteFile(filepath.Join(dir, "a.fsl"), []byte("type A { x: String! }"), 0o644))
		require.NoError(t, os.WriteFile(filepath.Join(nested, "b.fsl"), []byte("type B { y: String! }"), 0o644))
		require.NoError(t, os.WriteFile(filepath.Join(dir, "ignored.txt"), []byte("x"), 0o644))

		files, err := collectFSLFiles([]string{dir})
		require.NoError(t, err)
		assert.Len(t, files, 2)
	})

	t.Run("explicit file path is included regardless of extension", func(t *testing.T) {
		dir := t.TempDir()
		p := filepath.Join(dir, "schema.txt")
		require.NoError(t, os.WriteFile(p, []byte("type A { x: String! }"), 0o644))

		files, err := collectFSLFiles([]string{p})
		require.NoError(t, err)
		assert.Equal(t, []string{p}, files)
	})

	t.Run("multiple paths combine", func(t *testing.T) {
		dir := t.TempDir()
		p1 := filepath.Join(dir, "a.fsl")
		p2 := filepath.Join(dir, "b.fsl")
		require.NoError(t, os.WriteFile(p1, []byte("type A { x: String! }"), 0o644))
		require.NoError(t, os.WriteFile(p2, []byte("type B { y: String! }"), 0o644))

		files, err := collectFSLFiles([]string{p1, p2})
		require.NoError(t, err)
		assert.Len(t, files, 2)
	})
}

func TestLoadSchemas(t *testing.T) {
	t.Run("valid schema parses", func(t *testing.T) {
		dir := t.TempDir()
		require.NoError(t, os.WriteFile(filepath.Join(dir, "post.fsl"), []byte(`type Post { title: String! }`), 0o644))

		schemas, err := loadSchemas(dir)
		require.NoError(t, err)
		require.Len(t, schemas, 1)
	})

	t.Run("invalid schema errors with message", func(t *testing.T) {
		dir := t.TempDir()
		require.NoError(t, os.WriteFile(filepath.Join(dir, "bad.fsl"), []byte(`type Post { title: `), 0o644))

		_, err := loadSchemas(dir)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "schema validation failed")
	})

	t.Run("cross-file relations resolve", func(t *testing.T) {
		dir := t.TempDir()
		require.NoError(t, os.WriteFile(filepath.Join(dir, "article.fsl"), []byte(`type Article { author: Author! @relation }`), 0o644))
		require.NoError(t, os.WriteFile(filepath.Join(dir, "author.fsl"), []byte(`type Author { name: String! }`), 0o644))

		schemas, err := loadSchemas(dir)
		require.NoError(t, err)
		assert.Len(t, schemas, 2)
	})

	t.Run("no files found errors", func(t *testing.T) {
		_, err := loadSchemas(filepath.Join(t.TempDir(), "missing"))
		assert.Error(t, err)
	})
}
