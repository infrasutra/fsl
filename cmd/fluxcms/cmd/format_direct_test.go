package cmd

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func withFormatCheck(t *testing.T, check bool) {
	t.Helper()
	prev := formatCheck
	formatCheck = check
	t.Cleanup(func() { formatCheck = prev })
}

func TestRunFormat_WritesChanges(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "post.fsl")
	require.NoError(t, os.WriteFile(p, []byte(`type Post{title:String!}`), 0o644))

	withFormatCheck(t, false)
	require.NoError(t, runFormat(formatCmd, []string{p}))

	content, err := os.ReadFile(p)
	require.NoError(t, err)
	assert.Equal(t, "type Post {\n  title: String!\n}\n", string(content))
}

func TestRunFormat_AlreadyFormattedNoop(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "post.fsl")
	formatted := "type Post {\n  title: String!\n}\n"
	require.NoError(t, os.WriteFile(p, []byte(formatted), 0o644))

	withFormatCheck(t, false)
	require.NoError(t, runFormat(formatCmd, []string{p}))

	content, err := os.ReadFile(p)
	require.NoError(t, err)
	assert.Equal(t, formatted, string(content))
}

func TestRunFormat_CheckModeReportsWithoutWriting(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "post.fsl")
	unformatted := `type Post{title:String!}`
	require.NoError(t, os.WriteFile(p, []byte(unformatted), 0o644))

	withFormatCheck(t, true)
	err := runFormat(formatCmd, []string{p})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "need formatting")

	content, err := os.ReadFile(p)
	require.NoError(t, err)
	assert.Equal(t, unformatted, string(content), "check mode must not modify the file")
}

func TestRunFormat_CheckModeAllFormattedPasses(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "post.fsl")
	require.NoError(t, os.WriteFile(p, []byte("type Post {\n  title: String!\n}\n"), 0o644))

	withFormatCheck(t, true)
	require.NoError(t, runFormat(formatCmd, []string{p}))
}

func TestRunFormat_DefaultsToSchemaDirectory(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "post.fsl"), []byte(`type Post{title:String!}`), 0o644))

	cfg := &Config{}
	cfg.Schemas.Directory = dir
	withConfig(t, cfg)
	withFormatCheck(t, false)

	require.NoError(t, runFormat(formatCmd, nil))
}

func TestRunFormat_InvalidSchemaErrors(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "bad.fsl")
	require.NoError(t, os.WriteFile(p, []byte(`type Post { title: `), 0o644))

	withFormatCheck(t, false)
	err := runFormat(formatCmd, []string{p})
	assert.Error(t, err)
}
