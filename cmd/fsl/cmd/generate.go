package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/infrasutra/fsl/parser"
	"github.com/infrasutra/fsl/sdk"
	"github.com/infrasutra/fsl/sdk/typescript"
	"github.com/spf13/cobra"
)

var (
	generateSchemaPath     string
	generateOutputPath     string
	generateClient         string
	generateTarget         string
	generateWorkspaceAPIID string
)

var generateCmd = &cobra.Command{
	Use:   "generate [target]",
	Short: "Generate SDKs and code from schemas",
	Long: `Generate type-safe SDKs from FSL schemas.

Targets:
  typescript  Generate TypeScript SDK

Examples:
  fsl generate typescript --schema=./schemas/ --output=./sdk/
  fsl generate typescript --client=axios`,
}

var generateTypescriptCmd = &cobra.Command{
	Use:   "typescript",
	Short: "Generate TypeScript SDK",
	Long: `Generate a TypeScript SDK from FSL schemas.

The generated SDK includes:
  - Type definitions for all schema types
  - Client API for CRUD operations
  - Full TypeScript type safety`,
	RunE: runGenerateTypescript,
}

func init() {
	rootCmd.AddCommand(generateCmd)
	generateCmd.AddCommand(generateTypescriptCmd)

	generateCmd.PersistentFlags().StringVar(&generateSchemaPath, "schema", "", "Schema file or directory")
	generateCmd.PersistentFlags().StringVar(&generateOutputPath, "output", "", "Output directory")
	generateTypescriptCmd.Flags().StringVar(&generateClient, "client", "fetch", "HTTP client: fetch, axios")
	generateTypescriptCmd.Flags().StringVar(&generateTarget, "target", "content", "API target: content or cms")
	generateTypescriptCmd.Flags().StringVar(&generateWorkspaceAPIID, "workspace-api-id", "", "Workspace API ID for content SDK default")
}

func runGenerateTypescript(cmd *cobra.Command, args []string) error {
	if generateClient != "fetch" {
		return fmt.Errorf("only fetch client is supported right now")
	}

	target := generateTarget
	switch target {
	case "content":
		// valid
	case "cms":
		return fmt.Errorf("CMS SDK generation requires schema IDs and should be generated via the server SDK endpoint")
	default:
		return fmt.Errorf("invalid target %q: must be content or cms", target)
	}

	// Determine schema path
	schemaPath := generateSchemaPath
	if schemaPath == "" {
		if cfg := GetConfig(); cfg != nil && cfg.Schemas.Directory != "" {
			schemaPath = cfg.Schemas.Directory
		} else {
			schemaPath = GetSchemaDirectory()
		}
	}

	// Determine output path
	outputPath := generateOutputPath
	if outputPath == "" {
		if cfg := GetConfig(); cfg != nil && cfg.Output.TypeScript.Directory != "" {
			outputPath = cfg.Output.TypeScript.Directory
		} else {
			outputPath = "./sdk"
		}
	}

	// Determine client type
	client := generateClient
	if cfg := GetConfig(); cfg != nil && cfg.Output.TypeScript.Client != "" && generateClient == "fetch" {
		client = cfg.Output.TypeScript.Client
	}
	if client != "fetch" {
		return fmt.Errorf("only fetch client is supported right now")
	}

	files, err := collectFSLFiles([]string{schemaPath})
	if err != nil {
		return err
	}

	// Parse and compile all schemas
	compiledSchemas := make([]*parser.CompiledSchema, 0)
	for _, file := range files {
		content, err := os.ReadFile(file)
		if err != nil {
			return fmt.Errorf("cannot read %s: %w", file, err)
		}

		result := parser.ParseWithDiagnostics(string(content))
		if !result.Valid {
			message := "unknown error"
			if len(result.Diagnostics) > 0 {
				message = result.Diagnostics[0].Message
			}
			return fmt.Errorf("schema validation failed in %s: %s", file, message)
		}

		for _, typeDef := range result.Schema.Types {
			derived := deriveApiID(typeDef.Name)
			compiled, err := parser.Compile(result.Schema, typeDef.Name, derived, false)
			if err != nil {
				return fmt.Errorf("failed to compile schema %s in %s: %w", typeDef.Name, file, err)
			}
			compiled.ApiID = derived
			compiledSchemas = append(compiledSchemas, compiled)
		}
	}

	// Create output directory
	if err := os.MkdirAll(outputPath, 0o755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	baseURL := ""
	if cfg := GetConfig(); cfg != nil && cfg.Workspace.APIURL != "" {
		baseURL = cfg.Workspace.APIURL
	}

	generator := typescript.New()
	generated, err := generator.Generate(compiledSchemas, sdk.GeneratorConfig{
		BaseURL:          baseURL,
		WorkspaceAPIID:   generateWorkspaceAPIID,
		IncludeClient:    true,
		StrictNullChecks: true,
		TargetAPI:        target,
	})
	if err != nil {
		return fmt.Errorf("failed to generate TypeScript SDK: %w", err)
	}

	fileNames := make([]string, 0, len(generated.Files))
	for name := range generated.Files {
		fileNames = append(fileNames, name)
	}
	sort.Strings(fileNames)

	for _, name := range fileNames {
		content := generated.Files[name]
		filePath := filepath.Join(outputPath, name)
		if err := os.WriteFile(filePath, []byte(content), 0o644); err != nil {
			return fmt.Errorf("failed to write %s: %w", name, err)
		}
	}

	fmt.Printf("\033[32m✓\033[0m Generated TypeScript SDK in %s\n", outputPath)
	fmt.Printf("  - schemas: %d\n", len(compiledSchemas))
	fmt.Printf("  - files (%d): %s\n", len(fileNames), strings.Join(fileNames, ", "))

	return nil
}

func deriveApiID(name string) string {
	if name == "" {
		return ""
	}

	var sb strings.Builder
	sb.Grow(len(name))

	for i := 0; i < len(name); i++ {
		ch := name[i]
		if ch >= 'A' && ch <= 'Z' {
			if i > 0 {
				prev := name[i-1]
				nextLower := i+1 < len(name) && name[i+1] >= 'a' && name[i+1] <= 'z'
				if (prev >= 'a' && prev <= 'z') || (prev >= '0' && prev <= '9') || ((prev >= 'A' && prev <= 'Z') && nextLower) {
					sb.WriteByte('_')
				}
			}
			sb.WriteByte(ch + ('a' - 'A'))
			continue
		}
		sb.WriteByte(ch)
	}

	return sb.String()
}
