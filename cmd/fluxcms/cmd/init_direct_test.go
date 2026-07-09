package cmd

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRunInit_ExplicitDirectory(t *testing.T) {
	dir := t.TempDir()
	projectDir := filepath.Join(dir, "my-project")

	require.NoError(t, runInit(initCmd, []string{projectDir}))

	assert.FileExists(t, filepath.Join(projectDir, ".fluxcms.yaml"))
	assert.FileExists(t, filepath.Join(projectDir, "README.md"))
	assert.FileExists(t, filepath.Join(projectDir, ".gitignore"))
	assert.DirExists(t, filepath.Join(projectDir, "schemas"))
	assert.FileExists(t, filepath.Join(projectDir, "schemas", "example.fsl"))

	// The generated example schema must itself be valid FSL.
	schemas, err := loadSchemas(filepath.Join(projectDir, "schemas"))
	require.NoError(t, err)
	assert.NotEmpty(t, schemas)
}

func TestRunInit_DefaultsToCurrentDirectory(t *testing.T) {
	dir := t.TempDir()
	oldWd, err := os.Getwd()
	require.NoError(t, err)
	require.NoError(t, os.Chdir(dir))
	t.Cleanup(func() { _ = os.Chdir(oldWd) })

	require.NoError(t, runInit(initCmd, nil))

	assert.FileExists(t, filepath.Join(dir, ".fluxcms.yaml"))
	assert.DirExists(t, filepath.Join(dir, "schemas"))
}
