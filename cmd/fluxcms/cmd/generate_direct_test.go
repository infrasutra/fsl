package cmd

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDeriveApiID(t *testing.T) {
	cases := map[string]string{
		"":            "",
		"Post":        "post",
		"BlogPost":    "blog_post",
		"HTTPServer":  "http_server",
		"IOWriter":    "io_writer",
		"already_ok":  "already_ok",
		"ABTest":      "ab_test",
		"SimpleXYZAB": "simple_xyzab",
	}
	for input, want := range cases {
		assert.Equal(t, want, deriveApiID(input), "input: %s", input)
	}
}

func TestResolvedSchemaPath(t *testing.T) {
	t.Run("flag takes precedence", func(t *testing.T) {
		prev := generateSchemaPath
		generateSchemaPath = "./flag-schemas"
		t.Cleanup(func() { generateSchemaPath = prev })
		withConfig(t, nil)

		assert.Equal(t, "./flag-schemas", resolvedSchemaPath())
	})

	t.Run("config used when flag empty", func(t *testing.T) {
		prev := generateSchemaPath
		generateSchemaPath = ""
		t.Cleanup(func() { generateSchemaPath = prev })

		cfg := &Config{}
		cfg.Schemas.Directory = "./cfg-schemas"
		withConfig(t, cfg)

		assert.Equal(t, "./cfg-schemas", resolvedSchemaPath())
	})

	t.Run("falls back to default", func(t *testing.T) {
		prev := generateSchemaPath
		generateSchemaPath = ""
		t.Cleanup(func() { generateSchemaPath = prev })
		withConfig(t, nil)

		assert.Equal(t, "./schemas", resolvedSchemaPath())
	})
}

func TestLoadCompiledSchemas(t *testing.T) {
	t.Run("compiles all types with derived api ids", func(t *testing.T) {
		dir := t.TempDir()
		require.NoError(t, os.WriteFile(filepath.Join(dir, "schema.fsl"), []byte(`
type BlogPost { title: String! }
type Author { name: String! }
`), 0o644))

		schemas, err := loadCompiledSchemas(dir)
		require.NoError(t, err)
		require.Len(t, schemas, 2)

		apiIDs := map[string]bool{}
		for _, s := range schemas {
			apiIDs[s.ApiID] = true
		}
		assert.True(t, apiIDs["blog_post"])
		assert.True(t, apiIDs["author"])
	})

	t.Run("invalid schema errors", func(t *testing.T) {
		dir := t.TempDir()
		require.NoError(t, os.WriteFile(filepath.Join(dir, "bad.fsl"), []byte(`type Post { title: `), 0o644))

		_, err := loadCompiledSchemas(dir)
		assert.Error(t, err)
	})
}

func resetGenerateFlags(t *testing.T) {
	t.Helper()
	prevSchema, prevOutput, prevClient, prevTarget, prevWSAPIID, prevExportFmt :=
		generateSchemaPath, generateOutputPath, generateClient, generateTarget, generateWorkspaceAPIID, generateExportFormat
	generateClient = "fetch"
	generateTarget = "content"
	t.Cleanup(func() {
		generateSchemaPath, generateOutputPath, generateClient, generateTarget, generateWorkspaceAPIID, generateExportFormat =
			prevSchema, prevOutput, prevClient, prevTarget, prevWSAPIID, prevExportFmt
	})
}

func TestRunGenerateTypescript(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "post.fsl"), []byte(`type Post { title: String! }`), 0o644))

	t.Run("valid client generates files", func(t *testing.T) {
		resetGenerateFlags(t)
		withConfig(t, nil)
		generateSchemaPath = dir
		generateOutputPath = t.TempDir()
		generateClient = "fetch"

		require.NoError(t, runGenerateTypescript(generateTypescriptCmd, nil))

		entries, err := os.ReadDir(generateOutputPath)
		require.NoError(t, err)
		assert.NotEmpty(t, entries)
	})

	t.Run("invalid client errors", func(t *testing.T) {
		resetGenerateFlags(t)
		withConfig(t, nil)
		generateSchemaPath = dir
		generateOutputPath = t.TempDir()
		generateClient = "carrier-pigeon"

		err := runGenerateTypescript(generateTypescriptCmd, nil)
		assert.ErrorContains(t, err, "unsupported client")
	})
}

func TestRunGeneratePython(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "post.fsl"), []byte(`type Post { title: String! }`), 0o644))

	resetGenerateFlags(t)
	withConfig(t, nil)
	generateSchemaPath = dir
	generateOutputPath = t.TempDir()

	require.NoError(t, runGeneratePython(generatePythonCmd, nil))
	entries, err := os.ReadDir(generateOutputPath)
	require.NoError(t, err)
	assert.NotEmpty(t, entries)
}

func TestRunGenerateGo(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "post.fsl"), []byte(`type Post { title: String! }`), 0o644))

	resetGenerateFlags(t)
	withConfig(t, nil)
	generateSchemaPath = dir
	generateOutputPath = filepath.Join(t.TempDir(), "myclient")

	require.NoError(t, runGenerateGo(generateGoCmd, nil))

	content, err := os.ReadFile(filepath.Join(generateOutputPath, "client.go"))
	require.NoError(t, err)
	assert.Contains(t, string(content), "package myclient")
}

func TestRunGenerateOpenAPI(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "post.fsl"), []byte(`type Post { title: String! }`), 0o644))

	t.Run("openapi format", func(t *testing.T) {
		resetGenerateFlags(t)
		withConfig(t, nil)
		generateSchemaPath = dir
		generateOutputPath = t.TempDir()
		generateExportFormat = "openapi"

		require.NoError(t, runGenerateOpenAPI(generateOpenAPICmd, nil))
		assert.FileExists(t, filepath.Join(generateOutputPath, "openapi.json"))
	})

	t.Run("jsonschema format", func(t *testing.T) {
		resetGenerateFlags(t)
		withConfig(t, nil)
		generateSchemaPath = dir
		generateOutputPath = t.TempDir()
		generateExportFormat = "jsonschema"

		require.NoError(t, runGenerateOpenAPI(generateOpenAPICmd, nil))
		assert.FileExists(t, filepath.Join(generateOutputPath, "jsonschema.json"))
	})
}

func TestRunGenerate_TargetValidation(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "post.fsl"), []byte(`type Post { title: String! }`), 0o644))

	t.Run("cms target rejected", func(t *testing.T) {
		resetGenerateFlags(t)
		withConfig(t, nil)
		generateSchemaPath = dir
		generateOutputPath = t.TempDir()
		generateTarget = "cms"

		err := runGenerateTypescript(generateTypescriptCmd, nil)
		assert.ErrorContains(t, err, "CMS SDK generation")
	})

	t.Run("unknown target rejected", func(t *testing.T) {
		resetGenerateFlags(t)
		withConfig(t, nil)
		generateSchemaPath = dir
		generateOutputPath = t.TempDir()
		generateTarget = "bogus"

		err := runGenerateTypescript(generateTypescriptCmd, nil)
		assert.ErrorContains(t, err, "invalid target")
	})
}
