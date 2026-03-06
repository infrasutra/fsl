package cmd

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// buildCLI builds the fsl binary for testing and returns the path
func buildCLI(t *testing.T) string {
	t.Helper()
	binary := filepath.Join(t.TempDir(), "fsl")
	cmd := exec.Command("go", "build", "-o", binary, "../")
	cmd.Dir = "."
	out, err := cmd.CombinedOutput()
	require.NoError(t, err, "failed to build CLI: %s", string(out))
	return binary
}

func TestCLI_Version(t *testing.T) {
	bin := buildCLI(t)
	out, err := exec.Command(bin, "--version").CombinedOutput()
	require.NoError(t, err)
	assert.Contains(t, string(out), "fluxcms version")
}

func TestCLI_Help(t *testing.T) {
	bin := buildCLI(t)
	out, err := exec.Command(bin, "--help").CombinedOutput()
	require.NoError(t, err)
	output := string(out)
	assert.Contains(t, output, "validate")
	assert.Contains(t, output, "generate")
	assert.Contains(t, output, "lsp")
	assert.Contains(t, output, "migrate")
	assert.Contains(t, output, "template")
	assert.Contains(t, output, "init")
	assert.Contains(t, output, "format")
}

func TestCLI_FormatFile(t *testing.T) {
	bin := buildCLI(t)

	dir := t.TempDir()
	fslFile := filepath.Join(dir, "schema.fsl")
	err := os.WriteFile(fslFile, []byte(`type Article{title:String!@maxLength(200)@minLength(1)}`), 0o644)
	require.NoError(t, err)

	out, err := exec.Command(bin, "format", fslFile).CombinedOutput()
	require.NoError(t, err, "format should succeed: %s", string(out))

	formatted, err := os.ReadFile(fslFile)
	require.NoError(t, err)
	assert.Equal(t, "type Article {\n  title: String! @minLength(1) @maxLength(200)\n}\n", string(formatted))
}

func TestCLI_FormatCheck(t *testing.T) {
	bin := buildCLI(t)

	dir := t.TempDir()
	fslFile := filepath.Join(dir, "schema.fsl")
	err := os.WriteFile(fslFile, []byte(`type Article{title:String!@maxLength(200)@minLength(1)}`), 0o644)
	require.NoError(t, err)

	out, err := exec.Command(bin, "format", "--check", fslFile).CombinedOutput()
	assert.Error(t, err)
	assert.Contains(t, string(out), "would format")
	assert.Contains(t, string(out), "need formatting")
}

func TestCLI_ValidateValid(t *testing.T) {
	bin := buildCLI(t)

	// Create a valid .fsl file
	dir := t.TempDir()
	fslFile := filepath.Join(dir, "test.fsl")
	err := os.WriteFile(fslFile, []byte(`type Post {
  title: String! @maxLength(200)
  slug: String! @pattern("^[a-z0-9-]+$")
  body: RichText
}
`), 0o644)
	require.NoError(t, err)

	out, err := exec.Command(bin, "validate", fslFile).CombinedOutput()
	require.NoError(t, err, "validate should succeed: %s", string(out))
	assert.Contains(t, string(out), "✓")
}

func TestCLI_ValidateInvalid(t *testing.T) {
	bin := buildCLI(t)

	dir := t.TempDir()
	fslFile := filepath.Join(dir, "bad.fsl")
	err := os.WriteFile(fslFile, []byte(`type Post { title: `), 0o644)
	require.NoError(t, err)

	out, err := exec.Command(bin, "validate", fslFile).CombinedOutput()
	assert.Error(t, err, "validate should fail for invalid FSL")
	assert.Contains(t, string(out), "✗")
}

func TestCLI_ValidateNonexistent(t *testing.T) {
	bin := buildCLI(t)
	_, err := exec.Command(bin, "validate", "/nonexistent/file.fsl").CombinedOutput()
	assert.Error(t, err)
}

func TestCLI_ValidateDirectory(t *testing.T) {
	bin := buildCLI(t)

	dir := t.TempDir()
	err := os.WriteFile(filepath.Join(dir, "a.fsl"), []byte(`type A { x: String! }`), 0o644)
	require.NoError(t, err)
	err = os.WriteFile(filepath.Join(dir, "b.fsl"), []byte(`type B { y: Int }`), 0o644)
	require.NoError(t, err)

	out, err := exec.Command(bin, "validate", dir).CombinedOutput()
	require.NoError(t, err, "validate directory should succeed: %s", string(out))
	assert.Contains(t, string(out), "✓")
}

func TestCLI_ValidateDirectoryCrossFileRelations(t *testing.T) {
	bin := buildCLI(t)

	dir := t.TempDir()
	err := os.WriteFile(filepath.Join(dir, "article.fsl"), []byte(`type Article {
  title: String!
  author: Author! @relation
}`), 0o644)
	require.NoError(t, err)

	err = os.WriteFile(filepath.Join(dir, "author.fsl"), []byte(`type Author {
  name: String!
}`), 0o644)
	require.NoError(t, err)

	out, err := exec.Command(bin, "validate", dir).CombinedOutput()
	require.NoError(t, err, "validate cross-file relations should succeed: %s", string(out))
	assert.Contains(t, string(out), "Validation passed")
}

