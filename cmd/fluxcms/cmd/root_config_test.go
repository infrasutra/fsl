package cmd

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadConfig(t *testing.T) {
	t.Run("valid yaml populates config", func(t *testing.T) {
		dir := t.TempDir()
		p := filepath.Join(dir, ".fluxcms.yaml")
		require.NoError(t, os.WriteFile(p, []byte("version: \"1\"\nschemas:\n  directory: ./schemas\n"), 0o644))

		withConfig(t, nil)
		loadConfig(p)
		require.NotNil(t, GetConfig())
		assert.Equal(t, "./schemas", GetConfig().Schemas.Directory)
	})

	t.Run("invalid yaml clears config", func(t *testing.T) {
		dir := t.TempDir()
		p := filepath.Join(dir, ".fluxcms.yaml")
		require.NoError(t, os.WriteFile(p, []byte("not: [valid: yaml"), 0o644))

		withConfig(t, &Config{})
		loadConfig(p)
		assert.Nil(t, GetConfig())
	})

	t.Run("missing file is a no-op", func(t *testing.T) {
		prevCfg := &Config{}
		withConfig(t, prevCfg)
		loadConfig(filepath.Join(t.TempDir(), "missing.yaml"))
		assert.Same(t, prevCfg, GetConfig())
	})
}

func TestGetSchemaDirectory(t *testing.T) {
	t.Run("uses config value when set", func(t *testing.T) {
		cfg := &Config{}
		cfg.Schemas.Directory = "./custom"
		withConfig(t, cfg)
		assert.Equal(t, "./custom", GetSchemaDirectory())
	})

	t.Run("falls back to default", func(t *testing.T) {
		withConfig(t, nil)
		assert.Equal(t, "./schemas", GetSchemaDirectory())
	})
}

func TestInitConfig(t *testing.T) {
	t.Run("explicit cfgFile flag is loaded", func(t *testing.T) {
		dir := t.TempDir()
		p := filepath.Join(dir, "custom.yaml")
		require.NoError(t, os.WriteFile(p, []byte("schemas:\n  directory: ./from-flag\n"), 0o644))

		prevCfgFile := cfgFile
		cfgFile = p
		t.Cleanup(func() { cfgFile = prevCfgFile })
		withConfig(t, nil)

		initConfig()
		require.NotNil(t, GetConfig())
		assert.Equal(t, "./from-flag", GetConfig().Schemas.Directory)
	})

	t.Run("searches cwd upward when no flag set", func(t *testing.T) {
		dir := t.TempDir()
		require.NoError(t, os.WriteFile(filepath.Join(dir, ".fluxcms.yaml"), []byte("schemas:\n  directory: ./from-cwd\n"), 0o644))

		nested := filepath.Join(dir, "nested")
		require.NoError(t, os.MkdirAll(nested, 0o755))

		oldWd, err := os.Getwd()
		require.NoError(t, err)
		require.NoError(t, os.Chdir(nested))
		t.Cleanup(func() { _ = os.Chdir(oldWd) })

		prevCfgFile := cfgFile
		cfgFile = ""
		t.Cleanup(func() { cfgFile = prevCfgFile })
		withConfig(t, nil)

		initConfig()
		require.NotNil(t, GetConfig())
		assert.Equal(t, "./from-cwd", GetConfig().Schemas.Directory)
	})
}

func TestExecute(t *testing.T) {
	// Execute() runs the actual root command; --version is side-effect free
	// (prints and exits 0 via cobra rather than os.Exit in tests).
	rootCmd.SetArgs([]string{"--version"})
	t.Cleanup(func() { rootCmd.SetArgs(nil) })

	err := Execute()
	assert.NoError(t, err)
}
