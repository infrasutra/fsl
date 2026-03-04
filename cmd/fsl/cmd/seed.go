package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

var (
	seedProject string
	seedSchema  string
)

var seedCmd = &cobra.Command{
	Use:   "seed [file]",
	Short: "Seed content from JSON/YAML files",
	Long: `Upload content documents from local JSON or YAML files to the CMS.

The seed file must contain an array of document objects, each with a "schema"
field (schema slug or ID) and a "data" field (the document fields). The file
format is detected by extension (.json or .yaml/.yml).

Examples:
  # Seed documents from a JSON file
  fsl seed ./seeds/posts.json --project proj_abc123

  # Seed documents from a YAML file, overriding schema for all documents
  fsl seed ./seeds/posts.yaml --project proj_abc123 --schema post

Seed file format (JSON):
  [
    { "schema": "post", "data": { "title": "Hello World", "status": "draft" } },
    { "schema": "author", "data": { "name": "Jane Doe", "email": "jane@example.com" } }
  ]`,
	Args: cobra.ExactArgs(1),
	RunE: runSeed,
}

func init() {
	rootCmd.AddCommand(seedCmd)
	seedCmd.Flags().StringVar(&seedProject, "project", "", "Project ID to seed documents into (required)")
	seedCmd.Flags().StringVar(&seedSchema, "schema", "", "Override schema slug/ID for all documents in the file")
	_ = seedCmd.MarkFlagRequired("project")
}

// seedDocument represents one entry in the seed file.
type seedDocument struct {
	Schema string                 `json:"schema" yaml:"schema"`
	Data   map[string]interface{} `json:"data"   yaml:"data"`
}

// createDocumentRequest is the payload sent to POST /api/v1/projects/{id}/documents.
type createDocumentRequest struct {
	Schema string                 `json:"schema"`
	Data   map[string]interface{} `json:"data"`
}

func runSeed(cmd *cobra.Command, args []string) error {
	filePath := args[0]

	docs, err := loadSeedFile(filePath)
	if err != nil {
		return err
	}

	if len(docs) == 0 {
		fmt.Println("Seed file contains no documents.")
		return nil
	}

	client, err := newAPIClient()
	if err != nil {
		return err
	}

	apiPath := fmt.Sprintf("/api/v1/projects/%s/documents", seedProject)
	created := 0
	failed := 0

	fmt.Printf("Seeding %d document(s) into project %s...\n\n", len(docs), seedProject)

	for i, doc := range docs {
		schema := doc.Schema
		if seedSchema != "" {
			schema = seedSchema
		}
		if schema == "" {
			fmt.Printf("\033[31m✗\033[0m Document %d — missing \"schema\" field (use --schema to set one)\n", i+1)
			failed++
			continue
		}

		payload := createDocumentRequest{
			Schema: schema,
			Data:   doc.Data,
		}

		_, _, err := client.apiRequest("POST", apiPath, payload)
		if err != nil {
			fmt.Printf("\033[31m✗\033[0m Document %d (%s) — %v\n", i+1, schema, err)
			failed++
			continue
		}

		fmt.Printf("\033[32m✓\033[0m Document %d (%s) — created\n", i+1, schema)
		created++
	}

	fmt.Println()
	if failed > 0 {
		return fmt.Errorf("seed completed with errors: %d created, %d failed", created, failed)
	}

	fmt.Printf("\033[32mSeed complete:\033[0m %d document(s) created\n", created)
	return nil
}

// loadSeedFile reads and parses a JSON or YAML seed file.
func loadSeedFile(path string) ([]seedDocument, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("cannot read seed file %s: %w", path, err)
	}

	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".yaml", ".yml":
		return parseSeedYAML(data)
	case ".json":
		return parseSeedJSON(data)
	default:
		// Try JSON first, then YAML.
		docs, err := parseSeedJSON(data)
		if err != nil {
			return parseSeedYAML(data)
		}
		return docs, nil
	}
}

func parseSeedJSON(data []byte) ([]seedDocument, error) {
	var docs []seedDocument
	if err := json.Unmarshal(data, &docs); err != nil {
		return nil, fmt.Errorf("failed to parse JSON seed file: %w", err)
	}
	return docs, nil
}

func parseSeedYAML(data []byte) ([]seedDocument, error) {
	var docs []seedDocument
	if err := yaml.Unmarshal(data, &docs); err != nil {
		return nil, fmt.Errorf("failed to parse YAML seed file: %w", err)
	}
	return docs, nil
}
