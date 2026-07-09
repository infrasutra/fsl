package cmd

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/infrasutra/fsl/parser"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetSchemaPath(t *testing.T) {
	t.Run("flag takes precedence", func(t *testing.T) {
		prev := migrateSchemaPath
		migrateSchemaPath = "./flag-path"
		t.Cleanup(func() { migrateSchemaPath = prev })
		withConfig(t, nil)

		path, err := getSchemaPath()
		require.NoError(t, err)
		assert.Equal(t, "./flag-path", path)
	})

	t.Run("falls back to config", func(t *testing.T) {
		prev := migrateSchemaPath
		migrateSchemaPath = ""
		t.Cleanup(func() { migrateSchemaPath = prev })

		cfg := &Config{}
		cfg.Schemas.Directory = "./cfg-path"
		withConfig(t, cfg)

		path, err := getSchemaPath()
		require.NoError(t, err)
		assert.Equal(t, "./cfg-path", path)
	})

	t.Run("errors when neither set", func(t *testing.T) {
		prev := migrateSchemaPath
		migrateSchemaPath = ""
		t.Cleanup(func() { migrateSchemaPath = prev })
		withConfig(t, nil)

		_, err := getSchemaPath()
		assert.Error(t, err)
	})
}

func TestLoadPreviousState(t *testing.T) {
	t.Run("missing directory returns nil without error", func(t *testing.T) {
		state, err := loadPreviousState(filepath.Join(t.TempDir(), "does-not-exist"))
		require.NoError(t, err)
		assert.Nil(t, state)
	})

	t.Run("empty directory returns nil", func(t *testing.T) {
		dir := t.TempDir()
		state, err := loadPreviousState(dir)
		require.NoError(t, err)
		assert.Nil(t, state)
	})

	t.Run("reads latest migration by filename", func(t *testing.T) {
		dir := t.TempDir()
		older := map[string]any{
			"schemaState": map[string]*parser.CompiledSchema{
				"Post": {Name: "Post"},
			},
		}
		newer := map[string]any{
			"schemaState": map[string]*parser.CompiledSchema{
				"Post":   {Name: "Post"},
				"Author": {Name: "Author"},
			},
		}
		writeJSON(t, filepath.Join(dir, "20240101000000_first.json"), older)
		writeJSON(t, filepath.Join(dir, "20240202000000_second.json"), newer)

		state, err := loadPreviousState(dir)
		require.NoError(t, err)
		require.Len(t, state, 2)
		assert.Contains(t, state, "Author")
	})

	t.Run("migration without schema state warns but does not error", func(t *testing.T) {
		dir := t.TempDir()
		writeJSON(t, filepath.Join(dir, "20240101000000_baseline.json"), map[string]any{
			"name": "baseline",
		})

		state, err := loadPreviousState(dir)
		require.NoError(t, err)
		assert.Nil(t, state)
	})

	t.Run("invalid json errors", func(t *testing.T) {
		dir := t.TempDir()
		require.NoError(t, os.WriteFile(filepath.Join(dir, "20240101000000_bad.json"), []byte("not json"), 0o644))

		_, err := loadPreviousState(dir)
		assert.Error(t, err)
	})
}

func writeJSON(t *testing.T, path string, v any) {
	t.Helper()
	data, err := json.Marshal(v)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(path, data, 0o644))
}

