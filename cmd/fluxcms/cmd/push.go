package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/infrasutra/fsl/parser"
	"github.com/spf13/cobra"
)

var (
	pushDryRun bool
	pushForce  bool
)

var pushCmd = &cobra.Command{
	Use:   "push",
	Short: "Push local schemas to the CMS server",
	Long: `Upload FSL schema files to the CMS server for deployment.

Each .fsl file in the configured schema directory is validated locally before
being uploaded. Use --dry-run to validate without pushing, or --force to push
even when validation produces warnings.

Examples:
  # Push all schemas to the server
  fluxcms push

  # Validate locally without uploading
  fluxcms push --dry-run

  # Push even if schemas have warnings
  fluxcms push --force`,
	RunE: runPush,
}

func init() {
	rootCmd.AddCommand(pushCmd)
	pushCmd.Flags().BoolVar(&pushDryRun, "dry-run", false, "Validate schemas locally without pushing to the server")
	pushCmd.Flags().BoolVar(&pushForce, "force", false, "Push even if validation produces warnings")
}

// schemaSyncRequest is the payload sent to /api/v1/schemas/sync.
type schemaSyncRequest struct {
	Name    string `json:"name"`
	Content string `json:"content"`
}

func runPush(cmd *cobra.Command, args []string) error {
	schemaDir := GetSchemaDirectory()

	files, err := collectFSLFiles([]string{schemaDir})
	if err != nil {
		return fmt.Errorf("failed to collect schema files from %s: %w", schemaDir, err)
	}

	fmt.Printf("Found %d schema file(s) in %s\n\n", len(files), schemaDir)

	// Validate all files first (fail fast).
	type fileEntry struct {
		path    string
		content string
		result  *parser.DiagnosticsResult
	}

	entries := make([]fileEntry, 0, len(files))
	hasErrors := false
	hasWarnings := false
	fileContents := make(map[string]string, len(files))

	for _, f := range files {
		content, err := os.ReadFile(f)
		if err != nil {
			fmt.Printf("\033[31m✗\033[0m %s — cannot read file: %v\n", f, err)
			hasErrors = true
			continue
		}
		fileContents[f] = string(content)
	}

	results := parseFilesWithWorkspaceTypes(fileContents)

	for _, f := range files {
		content, ok := fileContents[f]
		if !ok {
			continue
		}

		result := results[f]

		for _, diag := range result.Diagnostics {
			if diag.Severity == parser.SeverityError {
				hasErrors = true
			} else if diag.Severity == parser.SeverityWarning {
				hasWarnings = true
			}
		}

		if result.Valid {
			if len(result.Diagnostics) > 0 {
				fmt.Printf("\033[33m!\033[0m %s — valid with warnings\n", f)
				for _, diag := range result.Diagnostics {
					fmt.Printf("    %s:%d:%d [%s] %s\n",
						filepath.Base(f), diag.StartLine, diag.StartColumn,
						getSeverityLabel(diag.Severity), diag.Message)
				}
			} else {
				fmt.Printf("\033[32m✓\033[0m %s\n", f)
			}
		} else {
			fmt.Printf("\033[31m✗\033[0m %s — invalid\n", f)
			for _, diag := range result.Diagnostics {
				fmt.Printf("    %s:%d:%d [%s] %s\n",
					filepath.Base(f), diag.StartLine, diag.StartColumn,
					getSeverityLabel(diag.Severity), diag.Message)
			}
		}

		entries = append(entries, fileEntry{
			path:    f,
			content: content,
			result:  result,
		})
	}

	fmt.Println()

	if hasErrors {
		return fmt.Errorf("push aborted: fix validation errors before pushing")
	}

	if hasWarnings && !pushForce {
		return fmt.Errorf("push aborted: schemas have warnings (use --force to push anyway)")
	}

	if pushDryRun {
		fmt.Printf("\033[32mDry run complete:\033[0m %d file(s) valid, nothing pushed\n", len(entries))
		return nil
	}

	// Push to server.
	client, err := newAPIClient()
	if err != nil {
		return err
	}

	pushed := 0
	failed := 0

	for _, entry := range entries {
		name := schemaName(entry.path, schemaDir)
		payload := schemaSyncRequest{
			Name:    name,
			Content: entry.content,
		}

		respBody, err := client.apiRequest("POST", "/api/v1/schemas/sync", payload)
		if err != nil {
			fmt.Printf("\033[31m✗\033[0m %s — push failed: %v\n", entry.path, err)
			failed++
			continue
		}

		// Try to extract a server-side message if present.
		var respMsg struct {
			Message string `json:"message"`
		}
		_ = json.Unmarshal(respBody, &respMsg)

		if respMsg.Message != "" {
			fmt.Printf("\033[32m✓\033[0m %s — %s\n", entry.path, respMsg.Message)
		} else {
			fmt.Printf("\033[32m✓\033[0m %s — pushed\n", entry.path)
		}
		pushed++
	}

	fmt.Println()
	if failed > 0 {
		return fmt.Errorf("push completed with errors: %d pushed, %d failed", pushed, failed)
	}

	fmt.Printf("\033[32mPush complete:\033[0m %d schema(s) pushed successfully\n", pushed)
	return nil
}

// schemaName derives a schema name from a file path relative to the schema directory.
func schemaName(filePath, schemaDir string) string {
	rel, err := filepath.Rel(schemaDir, filePath)
	if err != nil {
		rel = filepath.Base(filePath)
	}
	// Strip .fsl extension and convert path separators to dots.
	rel = strings.TrimSuffix(rel, ".fsl")
	rel = strings.ReplaceAll(rel, string(filepath.Separator), ".")
	return rel
}
