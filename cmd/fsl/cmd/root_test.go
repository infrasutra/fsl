package cmd

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFindConfigFilePrefersFSLConfig(t *testing.T) {
	root := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(root, ".fsl.yaml"), []byte("version: \"1\"\n"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(root, ".fluxcms.yaml"), []byte("version: \"1\"\n"), 0o644))

	nested := filepath.Join(root, "a", "b")
	require.NoError(t, os.MkdirAll(nested, 0o755))

	path := findConfigFile(nested)
	assert.Equal(t, filepath.Join(root, ".fsl.yaml"), path)
}

func TestFindConfigFileFallsBackToLegacyFluxCMSConfig(t *testing.T) {
	root := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(root, ".fluxcms.yml"), []byte("version: \"1\"\n"), 0o644))

	nested := filepath.Join(root, "x", "y")
	require.NoError(t, os.MkdirAll(nested, 0o755))

	path := findConfigFile(nested)
	assert.Equal(t, filepath.Join(root, ".fluxcms.yml"), path)
}
