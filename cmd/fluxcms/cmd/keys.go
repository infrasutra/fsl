package cmd

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

var keysCreateScope string

var keysCmd = &cobra.Command{
	Use:   "keys",
	Short: "Manage API keys",
	Long: `Manage API keys for the CMS workspace.

Subcommands allow listing, creating, and revoking API keys.`,
}

var keysListCmd = &cobra.Command{
	Use:   "list",
	Short: "List API keys",
	Long: `List all API keys for the current workspace.

Examples:
  fluxcms keys list`,
	RunE: runKeysList,
}

var keysCreateCmd = &cobra.Command{
	Use:   "create [name]",
	Short: "Create a new API key",
	Long: `Create a new API key with the given name.

Examples:
  # Create a read-only API key
  fluxcms keys create "CI Read Key" --scope read-only

  # Create an admin API key
  fluxcms keys create "Deploy Bot" --scope admin`,
	Args: cobra.ExactArgs(1),
	RunE: runKeysCreate,
}

var keysRevokeCmd = &cobra.Command{
	Use:   "revoke [key-id]",
	Short: "Revoke an API key",
	Long: `Revoke an API key by its ID. This action cannot be undone.

Examples:
  fluxcms keys revoke key_abc123`,
	Args: cobra.ExactArgs(1),
	RunE: runKeysRevoke,
}

func init() {
	rootCmd.AddCommand(keysCmd)
	keysCmd.AddCommand(keysListCmd)
	keysCmd.AddCommand(keysCreateCmd)
	keysCmd.AddCommand(keysRevokeCmd)

	keysCreateCmd.Flags().StringVar(&keysCreateScope, "scope", "read-only", "API key scope: admin, content, read-only")
}

// apiKey represents one API key entry returned by the server.
type apiKey struct {
	ID         string     `json:"id"`
	Name       string     `json:"name"`
	Scope      string     `json:"scope"`
	Prefix     string     `json:"prefix"`
	CreatedAt  time.Time  `json:"createdAt"`
	LastUsedAt *time.Time `json:"lastUsedAt"`
}

// apiKeyCreateRequest is the payload for POST /api/v1/api-keys.
type apiKeyCreateRequest struct {
	Name  string `json:"name"`
	Scope string `json:"scope"`
}

// apiKeyCreateResponse includes the plaintext key returned only at creation time.
type apiKeyCreateResponse struct {
	apiKey
	Key string `json:"key"`
}

func runKeysList(cmd *cobra.Command, args []string) error {
	client, err := newAPIClient()
	if err != nil {
		return err
	}

	body, err := client.apiRequest("GET", "/api/v1/api-keys", nil)
	if err != nil {
		return fmt.Errorf("failed to list API keys: %w", err)
	}

	keys, err := unmarshalListResponse[apiKey](body, "keys", "data")
	if err != nil {
		return err
	}

	if len(keys) == 0 {
		fmt.Println("No API keys found.")
		return nil
	}

	printKeysTable(keys)
	return nil
}

func runKeysCreate(cmd *cobra.Command, args []string) error {
	name := args[0]

	validScopes := map[string]bool{"admin": true, "content": true, "read-only": true}
	if !validScopes[keysCreateScope] {
		return fmt.Errorf("invalid scope %q: must be one of admin, content, read-only", keysCreateScope)
	}

	client, err := newAPIClient()
	if err != nil {
		return err
	}

	payload := apiKeyCreateRequest{
		Name:  name,
		Scope: keysCreateScope,
	}

	body, err := client.apiRequest("POST", "/api/v1/api-keys", payload)
	if err != nil {
		return fmt.Errorf("failed to create API key: %w", err)
	}

	var result apiKeyCreateResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return fmt.Errorf("failed to parse server response: %w", err)
	}

	fmt.Printf("\033[32m✓\033[0m API key created\n\n")
	fmt.Printf("  Name:  %s\n", result.Name)
	fmt.Printf("  ID:    %s\n", result.ID)
	fmt.Printf("  Scope: %s\n", result.Scope)
	if result.Key != "" {
		fmt.Printf("  Key:   %s\n", result.Key)
		fmt.Println()
		fmt.Println("\033[33mWarning:\033[0m Save this key now — it will not be shown again.")
	}

	return nil
}

func runKeysRevoke(cmd *cobra.Command, args []string) error {
	keyID := args[0]

	client, err := newAPIClient()
	if err != nil {
		return err
	}

	_, err = client.apiRequest("DELETE", "/api/v1/api-keys/"+url.PathEscape(keyID), nil)
	if err != nil {
		return fmt.Errorf("failed to revoke API key %s: %w", keyID, err)
	}

	fmt.Printf("\033[32m✓\033[0m API key %s revoked\n", keyID)
	return nil
}

// printKeysTable renders a simple aligned table of API keys.
func printKeysTable(keys []apiKey) {
	const (
		colID       = 24
		colName     = 24
		colScope    = 12
		colPrefix   = 12
		colCreated  = 20
		colLastUsed = 20
	)

	header := fmt.Sprintf("%-*s  %-*s  %-*s  %-*s  %-*s  %s",
		colID, "ID",
		colName, "NAME",
		colScope, "SCOPE",
		colPrefix, "PREFIX",
		colCreated, "CREATED",
		"LAST USED",
	)
	fmt.Println(header)
	fmt.Println(strings.Repeat("-", len(header)+4))

	for _, k := range keys {
		lastUsed := "never"
		if k.LastUsedAt != nil {
			lastUsed = k.LastUsedAt.Format("2006-01-02 15:04")
		}

		fmt.Printf("%-*s  %-*s  %-*s  %-*s  %-*s  %s\n",
			colID, truncate(k.ID, colID),
			colName, truncate(k.Name, colName),
			colScope, truncate(k.Scope, colScope),
			colPrefix, truncate(k.Prefix, colPrefix),
			colCreated, k.CreatedAt.Format("2006-01-02 15:04"),
			lastUsed,
		)
	}
}

// truncate shortens s to at most n characters, appending "..." if truncated.
func truncate(s string, n int) string {
	runes := []rune(s)
	if len(runes) <= n {
		return s
	}
	if n <= 3 {
		return string(runes[:n])
	}
	return string(runes[:n-3]) + "..."
}
