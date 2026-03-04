package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"
)

// unmarshalListResponse tries to decode body as []T, then falls back to
// wrapper objects keyed by wrapperKeys (e.g. {"keys":[…], "data":[…]}).
func unmarshalListResponse[T any](body []byte, wrapperKeys ...string) ([]T, error) {
	var items []T
	if err := json.Unmarshal(body, &items); err == nil {
		return items, nil
	}
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, fmt.Errorf("failed to parse server response: %w", err)
	}
	for _, key := range wrapperKeys {
		if data, ok := raw[key]; ok {
			var items []T
			if err := json.Unmarshal(data, &items); err == nil && len(items) > 0 {
				return items, nil
			}
		}
	}
	return nil, nil
}

const defaultAPIURL = "http://localhost:7080"

// apiClient holds configuration for making API requests.
type apiClient struct {
	baseURL    string
	apiKey     string
	httpClient *http.Client
}

// newAPIClient creates an HTTP client configured from the loaded config.
// Falls back to environment variables and defaults when config values are absent.
func newAPIClient() (*apiClient, error) {
	apiURL := defaultAPIURL
	apiKey := ""

	if config != nil && config.Workspace.APIURL != "" {
		apiURL = config.Workspace.APIURL
	}
	if config != nil && config.Workspace.APIKey != "" {
		apiKey = config.Workspace.APIKey
	}

	// Allow environment variable overrides
	if v := os.Getenv("FSL_API_URL"); v != "" {
		apiURL = v
	}
	if v := os.Getenv("FSL_API_KEY"); v != "" {
		apiKey = v
	}

	if apiKey == "" {
		return nil, fmt.Errorf("API key is required: set api_key in .fsl.yaml or FSL_API_KEY environment variable")
	}

	return &apiClient{
		baseURL: apiURL,
		apiKey:  apiKey,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}, nil
}

// apiRequest performs an HTTP request to the given path with an optional JSON body.
// path should start with "/" (e.g. "/api/v1/schemas").
func (c *apiClient) apiRequest(method, path string, body interface{}) ([]byte, error) {
	var bodyReader io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("failed to encode request body: %w", err)
		}
		bodyReader = bytes.NewReader(data)
	}

	url := c.baseURL + path
	req, err := http.NewRequest(method, url, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if err := handleAPIError(resp.StatusCode, respBody); err != nil {
		return respBody, err
	}

	return respBody, nil
}

// apiRequestRaw performs a request with a raw byte body (e.g. plain text schema content).
func (c *apiClient) apiRequestRaw(method, path string, contentType string, body []byte) ([]byte, error) {
	var bodyReader io.Reader
	if body != nil {
		bodyReader = bytes.NewReader(body)
	}

	url := c.baseURL + path
	req, err := http.NewRequest(method, url, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if err := handleAPIError(resp.StatusCode, respBody); err != nil {
		return respBody, err
	}

	return respBody, nil
}

// handleAPIError maps HTTP status codes to user-friendly error messages.
func handleAPIError(statusCode int, body []byte) error {
	switch statusCode {
	case http.StatusOK, http.StatusCreated, http.StatusAccepted, http.StatusNoContent:
		return nil
	case http.StatusUnauthorized:
		return fmt.Errorf("authentication failed (401): check your API key")
	case http.StatusForbidden:
		return fmt.Errorf("access denied (403): your API key does not have permission for this operation")
	case http.StatusNotFound:
		return fmt.Errorf("not found (404): the requested resource does not exist")
	case http.StatusUnprocessableEntity:
		msg := extractErrorMessage(body)
		if msg != "" {
			return fmt.Errorf("validation error (422): %s", msg)
		}
		return fmt.Errorf("validation error (422): the server rejected the request data")
	case http.StatusInternalServerError:
		return fmt.Errorf("server error (500): the CMS server encountered an internal error")
	default:
		msg := extractErrorMessage(body)
		if msg != "" {
			return fmt.Errorf("request failed (%d): %s", statusCode, msg)
		}
		return fmt.Errorf("request failed with status %d", statusCode)
	}
}

// extractErrorMessage tries to pull an error message from a JSON response body.
func extractErrorMessage(body []byte) string {
	if len(body) == 0 {
		return ""
	}
	var envelope struct {
		Message string `json:"message"`
		Error   string `json:"error"`
	}
	if err := json.Unmarshal(body, &envelope); err != nil {
		return ""
	}
	if envelope.Message != "" {
		return envelope.Message
	}
	return envelope.Error
}
