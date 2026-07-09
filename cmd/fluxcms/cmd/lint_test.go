package cmd

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/infrasutra/fsl/parser"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRunLint_CleanFile(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "post.fsl"), []byte(`
@description("Blog posts")
type Post {
  title: String! @maxLength(200)
}
`), 0o644))

	require.NoError(t, runLint(lintCmd, []string{dir}))
}

func TestRunLint_ParseErrorsFail(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "bad.fsl"), []byte(`type Post { title: `), 0o644))

	err := runLint(lintCmd, []string{dir})
	assert.ErrorContains(t, err, "parse errors")
}

func TestRunLint_FindingsDoNotError(t *testing.T) {
	dir := t.TempDir()
	// A type name that isn't PascalCase should trip the naming-convention lint rule
	// without producing a hard parse error.
	require.NoError(t, os.WriteFile(filepath.Join(dir, "post.fsl"), []byte(`type lowercase_type { x: String! }`), 0o644))

	require.NoError(t, runLint(lintCmd, []string{dir}))
}

func TestRunLint_DefaultsToSchemaDirectory(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "post.fsl"), []byte(`type Post { title: String! }`), 0o644))

	cfg := &Config{}
	cfg.Schemas.Directory = dir
	withConfig(t, cfg)

	require.NoError(t, runLint(lintCmd, nil))
}

func TestLintSeverityFormat(t *testing.T) {
	_, warnLabel := lintSeverityFormat(parser.LintWarning)
	assert.Equal(t, "warning", warnLabel)

	_, hintLabel := lintSeverityFormat(parser.LintHint)
	assert.Equal(t, "hint   ", hintLabel)

	_, infoLabel := lintSeverityFormat(parser.LintSeverity(99))
	assert.Equal(t, "info   ", infoLabel)
}

func TestBuildLintConfig(t *testing.T) {
	t.Run("nil config returns defaults", func(t *testing.T) {
		withConfig(t, nil)
		cfg := buildLintConfig()
		assert.Equal(t, parser.DefaultLinterConfig(), cfg)
	})

	t.Run("config overrides applied", func(t *testing.T) {
		falseVal := false
		c := &Config{}
		c.Lint.NamingConvention = &falseVal
		c.Lint.UnusedTypes = &falseVal
		c.Lint.RequiredFieldOrdering = &falseVal
		c.Lint.RelationCardinality = &falseVal
		c.Lint.MaxFieldCount = 10
		withConfig(t, c)

		cfg := buildLintConfig()
		assert.False(t, cfg.NamingConvention)
		assert.False(t, cfg.UnusedTypes)
		assert.False(t, cfg.RequiredFieldOrdering)
		assert.False(t, cfg.RelationCardinality)
		assert.Equal(t, 10, cfg.MaxFieldCount)
	})
}