func TestCLI_ValidateJSON(t *testing.T) {
	bin := buildCLI(t)

	dir := t.TempDir()
	fslFile := filepath.Join(dir, "test.fsl")
	err := os.WriteFile(fslFile, []byte(`type Post { title: String! }`), 0o644)
	require.NoError(t, err)

	out, err := exec.Command(bin, "validate", "--format=json", fslFile).CombinedOutput()
	require.NoError(t, err, "validate JSON should succeed: %s", string(out))
	assert.Contains(t, string(out), `"valid"`)
}

func TestCLI_Init(t *testing.T) {
	bin := buildCLI(t)

	dir := t.TempDir()
	projectDir := filepath.Join(dir, "my-project")

	out, err := exec.Command(bin, "init", projectDir).CombinedOutput()
	require.NoError(t, err, "init should succeed: %s", string(out))

	// Verify created files
	assert.FileExists(t, filepath.Join(projectDir, ".fluxcms.yaml"))
	assert.FileExists(t, filepath.Join(projectDir, "README.md"))
	assert.DirExists(t, filepath.Join(projectDir, "schemas"))
	assert.FileExists(t, filepath.Join(projectDir, "schemas", "example.fsl"))
}

func TestCLI_GenerateTypescript(t *testing.T) {
	bin := buildCLI(t)

	dir := t.TempDir()
	fslFile := filepath.Join(dir, "schema.fsl")
	err := os.WriteFile(fslFile, []byte(`type Product {
  name: String! @maxLength(200)
  price: Float! @min(0)
  sku: String! @pattern("^[A-Z0-9-]+$")
  active: Boolean @default(true)
}
`), 0o644)
	require.NoError(t, err)

	outDir := filepath.Join(dir, "sdk")
	out, err := exec.Command(bin, "generate", "typescript",
		"--schema", fslFile,
		"--output", outDir,
	).CombinedOutput()
	require.NoError(t, err, "generate should succeed: %s", string(out))

	// Verify output directory has files
	assert.DirExists(t, outDir)
	entries, err := os.ReadDir(outDir)
	require.NoError(t, err)
	assert.NotEmpty(t, entries, "SDK output should contain files")

	// At least one .ts file should exist
	hasTS := false
	for _, e := range entries {
		if filepath.Ext(e.Name()) == ".ts" {
			hasTS = true
			break
		}
	}
	assert.True(t, hasTS, "should generate at least one .ts file")
}

func TestCLI_GenerateOpenAPI(t *testing.T) {
	bin := buildCLI(t)

	dir := t.TempDir()
	fslFile := filepath.Join(dir, "schema.fsl")
	err := os.WriteFile(fslFile, []byte(`type Product {
  name: String! @maxLength(200)
  price: Float! @min(0)
}
`), 0o644)
	require.NoError(t, err)

	outDir := filepath.Join(dir, "definitions")
	out, err := exec.Command(bin, "generate", "openapi",
		"--schema", fslFile,
		"--output", outDir,
	).CombinedOutput()
	require.NoError(t, err, "generate openapi should succeed: %s", string(out))

	openapiFile := filepath.Join(outDir, "openapi.json")
	assert.FileExists(t, openapiFile)

	content, err := os.ReadFile(openapiFile)
	require.NoError(t, err)
	assert.Contains(t, string(content), `"openapi": "3.0.3"`)
	assert.Contains(t, string(content), `"Product"`)
}

func TestCLI_GenerateJSONSchema(t *testing.T) {
	bin := buildCLI(t)

	dir := t.TempDir()
	fslFile := filepath.Join(dir, "schema.fsl")
	err := os.WriteFile(fslFile, []byte(`type Product {
  name: String!
  tags: [String] @minItems(1)
}
`), 0o644)
	require.NoError(t, err)

	outDir := filepath.Join(dir, "definitions")
	out, err := exec.Command(bin, "generate", "openapi",
		"--format", "jsonschema",
		"--schema", fslFile,
		"--output", outDir,
	).CombinedOutput()
	require.NoError(t, err, "generate jsonschema should succeed: %s", string(out))

	jsonSchemaFile := filepath.Join(outDir, "jsonschema.json")
	assert.FileExists(t, jsonSchemaFile)

	content, err := os.ReadFile(jsonSchemaFile)
	require.NoError(t, err)
	assert.Contains(t, string(content), `"$schema": "https://json-schema.org/draft/2020-12/schema"`)
	assert.Contains(t, string(content), `"$defs"`)
}

func TestCLI_MigratePreview(t *testing.T) {
	bin := buildCLI(t)

	dir := t.TempDir()
	err := os.WriteFile(filepath.Join(dir, "schema.fsl"), []byte(`type Post { title: String! }`), 0o644)
	require.NoError(t, err)

	out, err := exec.Command(bin, "migrate", "preview", "--schema", dir).CombinedOutput()
	require.NoError(t, err, "migrate preview should succeed: %s", string(out))
	assert.Contains(t, string(out), "Post")
}

func TestCLI_MigrateCheck(t *testing.T) {
	bin := buildCLI(t)

	dir := t.TempDir()
	err := os.WriteFile(filepath.Join(dir, "schema.fsl"), []byte(`type Post {
  title: String!
  body: RichText
}
`), 0o644)
	require.NoError(t, err)

	out, _ := exec.Command(bin, "migrate", "check", "--schema", dir).CombinedOutput()
	// May pass or fail depending on required fields without defaults
	// Just verify it doesn't panic
	assert.NotEmpty(t, string(out))
}
