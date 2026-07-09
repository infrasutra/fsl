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

func TestSchemaName(t *testing.T) {
	assert.Equal(t, "post", schemaName(filepath.Join("schemas", "post.fsl"), "schemas"))
	assert.Equal(t, "blog.post", schemaName(filepath.Join("schemas", "blog", "post.fsl"), "schemas"))
	assert.Equal(t, "post", schemaName("/unrelated/post.fsl", "schemas"), "falls back to base name when not relative")
}

func withPushFlags(t *testing.T, dryRun, force bool) {
	t.Helper()
	prevDry, prevForce := pushDryRun, pushForce
	pushDryRun, pushForce = dryRun, force
	t.Cleanup(func() {
		pushDryRun, pushForce = prevDry, prevForce
	})
}

func TestRunPush_DryRun(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "post.fsl"), []byte(`type Post { title: String! }`), 0o644))

	prevCfg := config
	config = &Config{}
	config.Schemas.Directory = dir
	t.Cleanup(func() { config = prevCfg })

	withPushFlags(t, true, false)

	require.NoError(t, runPush(pushCmd, nil))
}

func TestRunPush_ValidationErrorsAbort(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "bad.fsl"), []byte(`type Post { title: `), 0o644))

	prevCfg := config
	config = &Config{}
	config.Schemas.Directory = dir
	t.Cleanup(func() { config = prevCfg })

	withPushFlags(t, false, false)

	err := runPush(pushCmd, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "fix validation errors")
}

func TestRunPush_PushesToServer(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "post.fsl"), []byte(`type Post { title: String! }`), 0o644))

	var received []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		received = append(received, r.URL.Path)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"message":"synced"}`))
	}))
	defer server.Close()

	prevCfg := config
	config = &Config{}
	config.Schemas.Directory = dir
	t.Cleanup(func() { config = prevCfg })

	withEnv(t, "FSL_API_URL", server.URL)
	withEnv(t, "FSL_API_KEY", "test-key")
	withPushFlags(t, false, false)

	require.NoError(t, runPush(pushCmd, nil))
	assert.Equal(t, []string{"/api/v1/schemas/sync"}, received)
}

func TestRunPush_ForcePushesMultipleFiles(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "post.fsl"), []byte(`type Post { title: String! }`), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "author.fsl"), []byte(`type Author { name: String! }`), 0o644))

	var count int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		count++
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{}`))
	}))
	defer server.Close()

	prevCfg := config
	config = &Config{}
	config.Schemas.Directory = dir
	t.Cleanup(func() { config = prevCfg })

	withEnv(t, "FSL_API_URL", server.URL)
	withEnv(t, "FSL_API_KEY", "test-key")
	withPushFlags(t, false, true)

	require.NoError(t, runPush(pushCmd, nil))
	assert.Equal(t, 2, count)
}

func TestRunPush_ServerFailurePartial(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "post.fsl"), []byte(`type Post { title: String! }`), 0o644))

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	prevCfg := config
	config = &Config{}
	config.Schemas.Directory = dir
	t.Cleanup(func() { config = prevCfg })

	withEnv(t, "FSL_API_URL", server.URL)
	withEnv(t, "FSL_API_KEY", "test-key")
	withPushFlags(t, false, false)

	err := runPush(pushCmd, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "push completed with errors")
}