func TestDiffCurrentVsPrevious(t *testing.T) {
	t.Run("nil previous marks everything added", func(t *testing.T) {
		current := map[string]*parser.CompiledSchema{
			"Post": {Name: "Post"},
		}
		changes := diffCurrentVsPrevious(current, nil)
		require.Len(t, changes, 1)
		assert.Equal(t, parser.ChangeTypeAdded, changes[0].Type)
		assert.False(t, changes[0].Breaking)
	})

	t.Run("detects removed types as breaking", func(t *testing.T) {
		current := map[string]*parser.CompiledSchema{}
		previous := map[string]*parser.CompiledSchema{
			"Post": {Name: "Post"},
		}
		changes := diffCurrentVsPrevious(current, previous)
		require.Len(t, changes, 1)
		assert.Equal(t, parser.ChangeTypeRemoved, changes[0].Type)
		assert.True(t, changes[0].Breaking)
	})

	t.Run("detects added types among existing ones", func(t *testing.T) {
		current := map[string]*parser.CompiledSchema{
			"Post":   {Name: "Post"},
			"Author": {Name: "Author"},
		}
		previous := map[string]*parser.CompiledSchema{
			"Post": {Name: "Post"},
		}
		changes := diffCurrentVsPrevious(current, previous)
		require.Len(t, changes, 1)
		assert.Equal(t, parser.ChangeTypeAdded, changes[0].Type)
		assert.Contains(t, changes[0].Path, "Author")
	})

	t.Run("detects modified fields on existing types", func(t *testing.T) {
		previousSchemas := parseSchemaSources(t, map[string]string{"a.fsl": `type Post { title: String }`})
		previousCompiled, err := parser.Compile(previousSchemas[0], "Post", "post", false)
		require.NoError(t, err)

		currentSchemas := parseSchemaSources(t, map[string]string{"a.fsl": `type Post { title: String! }`})
		currentCompiled, err := parser.Compile(currentSchemas[0], "Post", "post", false)
		require.NoError(t, err)

		current := map[string]*parser.CompiledSchema{"Post": currentCompiled}
		previous := map[string]*parser.CompiledSchema{"Post": previousCompiled}

		changes := diffCurrentVsPrevious(current, previous)
		require.NotEmpty(t, changes)
		for _, c := range changes {
			assert.Contains(t, c.Path, "Post.")
		}
	})
}

func TestCompileSchemasByType(t *testing.T) {
	t.Run("compiles types across schemas", func(t *testing.T) {
		schemas := parseSchemaSources(t, map[string]string{
			"a.fsl": `type Post { title: String! }`,
			"b.fsl": `type Author { name: String! }`,
		})

		compiled, names, err := compileSchemasByType(schemas)
		require.NoError(t, err)
		assert.ElementsMatch(t, []string{"Author", "Post"}, names)
		assert.Len(t, compiled, 2)
	})

	t.Run("duplicate type name errors", func(t *testing.T) {
		schemas := parseSchemaSources(t, map[string]string{
			"a.fsl": `type Post { title: String! }`,
			"b.fsl": `type Post { body: String }`,
		})

		_, _, err := compileSchemasByType(schemas)
		assert.Error(t, err)
	})

	t.Run("empty schemas errors", func(t *testing.T) {
		_, _, err := compileSchemasByType(nil)
		assert.Error(t, err)
	})
}

// parseSchemaSources parses each fsl source independently (no cross-file type resolution)
// and returns the resulting *parser.Schema list, failing the test on any parse error.
func parseSchemaSources(t *testing.T, sources map[string]string) []*parser.Schema {
	t.Helper()
	schemas := make([]*parser.Schema, 0, len(sources))
	for _, content := range sources {
		result := parser.ParseWithDiagnostics(content)
		require.True(t, result.Valid, "expected valid schema: %v", result.Diagnostics)
		schemas = append(schemas, result.Schema)
	}
	return schemas
}

func TestResolveDiffType(t *testing.T) {
	t.Run("explicit type must exist in both", func(t *testing.T) {
		_, err := resolveDiffType("Post", []string{"Author"}, []string{"Post"})
		assert.ErrorContains(t, err, "not found in --from")

		_, err = resolveDiffType("Post", []string{"Post"}, []string{"Author"})
		assert.ErrorContains(t, err, "not found in --to")

		name, err := resolveDiffType("Post", []string{"Post"}, []string{"Post"})
		require.NoError(t, err)
		assert.Equal(t, "Post", name)
	})

	t.Run("infers single common type", func(t *testing.T) {
		name, err := resolveDiffType("", []string{"Post"}, []string{"Post"})
		require.NoError(t, err)
		assert.Equal(t, "Post", name)
	})

	t.Run("ambiguous without explicit type errors", func(t *testing.T) {
		_, err := resolveDiffType("", []string{"Post", "Author"}, []string{"Post", "Author"})
		assert.ErrorContains(t, err, "multiple types detected")
	})
}

func TestFormatTypeList(t *testing.T) {
	assert.Equal(t, "none", formatTypeList(nil))
	assert.Equal(t, "Post", formatTypeList([]string{"Post"}))
	assert.Equal(t, "Post, Author", formatTypeList([]string{"Post", "Author"}))
}

func withMigrateFlags(t *testing.T, format string) {
	t.Helper()
	prev := migrateFormat
	migrateFormat = format
	t.Cleanup(func() { migrateFormat = prev })
}

