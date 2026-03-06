package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/infrasutra/fsl/parser"
	"github.com/infrasutra/fsl/sdk"
	gosdk "github.com/infrasutra/fsl/sdk/go"
	"github.com/infrasutra/fsl/sdk/openapi"
	"github.com/infrasutra/fsl/sdk/python"
	"github.com/infrasutra/fsl/sdk/typescript"
	"github.com/spf13/cobra"
)

var (
	generateSchemaPath     string
	generateOutputPath     string
	generateClient         string
	generateTarget         string
	generateWorkspaceAPIID string
	generateExportFormat   string
)

var generateCmd = &cobra.Command{
	Use:   "generate [target]",
	Short: "Generate SDKs and code from schemas",
	Long: `Generate type-safe SDKs from FSL schemas.

Targets:
  typescript  Generate TypeScript SDK
  python      Generate Python SDK
  go          Generate Go SDK
  openapi     Generate OpenAPI or JSON Schema definitions

Examples:
  fluxcms generate typescript --schema=./schemas/ --output=./sdk/
  fluxcms generate python --schema=./schemas/ --output=./sdk/
  fluxcms generate go --schema=./schemas/ --output=./pkg/client
  fluxcms generate openapi --schema=./schemas/ --output=./definitions/`,
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

var generatePythonCmd = &cobra.Command{
	Use:   "python",
	Short: "Generate Python SDK",
	Long: `Generate a Python SDK from FSL schemas.

The generated SDK includes:
  - Pydantic models for all schema types
  - httpx-based API client for content delivery
  - Full Python type safety`,
	RunE: runGeneratePython,
}

var generateGoCmd = &cobra.Command{
	Use:   "go",
	Short: "Generate Go SDK",
	Long: `Generate a Go SDK from FSL schemas.

The generated SDK includes:
  - Go structs for all schema types
  - net/http based API client for content delivery
  - JSON tags for wire compatibility`,
	RunE: runGenerateGo,
}

var generateOpenAPICmd = &cobra.Command{
	Use:     "openapi",
	Aliases: []string{"jsonschema"},
	Short:   "Generate OpenAPI or JSON Schema definitions",
	Long: `Generate standard OpenAPI 3.0 or JSON Schema definitions from FSL schemas.

Formats:
  openapi    OpenAPI 3.0.3 document
  jsonschema JSON Schema draft 2020-12 document`,
	RunE: runGenerateOpenAPI,
}

func init() {
	rootCmd.AddCommand(generateCmd)
	generateCmd.AddCommand(generateTypescriptCmd)
	generateCmd.AddCommand(generatePythonCmd)
	generateCmd.AddCommand(generateGoCmd)
	generateCmd.AddCommand(generateOpenAPICmd)

	generateCmd.PersistentFlags().StringVar(&generateSchemaPath, "schema", "", "Schema file or directory")
	generateCmd.PersistentFlags().StringVar(&generateOutputPath, "output", "", "Output directory")
	generateTypescriptCmd.Flags().StringVar(&generateClient, "client", "fetch", "HTTP client: fetch, axios")
	generateTypescriptCmd.Flags().StringVar(&generateTarget, "target", "content", "API target: content or cms")
	generateTypescriptCmd.Flags().StringVar(&generateWorkspaceAPIID, "workspace-api-id", "", "Workspace API ID for content SDK default")
	generatePythonCmd.Flags().StringVar(&generateTarget, "target", "content", "API target: content or cms")
	generatePythonCmd.Flags().StringVar(&generateWorkspaceAPIID, "workspace-api-id", "", "Workspace API ID for content SDK default")
	generateGoCmd.Flags().StringVar(&generateTarget, "target", "content", "API target: content or cms")
	generateGoCmd.Flags().StringVar(&generateWorkspaceAPIID, "workspace-api-id", "", "Workspace API ID for content SDK default")
	generateOpenAPICmd.Flags().StringVar(&generateExportFormat, "format", "openapi", "Export format: openapi or jsonschema")
}

// generateParams holds the configuration for the shared generate logic.
type generateParams struct {
	language      string        // "TypeScript" or "Python"
	outputDefault string        // default output directory
	outputConfig  string        // output directory from config
	generator     sdk.Generator // the language-specific generator
	genConfig     sdk.GeneratorConfig
}

// runGenerate contains the shared logic for SDK generation.
func runGenerate(params generateParams) error {
	target := params.genConfig.TargetAPI
	switch target {
	case "content":
		// valid
	case "cms":
		return fmt.Errorf("CMS SDK generation requires schema IDs and should be generated via the server SDK endpoint")
	default:
		return fmt.Errorf("invalid target %q: must be content or cms", target)
	}

	schemaPath := resolvedSchemaPath()

	outputPath := generateOutputPath
	if outputPath == "" {
		if params.outputConfig != "" {
			outputPath = params.outputConfig
		} else {
			outputPath = params.outputDefault
		}
	}

	compiledSchemas, err := loadCompiledSchemas(schemaPath)
	if err != nil {
		return err
	}

	if err := os.MkdirAll(outputPath, 0o755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	baseURL := ""
	if cfg := GetConfig(); cfg != nil && cfg.Workspace.APIURL != "" {
		baseURL = cfg.Workspace.APIURL
	}
	params.genConfig.BaseURL = baseURL
	params.genConfig.WorkspaceAPIID = generateWorkspaceAPIID
	params.genConfig.IncludeClient = true

	generated, err := params.generator.Generate(compiledSchemas, params.genConfig)
	if err != nil {
		return fmt.Errorf("failed to generate %s SDK: %w", params.language, err)
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

	fmt.Printf("\033[32m✓\033[0m Generated %s SDK in %s\n", params.language, outputPath)
	fmt.Printf("  - schemas: %d\n", len(compiledSchemas))
	fmt.Printf("  - files (%d): %s\n", len(fileNames), strings.Join(fileNames, ", "))

	return nil
}

func runGenerateTypescript(cmd *cobra.Command, args []string) error {
	if generateClient != "fetch" {
		return fmt.Errorf("only fetch client is supported right now")
	}

	// Determine client type from config
	client := generateClient
	if cfg := GetConfig(); cfg != nil && cfg.Output.TypeScript.Client != "" && generateClient == "fetch" {
		client = cfg.Output.TypeScript.Client
	}
	if client != "fetch" {
		return fmt.Errorf("only fetch client is supported right now")
	}

	var outputConfig string
	if cfg := GetConfig(); cfg != nil && cfg.Output.TypeScript.Directory != "" {
		outputConfig = cfg.Output.TypeScript.Directory
	}

	return runGenerate(generateParams{
		language:      "TypeScript",
		outputDefault: "./sdk",
		outputConfig:  outputConfig,
		generator:     typescript.New(),
		genConfig: sdk.GeneratorConfig{
			StrictNullChecks: true,
			TargetAPI:        generateTarget,
		},
	})
}

func runGeneratePython(cmd *cobra.Command, args []string) error {
	var outputConfig string
	if cfg := GetConfig(); cfg != nil && cfg.Output.Python.Directory != "" {
		outputConfig = cfg.Output.Python.Directory
	}

	return runGenerate(generateParams{
		language:      "Python",
		outputDefault: "./sdk-python",
		outputConfig:  outputConfig,
		generator:     python.New(),
		genConfig: sdk.GeneratorConfig{
			TargetAPI: generateTarget,
		},
	})
}

func runGenerateGo(cmd *cobra.Command, args []string) error {
	var outputConfig string
	if cfg := GetConfig(); cfg != nil && cfg.Output.Go.Directory != "" {
		outputConfig = cfg.Output.Go.Directory
	}

	packageName := "client"
	if outputDir := generateOutputPath; outputDir != "" {
		packageName = filepath.Base(outputDir)
	} else if outputConfig != "" {
		packageName = filepath.Base(outputConfig)
	}

	return runGenerate(generateParams{
		language:      "Go",
		outputDefault: "./pkg/client",
		outputConfig:  outputConfig,
		generator:     gosdk.New(),
		genConfig: sdk.GeneratorConfig{
			PackageName: packageName,
			TargetAPI:   generateTarget,
		},
	})
}

func runGenerateOpenAPI(cmd *cobra.Command, args []string) error {
	schemaPath := resolvedSchemaPath()

	outputPath := generateOutputPath
	if outputPath == "" {
		outputPath = "./sdk-openapi"
	}

	compiledSchemas, err := loadCompiledSchemas(schemaPath)
	if err != nil {
		return err
	}

	if err := os.MkdirAll(outputPath, 0o755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	generated, err := openapi.New().Generate(compiledSchemas, sdk.GeneratorConfig{ExportFormat: generateExportFormat})
	if err != nil {
		return fmt.Errorf("failed to generate schema definitions: %w", err)
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

	fmt.Printf("\033[32m✓\033[0m Generated %s definitions in %s\n", strings.ToUpper(generateExportFormat), outputPath)
	fmt.Printf("  - schemas: %d\n", len(compiledSchemas))
	fmt.Printf("  - files (%d): %s\n", len(fileNames), strings.Join(fileNames, ", "))

	return nil
}

func resolvedSchemaPath() string {
	schemaPath := generateSchemaPath
	if schemaPath == "" {
		if cfg := GetConfig(); cfg != nil && cfg.Schemas.Directory != "" {
			schemaPath = cfg.Schemas.Directory
		} else {
			schemaPath = GetSchemaDirectory()
		}
	}

	return schemaPath
}

func loadCompiledSchemas(schemaPath string) ([]*parser.CompiledSchema, error) {
	files, err := collectFSLFiles([]string{schemaPath})
	if err != nil {
		return nil, err
	}

	fileContents := make(map[string]string, len(files))
	for _, file := range files {
		content, readErr := os.ReadFile(file)
		if readErr != nil {
			return nil, fmt.Errorf("cannot read %s: %w", file, readErr)
		}
		fileContents[file] = string(content)
	}

	results := parseFilesWithWorkspaceTypes(fileContents)

	compiledSchemas := make([]*parser.CompiledSchema, 0)
	for _, file := range files {
		result := results[file]
		if !result.Valid {
			message := "unknown error"
			if len(result.Diagnostics) > 0 {
				message = result.Diagnostics[0].Message
			}
			return nil, fmt.Errorf("schema validation failed in %s: %s", file, message)
		}

		for _, typeDef := range result.Schema.Types {
			derived := deriveApiID(typeDef.Name)
			compiled, compileErr := parser.Compile(result.Schema, typeDef.Name, derived, false)
			if compileErr != nil {
				return nil, fmt.Errorf("failed to compile schema %s in %s: %w", typeDef.Name, file, compileErr)
			}
			compiled.ApiID = derived
			compiledSchemas = append(compiledSchemas, compiled)
		}
	}

	return compiledSchemas, nil
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
