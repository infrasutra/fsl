package cmd

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// withConfig temporarily swaps the package-level config and restores it after the test.
func withConfig(t *testing.T, c *Config) {
	t.Helper()
	prev := config
	config = c
	t.Cleanup(func() { config = prev })
}

// withEnv sets an environment variable for the duration of the test and restores it after.
func withEnv(t *testing.T, key, value string) {
	t.Helper()
	t.Setenv(key, value)
}

func TestUnmarshalListResponse(t *testing.T) {
	t.Run("plain array", func(t *testing.T) {
		items, err := unmarshalListResponse[apiKey]([]byte(`[{"id":"1"},{"id":"2"}]`))
		require.NoError(t, err)
		assert.Len(t, items, 2)
	})

	t.Run("wrapped in first matching key", func(t *testing.T) {
		items, err := unmarshalListResponse[apiKey]([]byte(`{"data":[{"id":"1"}]}`), "keys", "data")
		require.NoError(t, err)
		require.Len(t, items, 1)
		assert.Equal(t, "1", items[0].ID)
	})

	t.Run("wrapped in preferred key over later key", func(t *testing.T) {
		items, err := unmarshalListResponse[apiKey]([]byte(`{"keys":[{"id":"k"}],"data":[{"id":"d"}]}`), "keys", "data")
		require.NoError(t, err)
		require.Len(t, items, 1)
		assert.Equal(t, "k", items[0].ID)
	})

	t.Run("no wrapper key present returns nil", func(t *testing.T) {
		items, err := unmarshalListResponse[apiKey]([]byte(`{"other":[]}`), "keys", "data")
		require.NoError(t, err)
		assert.Nil(t, items)
	})

	t.Run("invalid json errors", func(t *testing.T) {
		_, err := unmarshalListResponse[apiKey]([]byte(`not json`), "keys")
		assert.Error(t, err)
	})
}

func TestNewAPIClient(t *testing.T) {
	t.Run("missing key errors", func(t *testing.T) {
		withConfig(t, nil)
		withEnv(t, "FSL_API_URL", "")
		withEnv(t, "FSL_API_KEY", "")

		_, err := newAPIClient()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "API key is required")
	})

	t.Run("env vars override config and defaults", func(t *testing.T) {
		withConfig(t, &Config{})
		withEnv(t, "FSL_API_URL", "https://env.example.com")
		withEnv(t, "FSL_API_KEY", "env-key")

		client, err := newAPIClient()
		require.NoError(t, err)
		assert.Equal(t, "https://env.example.com", client.baseURL)
		assert.Equal(t, "env-key", client.apiKey)
	})

	t.Run("config supplies values when env unset", func(t *testing.T) {
		cfg := &Config{}
		cfg.Workspace.APIURL = "https://cfg.example.com"
		cfg.Workspace.APIKey = "cfg-key"
		withConfig(t, cfg)
		withEnv(t, "FSL_API_URL", "")
		withEnv(t, "FSL_API_KEY", "")

		client, err := newAPIClient()
		require.NoError(t, err)
		assert.Equal(t, "https://cfg.example.com", client.baseURL)
		assert.Equal(t, "cfg-key", client.apiKey)
	})

	t.Run("falls back to default URL", func(t *testing.T) {
		withConfig(t, nil)
		withEnv(t, "FSL_API_URL", "")
		withEnv(t, "FSL_API_KEY", "some-key")

		client, err := newAPIClient()
		require.NoError(t, err)
		assert.Equal(t, defaultAPIURL, client.baseURL)
	})
}

func TestAPIClient_ApiRequest(t *testing.T) {
	t.Run("success round trip", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "Bearer test-key", r.Header.Get("Authorization"))
			assert.Equal(t, "application/json", r.Header.Get("Content-Type"))
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"ok":true}`))
		}))
		defer server.Close()

		client := &apiClient{baseURL: server.URL, apiKey: "test-key", httpClient: server.Client()}
		body, err := client.apiRequest("POST", "/api/v1/thing", map[string]string{"a": "b"})
		require.NoError(t, err)
		assert.JSONEq(t, `{"ok":true}`, string(body))
	})

	t.Run("nil body sends no payload", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, int64(0), r.ContentLength)
			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()

		client := &apiClient{baseURL: server.URL, apiKey: "test-key", httpClient: server.Client()}
		_, err := client.apiRequest("GET", "/api/v1/thing", nil)
		require.NoError(t, err)
	})

	t.Run("server error surfaces handleAPIError message", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusUnauthorized)
		}))
		defer server.Close()

		client := &apiClient{baseURL: server.URL, apiKey: "test-key", httpClient: server.Client()}
		_, err := client.apiRequest("GET", "/api/v1/thing", nil)
		assert.ErrorContains(t, err, "authentication failed")
	})

	t.Run("unreachable server errors", func(t *testing.T) {
		client := &apiClient{baseURL: "http://127.0.0.1:1", apiKey: "k", httpClient: http.DefaultClient}
		_, err := client.apiRequest("GET", "/x", nil)
		assert.ErrorContains(t, err, "request failed")
	})
}

func TestHandleAPIError(t *testing.T) {
	cases := []struct {
		name    string
		status  int
		body    string
		wantErr string
	}{
		{"200 ok", http.StatusOK, "", ""},
		{"201 created", http.StatusCreated, "", ""},
		{"202 accepted", http.StatusAccepted, "", ""},
		{"204 no content", http.StatusNoContent, "", ""},
		{"401 unauthorized", http.StatusUnauthorized, "", "authentication failed (401)"},
		{"403 forbidden", http.StatusForbidden, "", "access denied (403)"},
		{"404 not found", http.StatusNotFound, "", "not found (404)"},
		{"422 with message", http.StatusUnprocessableEntity, `{"message":"bad field"}`, "validation error (422): bad field"},
		{"422 without message", http.StatusUnprocessableEntity, "", "validation error (422): the server rejected"},
		{"500 server error", http.StatusInternalServerError, "", "server error (500)"},
		{"other status with message", http.StatusTeapot, `{"error":"teapot"}`, "request failed (418): teapot"},
		{"other status without message", http.StatusTeapot, "", "request failed with status 418"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := handleAPIError(tc.status, []byte(tc.body))
			if tc.wantErr == "" {
				assert.NoError(t, err)
				return
			}
			require.Error(t, err)
			assert.Contains(t, err.Error(), tc.wantErr)
		})
	}
}

func TestExtractErrorMessage(t *testing.T) {
	assert.Equal(t, "", extractErrorMessage(nil))
	assert.Equal(t, "", extractErrorMessage([]byte("")))
	assert.Equal(t, "", extractErrorMessage([]byte("not json")))
	assert.Equal(t, "boom", extractErrorMessage([]byte(`{"message":"boom"}`)))
	assert.Equal(t, "boom2", extractErrorMessage([]byte(`{"error":"boom2"}`)))
	assert.Equal(t, "boom", extractErrorMessage([]byte(`{"message":"boom","error":"boom2"}`)))
}