func TestRunMigrateGenerate_CreatesMigrationFile(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "post.fsl"), []byte(`type Post { title: String! }`), 0o644))

	prevSchema, prevName := migrateSchemaPath, migrateName
	migrateSchemaPath, migrateName = dir, "add post"
	t.Cleanup(func() { migrateSchemaPath, migrateName = prevSchema, prevName })

	require.NoError(t, runMigrateGenerate(migrateGenerateCmd, nil))

	// migrations/ is created as a sibling of the schema directory, not inside it.
	entries, err := os.ReadDir(filepath.Join(filepath.Dir(dir), "migrations"))
	require.NoError(t, err)
	require.Len(t, entries, 1)
	assert.Contains(t, entries[0].Name(), "add_post")
}

func TestRunMigratePreview_NoChanges(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "post.fsl"), []byte(`type Post { title: String! }`), 0o644))
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "migrations"), 0o755))

	prevSchema := migrateSchemaPath
	migrateSchemaPath = dir
	t.Cleanup(func() { migrateSchemaPath = prevSchema })
	withMigrateFlags(t, "pretty")

	require.NoError(t, runMigratePreview(migratePreviewCmd, nil))
}

func TestRunMigratePreview_DetectsAddedType(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "post.fsl"), []byte(`type Post { title: String! }`), 0o644))

	prevSchema := migrateSchemaPath
	migrateSchemaPath = dir
	t.Cleanup(func() { migrateSchemaPath = prevSchema })
	withMigrateFlags(t, "json")

	require.NoError(t, runMigratePreview(migratePreviewCmd, nil))
}

func TestRunMigrateCheck_NoPreviousStateIsSafe(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "post.fsl"), []byte(`type Post { title: String! }`), 0o644))

	prevSchema := migrateSchemaPath
	migrateSchemaPath = dir
	t.Cleanup(func() { migrateSchemaPath = prevSchema })
	withMigrateFlags(t, "pretty")

	require.NoError(t, runMigrateCheck(migrateCheckCmd, nil))
}

func TestRunMigrateCheck_DetectsBreakingRemoval(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "post.fsl"), []byte(`type Post { title: String! }`), 0o644))

	prevSchema, prevName := migrateSchemaPath, migrateName
	migrateSchemaPath, migrateName = dir, "baseline"
	t.Cleanup(func() { migrateSchemaPath, migrateName = prevSchema, prevName })
	require.NoError(t, runMigrateGenerate(migrateGenerateCmd, nil))

	// Now remove the type entirely -- this is a breaking change vs the baseline migration.
	require.NoError(t, os.Remove(filepath.Join(dir, "post.fsl")))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "author.fsl"), []byte(`type Author { name: String! }`), 0o644))

	withMigrateFlags(t, "pretty")
	err := runMigrateCheck(migrateCheckCmd, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "breaking changes detected")
}

func TestRunMigrateDiff(t *testing.T) {
	fromDir := t.TempDir()
	toDir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(fromDir, "post.fsl"), []byte(`type Post { title: String! }`), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(toDir, "post.fsl"), []byte(`type Post { title: String! body: String }`), 0o644))

	prevFrom, prevTo, prevType, prevOutput := migrateDiffFrom, migrateDiffTo, migrateDiffType, migrateDiffOutput
	migrateDiffFrom, migrateDiffTo, migrateDiffType = fromDir, toDir, ""
	t.Cleanup(func() {
		migrateDiffFrom, migrateDiffTo, migrateDiffType, migrateDiffOutput = prevFrom, prevTo, prevType, prevOutput
	})

	t.Run("pretty output to stdout", func(t *testing.T) {
		migrateDiffOutput = ""
		withMigrateFlags(t, "pretty")
		require.NoError(t, runMigrateDiff(migrateDiffCmd, nil))
	})

	t.Run("json output to file", func(t *testing.T) {
		outFile := filepath.Join(t.TempDir(), "diff.json")
		migrateDiffOutput = outFile
		withMigrateFlags(t, "json")
		require.NoError(t, runMigrateDiff(migrateDiffCmd, nil))

		data, err := os.ReadFile(outFile)
		require.NoError(t, err)
		assert.Contains(t, string(data), `"type": "Post"`)
	})

	t.Run("type not found errors", func(t *testing.T) {
		migrateDiffOutput = ""
		migrateDiffType = "Missing"
		t.Cleanup(func() { migrateDiffType = "" })
		withMigrateFlags(t, "pretty")

		err := runMigrateDiff(migrateDiffCmd, nil)
		assert.Error(t, err)
	})
}
