package cmd

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func withTemplateFlags(t *testing.T, path, category string) {
	t.Helper()
	prevPath, prevCat := templatePath, templateCategory
	templatePath, templateCategory = path, category
	t.Cleanup(func() { templatePath, templateCategory = prevPath, prevCat })
}

func TestRunTemplateList_MissingDirectory(t *testing.T) {
	withTemplateFlags(t, filepath.Join(t.TempDir(), "does-not-exist"), "")
	require.NoError(t, runTemplateList(templateListCmd, nil))
}

func TestRunTemplateList_NotADirectory(t *testing.T) {
	dir := t.TempDir()
	filePath := filepath.Join(dir, "file.txt")
	require.NoError(t, os.WriteFile(filePath, []byte("x"), 0o644))

	withTemplateFlags(t, filePath, "")
	err := runTemplateList(templateListCmd, nil)
	assert.ErrorContains(t, err, "is not a directory")
}

func TestRunTemplateList_EmptyDirectory(t *testing.T) {
	withTemplateFlags(t, t.TempDir(), "")
	require.NoError(t, runTemplateList(templateListCmd, nil))
}

func TestRunTemplateList_FindsAndFiltersByCategory(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "blog.yaml"), []byte(
		"name: Blog\ncategory: content\nfsl: |\n  type Blog { title: String! }\n"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "product.yaml"), []byte(
		"name: Product\ncategory: commerce\nfsl: |\n  type Product { name: String! }\n"), 0o644))
	// Non-template files should be ignored.
	require.NoError(t, os.WriteFile(filepath.Join(dir, "README.md"), []byte("ignore me"), 0o644))

	t.Run("no filter lists all", func(t *testing.T) {
		withTemplateFlags(t, dir, "")
		require.NoError(t, runTemplateList(templateListCmd, nil))
	})

	t.Run("category filter narrows results", func(t *testing.T) {
		withTemplateFlags(t, dir, "commerce")
		require.NoError(t, runTemplateList(templateListCmd, nil))
	})

	t.Run("category filter with no matches", func(t *testing.T) {
		withTemplateFlags(t, dir, "system")
		require.NoError(t, runTemplateList(templateListCmd, nil))
	})
}

func TestRunTemplateList_WarnsOnInvalidTemplateButContinues(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "broken.json"), []byte(`not valid json`), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "ok.yaml"), []byte(
		"name: OK\ncategory: content\nfsl: |\n  type OK { x: String! }\n"), 0o644))

	withTemplateFlags(t, dir, "")
	require.NoError(t, runTemplateList(templateListCmd, nil))
}

func TestRunTemplateValidate(t *testing.T) {
	dir := t.TempDir()

	t.Run("valid template", func(t *testing.T) {
		p := filepath.Join(dir, "valid.yaml")
		require.NoError(t, os.WriteFile(p, []byte(
			"name: Valid\ncategory: content\nicon: file-text\nis_singleton: true\ntags: [a, b]\nfsl: |\n  type Valid { x: String! }\n"), 0o644))
		require.NoError(t, runTemplateValidate(templateValidateCmd, []string{p}))
	})

	t.Run("invalid template does not return error but prints failure", func(t *testing.T) {
		p := filepath.Join(dir, "invalid.yaml")
		require.NoError(t, os.WriteFile(p, []byte("category: content\nfsl: |\n  type X { a: String! }\n"), 0o644))
		require.NoError(t, runTemplateValidate(templateValidateCmd, []string{p}))
	})

	t.Run("nonexistent file errors via parse", func(t *testing.T) {
		require.NoError(t, runTemplateValidate(templateValidateCmd, []string{filepath.Join(dir, "missing.yaml")}))
	})
}

func TestRunTemplateConvert(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "source.yaml")
	require.NoError(t, os.WriteFile(src, []byte(
		"name: Convertible\ncategory: content\nfsl: |\n  type Convertible { x: String! }\n"), 0o644))

	t.Run("yaml to json", func(t *testing.T) {
		dst := filepath.Join(dir, "out.json")
		require.NoError(t, runTemplateConvert(templateConvertCmd, []string{src, dst}))
		content, err := os.ReadFile(dst)
		require.NoError(t, err)
		assert.Contains(t, string(content), `"name": "Convertible"`)
	})

	t.Run("yaml to fsl", func(t *testing.T) {
		dst := filepath.Join(dir, "out.fsl")
		require.NoError(t, runTemplateConvert(templateConvertCmd, []string{src, dst}))
		content, err := os.ReadFile(dst)
		require.NoError(t, err)
		assert.Contains(t, string(content), "Convertible")
	})

	t.Run("defaults to yaml for unknown extension", func(t *testing.T) {
		dst := filepath.Join(dir, "out.unknown")
		require.NoError(t, runTemplateConvert(templateConvertCmd, []string{src, dst}))
		content, err := os.ReadFile(dst)
		require.NoError(t, err)
		assert.Contains(t, string(content), "name: Convertible")
	})

	t.Run("missing input errors", func(t *testing.T) {
		err := runTemplateConvert(templateConvertCmd, []string{filepath.Join(dir, "missing.yaml"), filepath.Join(dir, "out.json")})
		assert.Error(t, err)
	})
}
