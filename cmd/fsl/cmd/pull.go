package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

var pullOutput string

var pullCmd = &cobra.Command{
	Use:   "pull",
	Short: "Pull schemas from the CMS server",
	Long: `Download FSL schemas from the CMS server to a local directory.

Schemas are written as .fsl files using the schema name as the filename.
Existing files with the same name will be overwritten.

Examples:
  # Pull schemas into the configured schema directory
  fsl pull

  # Pull schemas into a custom directory
  fsl pull --output ./downloaded-schemas`,
	RunE: runPull,
}

func init() {
	rootCmd.AddCommand(pullCmd)
	pullCmd.Flags().StringVar(&pullOutput, "output", "", "Directory to write downloaded schemas (defaults to configured schema directory)")
}

// schemaListItem represents one schema entry returned by GET /api/v1/schemas.
type schemaListItem struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Content string `json:"content"`
	Slug    string `json:"slug"`
}

func runPull(cmd *cobra.Command, args []string) error {
	outputDir := pullOutput
	if outputDir == "" {
		outputDir = GetSchemaDirectory()
	}

	client, err := newAPIClient()
	if err != nil {
		return err
	}

	_, body, err := client.apiRequest("GET", "/api/v1/schemas", nil)
	if err != nil {
		return fmt.Errorf("failed to fetch schemas: %w", err)
	}

	// The server may return a top-level array or a wrapper object.
	var schemas []schemaListItem
	if err := json.Unmarshal(body, &schemas); err != nil {
		// Try wrapped format: { "schemas": [...] } or { "data": [...] }
		var wrapper struct {
			Schemas []schemaListItem `json:"schemas"`
			Data    []schemaListItem `json:"data"`
		}
		if err2 := json.Unmarshal(body, &wrapper); err2 != nil {
			return fmt.Errorf("failed to parse server response: %w", err)
		}
		if len(wrapper.Schemas) > 0 {
			schemas = wrapper.Schemas
		} else {
			schemas = wrapper.Data
		}
	}

	if len(schemas) == 0 {
		fmt.Println("No schemas found on the server.")
		return nil
	}

	// Ensure output directory exists.
	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		return fmt.Errorf("failed to create output directory %s: %w", outputDir, err)
	}

	downloaded := 0
	for _, schema := range schemas {
		filename := schemaFilename(schema)
		filePath := filepath.Join(outputDir, filename)

		if err := os.WriteFile(filePath, []byte(schema.Content), 0o644); err != nil {
			fmt.Printf("\033[31m✗\033[0m %s — write failed: %v\n", filename, err)
			continue
		}

		fmt.Printf("\033[32m✓\033[0m %s\n", filePath)
		downloaded++
	}

	fmt.Println()
	fmt.Printf("\033[32mPull complete:\033[0m %d schema(s) downloaded to %s\n", downloaded, outputDir)
	return nil
}

// schemaFilename returns an .fsl filename for the given schema.
// Prefers the slug field, then name, then the ID.
func schemaFilename(s schemaListItem) string {
	base := s.Slug
	if base == "" {
		base = s.Name
	}
	if base == "" {
		base = s.ID
	}
	if filepath.Ext(base) != ".fsl" {
		base += ".fsl"
	}
	return base
}
