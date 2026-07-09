package cmd

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseSeedJSON(t *testing.T) {
	t.Run("valid", func(t *testing.T) {
		docs, err := parseSeedJSON([]byte(`[{"schema":"post","data":{"title":"Hi"}}]`))
		require.NoError(t, err)
		require.Len(t, docs, 1)
		assert.Equal(t, "post", docs[0].Schema)
		assert.Equal(t, "Hi", docs[0].Data["title"])
	})

	t.Run("invalid", func(t *testing.T) {
		_, err := parseSeedJSON([]byte(`not json`))
		assert.Error(t, err)
	})
}

func TestParseSeedYAML(t *testing.T) {
	t.Run("valid", func(t *testing.T) {
		docs, err := parseSeedYAML([]byte("- schema: post\n  data:\n    title: Hi\n"))
		require.NoError(t, err)
		require.Len(t, docs, 1)
		assert.Equal(t, "post", docs[0].Schema)
	})

	t.Run("invalid", func(t *testing.T) {
		_, err := parseSeedYAML([]byte("not: [valid: yaml"))
		assert.Error(t, err)
	})
}

func TestLoadSeedFile(t *testing.T) {
	dir := t.TempDir()

	t.Run("json extension", func(t *testing.T) {
		p := filepath.Join(dir, "docs.json")
		require.NoError(t, os.WriteFile(p, []byte(`[{"schema":"post","data":{}}]`), 0o644))
		docs, err := loadSeedFile(p)
		require.NoError(t, err)
		assert.Len(t, docs, 1)
	})

	t.Run("yaml extension", func(t *testing.T) {
		p := filepath.Join(dir, "docs.yaml")
		require.NoError(t, os.WriteFile(p, []byte("- schema: post\n  data: {}\n"), 0o644))
		docs, err := loadSeedFile(p)
		require.NoError(t, err)
		assert.Len(t, docs, 1)
	})

	t.Run("unknown extension falls back to yaml after json fails", func(t *testing.T) {
		p := filepath.Join(dir, "docs.txt")
		require.NoError(t, os.WriteFile(p, []byte("- schema: post\n  data: {}\n"), 0o644))
		docs, err := loadSeedFile(p)
		require.NoError(t, err)
		assert.Len(t, docs, 1)
	})

	t.Run("missing file errors", func(t *testing.T) {
		_, err := loadSeedFile(filepath.Join(dir, "missing.json"))
		assert.Error(t, err)
	})
}

func withSeedFlags(t *testing.T, project, schema string) {
	t.Helper()
	prevProject, prevSchema := seedProject, seedSchema
	seedProject, seedSchema = project, schema
	t.Cleanup(func() {
		seedProject, seedSchema = prevProject, prevSchema
	})
}

func TestRunSeed_MissingSchemaField(t *testing.T) {
	dir := t.TempDir()
	seedFile := filepath.Join(dir, "seeds.json")
	require.NoError(t, os.WriteFile(seedFile, []byte(`[{"data":{"title":"no schema here"}}]`), 0o644))

	withConfig(t, nil)
	withEnv(t, "FSL_API_URL", "http://unused.invalid")
	withEnv(t, "FSL_API_KEY", "test-key")
	withSeedFlags(t, "proj_1", "")

	err := runSeed(seedCmd, []string{seedFile})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "1 failed")
}

func TestRunSeed_CreatesDocuments(t *testing.T) {
	dir := t.TempDir()
	seedFile := filepath.Join(dir, "seeds.json")
	require.NoError(t, os.WriteFile(seedFile, []byte(`[{"schema":"post","data":{"title":"Hi"}}]`), 0o644))

	var gotPath string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{}`))
	}))
	defer server.Close()

	withConfig(t, nil)
	withEnv(t, "FSL_API_URL", server.URL)
	withEnv(t, "FSL_API_KEY", "test-key")
	withSeedFlags(t, "proj_1", "")

	require.NoError(t, runSeed(seedCmd, []string{seedFile}))
	assert.Equal(t, "/api/v1/projects/proj_1/documents", gotPath)
}

func TestRunSeed_SchemaOverride(t *testing.T) {
	dir := t.TempDir()
	seedFile := filepath.Join(dir, "seeds.json")
	require.NoError(t, os.WriteFile(seedFile, []byte(`[{"data":{"title":"Hi"}}]`), 0o644))

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{}`))
	}))
	defer server.Close()

	withConfig(t, nil)
	withEnv(t, "FSL_API_URL", server.URL)
	withEnv(t, "FSL_API_KEY", "test-key")
	withSeedFlags(t, "proj_1", "post")

	require.NoError(t, runSeed(seedCmd, []string{seedFile}))
}

func TestRunSeed_EmptyFile(t *testing.T) {
	dir := t.TempDir()
	seedFile := filepath.Join(dir, "empty.json")
	require.NoError(t, os.WriteFile(seedFile, []byte(`[]`), 0o644))

	withSeedFlags(t, "proj_1", "")
	require.NoError(t, runSeed(seedCmd, []string{seedFile}))
}
