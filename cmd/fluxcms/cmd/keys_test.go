package cmd

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTruncate(t *testing.T) {
	assert.Equal(t, "abc", truncate("abc", 5), "shorter than limit is unchanged")
	assert.Equal(t, "abcde", truncate("abcde", 5), "exact length is unchanged")
	assert.Equal(t, "ab...", truncate("abcdefgh", 5), "over limit truncates with ellipsis")
	assert.Equal(t, "ab", truncate("abcdefgh", 2), "limit too small for ellipsis returns raw prefix")
	assert.Equal(t, "日本語", truncate("日本語", 5), "multi-byte runes within limit are unchanged")
}

func TestRunKeysCreate_InvalidScope(t *testing.T) {
	prevScope := keysCreateScope
	keysCreateScope = "bogus"
	t.Cleanup(func() { keysCreateScope = prevScope })

	err := runKeysCreate(keysCreateCmd, []string{"My Key"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid scope")
}

func TestKeysCommands_ViaServer(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == "GET" && r.URL.Path == "/api/v1/api-keys":
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"keys":[{"id":"key_1","name":"CI","scope":"read-only","prefix":"fsl_abc","createdAt":"2024-01-01T00:00:00Z"}]}`))
		case r.Method == "POST" && r.URL.Path == "/api/v1/api-keys":
			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write([]byte(`{"id":"key_2","name":"Deploy Bot","scope":"admin","key":"fsl_secret123"}`))
		case r.Method == "DELETE" && r.URL.Path == "/api/v1/api-keys/key_1":
			w.WriteHeader(http.StatusNoContent)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	withConfig(t, nil)
	withEnv(t, "FSL_API_URL", server.URL)
	withEnv(t, "FSL_API_KEY", "test-key")

	t.Run("list", func(t *testing.T) {
		require.NoError(t, runKeysList(keysListCmd, nil))
	})

	t.Run("create", func(t *testing.T) {
		prevScope := keysCreateScope
		keysCreateScope = "admin"
		t.Cleanup(func() { keysCreateScope = prevScope })

		require.NoError(t, runKeysCreate(keysCreateCmd, []string{"Deploy Bot"}))
	})

	t.Run("revoke", func(t *testing.T) {
		require.NoError(t, runKeysRevoke(keysRevokeCmd, []string{"key_1"}))
	})
}

func TestRunKeysList_EmptyServer(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`[]`))
	}))
	defer server.Close()

	withConfig(t, nil)
	withEnv(t, "FSL_API_URL", server.URL)
	withEnv(t, "FSL_API_KEY", "test-key")

	require.NoError(t, runKeysList(keysListCmd, nil))
}

func TestRunKeysRevoke_ServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	withConfig(t, nil)
	withEnv(t, "FSL_API_URL", server.URL)
	withEnv(t, "FSL_API_KEY", "test-key")

	err := runKeysRevoke(keysRevokeCmd, []string{"missing_key"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to revoke API key")
}
