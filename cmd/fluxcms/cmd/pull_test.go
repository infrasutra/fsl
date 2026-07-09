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

func TestSchemaFilename(t *testing.T) {
	assert.Equal(t, "post.fsl", schemaFilename(schemaListItem{Slug: "post"}), "prefers slug")
	assert.Equal(t, "post.fsl", schemaFilename(schemaListItem{Name: "post"}), "falls back to name")
	assert.Equal(t, "id123.fsl", schemaFilename(schemaListItem{ID: "id123"}), "falls back to id")
	assert.Equal(t, "post.fsl", schemaFilename(schemaListItem{Slug: "post.fsl"}), "does not double the extension")
}

func TestRunPull_DownloadsSchemas(t *testing.T) {
	outDir := t.TempDir()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/v1/schemas", r.URL.Path)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"schemas":[{"id":"1","slug":"post","content":"type Post { title: String! }"}]}`))
	}))
	defer server.Close()

	prevOut := pullOutput
	pullOutput = outDir
	t.Cleanup(func() { pullOutput = prevOut })

	withConfig(t, nil)
	withEnv(t, "FSL_API_URL", server.URL)
	withEnv(t, "FSL_API_KEY", "test-key")

	require.NoError(t, runPull(pullCmd, nil))

	content, err := os.ReadFile(filepath.Join(outDir, "post.fsl"))
	require.NoError(t, err)
	assert.Equal(t, "type Post { title: String! }", string(content))
}

func TestRunPull_NoSchemas(t *testing.T) {
	outDir := t.TempDir()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`[]`))
	}))
	defer server.Close()

	prevOut := pullOutput
	pullOutput = outDir
	t.Cleanup(func() { pullOutput = prevOut })

	withConfig(t, nil)
	withEnv(t, "FSL_API_URL", server.URL)
	withEnv(t, "FSL_API_KEY", "test-key")

	require.NoError(t, runPull(pullCmd, nil))

	entries, err := os.ReadDir(outDir)
	require.NoError(t, err)
	assert.Empty(t, entries)
}

func TestRunPull_MissingAPIKey(t *testing.T) {
	prevOut := pullOutput
	pullOutput = t.TempDir()
	t.Cleanup(func() { pullOutput = prevOut })

	withConfig(t, nil)
	withEnv(t, "FSL_API_URL", "")
	withEnv(t, "FSL_API_KEY", "")

	err := runPull(pullCmd, nil)
	assert.Error(t, err)
}
