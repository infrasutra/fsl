package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/infrasutra/fsl/template"
	"github.com/spf13/cobra"
)

var templateCmd = &cobra.Command{
	Use:   "template",
	Short: "Manage schema templates",
	Long: `Manage schema templates for reuse across projects.

Templates are schema definitions that can be exported, imported,
and shared between workspaces or projects.

Supported formats:
  - YAML (.yaml, .yml) - Default format with all metadata
  - JSON (.json) - Alternative structured format
  - FSL (.fsl) - FSL with YAML frontmatter`,
}

var templateListCmd = &cobra.Command{
	Use:   "list",
	Short: "List available templates",
	Long: `List all available templates in the templates directory.

By default, looks for templates in ./templates/ directory.
Use --path to specify a different location.`,
	RunE: runTemplateList,
}

var templateValidateCmd = &cobra.Command{
	Use:   "validate <file>",
	Short: "Validate a template file",
	Long: `Validate a template file for correctness.

Checks:
  - Required fields (name, fsl)
  - Valid category (if specified)
  - FSL syntax validation`,
	Args: cobra.ExactArgs(1),
	RunE: runTemplateValidate,
}

var templateExportCmd = &cobra.Command{
	Use:   "export <slug> [output-file]",
	Short: "Export a template to a file",
	Long: `Export a template from the workspace to a file.

If no output file is specified, writes to <slug>.yaml in current directory.`,
	Args: cobra.RangeArgs(1, 2),
	RunE: runTemplateExport,
}

var templateCreateCmd = &cobra.Command{
	Use:   "create <file>",
	Short: "Create a template from a file",
	Long: `Create a new template in the workspace from a template file.

Supported formats:
  - YAML (.yaml, .yml)
  - JSON (.json)
  - FSL with frontmatter (.fsl)`,
	Args: cobra.ExactArgs(1),
	RunE: runTemplateCreate,
}

var templateImportCmd = &cobra.Command{
	Use:   "import <directory|file>",
	Short: "Import templates from a directory or file",
	Long: `Import one or more templates from files.

If a directory is specified, imports all .yaml, .yml, .json, and .fsl files.`,
	Args: cobra.ExactArgs(1),
	RunE: runTemplateImport,
}

var templateConvertCmd = &cobra.Command{
	Use:   "convert <input> <output>",
	Short: "Convert template between formats",
	Long: `Convert a template file between different formats.

Supported conversions:
  - YAML <-> JSON
  - YAML <-> FSL (with frontmatter)
  - JSON <-> FSL (with frontmatter)`,
	Args: cobra.ExactArgs(2),
	RunE: runTemplateConvert,
}

// Flags
var (
	templatePath     string
	templateCategory string
	templateFormat   string
)

func init() {
	rootCmd.AddCommand(templateCmd)

	// Add subcommands
	templateCmd.AddCommand(templateListCmd)
	templateCmd.AddCommand(templateValidateCmd)
	templateCmd.AddCommand(templateExportCmd)
	templateCmd.AddCommand(templateCreateCmd)
	templateCmd.AddCommand(templateImportCmd)
	templateCmd.AddCommand(templateConvertCmd)

	// Flags for list
	templateListCmd.Flags().StringVar(&templatePath, "path", "./templates", "Path to templates directory")
	templateListCmd.Flags().StringVar(&templateCategory, "category", "", "Filter by category (content, commerce, marketing, system, custom)")

	// Flags for export
	templateExportCmd.Flags().StringVar(&templateFormat, "format", "yaml", "Output format (yaml, json, fsl)")
}

func runTemplateList(cmd *cobra.Command, args []string) error {
	// Check if directory exists
	info, err := os.Stat(templatePath)
	if os.IsNotExist(err) {
		fmt.Printf("Templates directory '%s' does not exist.\n", templatePath)
		fmt.Println("\nTo create templates, use: fsl template create <file>")
		return nil
	}
	if err != nil {
		return fmt.Errorf("failed to access templates directory: %w", err)
	}
	if !info.IsDir() {
		return fmt.Errorf("'%s' is not a directory", templatePath)
	}

	// Find template files
	var templates []*template.TemplateFile
	var files []string

	err = filepath.Walk(templatePath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}

		ext := strings.ToLower(filepath.Ext(path))
		if ext == ".yaml" || ext == ".yml" || ext == ".json" || ext == ".fsl" {
			t, parseErr := template.ParseFile(path)
			if parseErr != nil {
				fmt.Printf("\033[33m⚠\033[0m  %s: %v\n", filepath.Base(path), parseErr)
				return nil
			}

			// Apply category filter
			if templateCategory != "" && t.Category != templateCategory {
				return nil
			}

			templates = append(templates, t)
			files = append(files, filepath.Base(path))
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("failed to read templates directory: %w", err)
	}

	if len(templates) == 0 {
		if templateCategory != "" {
			fmt.Printf("No templates found in category '%s'.\n", templateCategory)
		} else {
			fmt.Println("No templates found.")
		}
		return nil
	}

	// Print templates
	fmt.Printf("Found %d template(s):\n\n", len(templates))

	for i, t := range templates {
		fmt.Printf("  \033[1m%s\033[0m (%s)\n", t.Name, files[i])
		if t.Description != "" {
			fmt.Printf("    %s\n", t.Description)
		}
		if t.Category != "" {
			fmt.Printf("    Category: %s\n", t.Category)
		}
		if len(t.Tags) > 0 {
			fmt.Printf("    Tags: %s\n", strings.Join(t.Tags, ", "))
		}
		fmt.Println()
	}

	return nil
}

func runTemplateValidate(cmd *cobra.Command, args []string) error {
	path := args[0]

	// Parse and validate
	t, err := template.ParseFile(path)
	if err != nil {
		fmt.Printf("\033[31m✗\033[0m Template validation failed:\n")
		fmt.Printf("  %v\n", err)
		return nil // Don't return error to allow other processing
	}

	fmt.Printf("\033[32m✓\033[0m Template '%s' is valid\n", t.Name)
	fmt.Println()
	fmt.Println("Details:")
	fmt.Printf("  Name:        %s\n", t.Name)
	if t.Description != "" {
		fmt.Printf("  Description: %s\n", t.Description)
	}
	if t.Category != "" {
		fmt.Printf("  Category:    %s\n", t.Category)
	}
	if t.Icon != "" {
		fmt.Printf("  Icon:        %s\n", t.Icon)
	}
	if t.IsSingleton {
		fmt.Printf("  Singleton:   yes\n")
	}
	if len(t.Tags) > 0 {
		fmt.Printf("  Tags:        %s\n", strings.Join(t.Tags, ", "))
	}

	return nil
}

func runTemplateExport(cmd *cobra.Command, args []string) error {
	slug := args[0]

	// Determine output file
	var outputPath string
	if len(args) > 1 {
		outputPath = args[1]
	} else {
		ext := ".yaml"
		if templateFormat == "json" {
			ext = ".json"
		} else if templateFormat == "fsl" {
			ext = ".fsl"
		}
		outputPath = slug + ext
	}

	// For now, we can't actually export from the API without workspace context
	// This command would need API credentials to work with a remote workspace
	fmt.Printf("To export templates from a workspace, you need API credentials.\n")
	fmt.Printf("Configure in .fsl.yaml:\n\n")
	fmt.Printf("  workspace:\n")
	fmt.Printf("    api_url: \"https://your-api.com\"\n")
	fmt.Printf("    api_key: \"${FSL_API_KEY}\"\n\n")
	fmt.Printf("Then run: fsl template export %s %s\n", slug, outputPath)

	return nil
}

func runTemplateCreate(cmd *cobra.Command, args []string) error {
	path := args[0]

	// Parse and validate the template
	t, err := template.ParseFile(path)
	if err != nil {
		return fmt.Errorf("invalid template: %w", err)
	}

	fmt.Printf("\033[32m✓\033[0m Parsed template: %s\n", t.Name)

	// For now, we can't actually create in the API without workspace context
	fmt.Printf("\nTo create templates in a workspace, you need API credentials.\n")
	fmt.Printf("Configure in .fsl.yaml:\n\n")
	fmt.Printf("  workspace:\n")
	fmt.Printf("    api_url: \"https://your-api.com\"\n")
	fmt.Printf("    api_key: \"${FSL_API_KEY}\"\n\n")

	// Show what would be created
	fmt.Println("Template details:")
	fmt.Printf("  Name:     %s\n", t.Name)
	fmt.Printf("  Slug:     %s\n", template.GenerateSlug(t.Name))
	if t.Category != "" {
		fmt.Printf("  Category: %s\n", t.Category)
	}
	if len(t.Tags) > 0 {
		fmt.Printf("  Tags:     %s\n", strings.Join(t.Tags, ", "))
	}

	return nil
}

func runTemplateImport(cmd *cobra.Command, args []string) error {
	path := args[0]

	info, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("failed to access path: %w", err)
	}

	var files []string

	if info.IsDir() {
		// Find all template files in directory
		err = filepath.Walk(path, func(p string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if info.IsDir() {
				return nil
			}

			ext := strings.ToLower(filepath.Ext(p))
			if ext == ".yaml" || ext == ".yml" || ext == ".json" || ext == ".fsl" {
				files = append(files, p)
			}
			return nil
		})
		if err != nil {
			return fmt.Errorf("failed to read directory: %w", err)
		}
	} else {
		files = []string{path}
	}

	if len(files) == 0 {
		fmt.Println("No template files found.")
		return nil
	}

	fmt.Printf("Found %d template file(s) to import:\n\n", len(files))

	for _, f := range files {
		t, parseErr := template.ParseFile(f)
		if parseErr != nil {
			fmt.Printf("\033[31m✗\033[0m %s: %v\n", filepath.Base(f), parseErr)
			continue
		}
		fmt.Printf("\033[32m✓\033[0m %s: %s\n", filepath.Base(f), t.Name)
	}

	fmt.Printf("\nTo import into a workspace, configure API credentials in .fsl.yaml\n")

	return nil
}

func runTemplateConvert(cmd *cobra.Command, args []string) error {
	inputPath := args[0]
	outputPath := args[1]

	// Parse input
	t, err := template.ParseFile(inputPath)
	if err != nil {
		return fmt.Errorf("failed to parse input: %w", err)
	}

	// Determine output format from extension
	ext := strings.ToLower(filepath.Ext(outputPath))
	var format string
	switch ext {
	case ".json":
		format = "json"
	case ".fsl":
		format = "fsl"
	default:
		format = "yaml"
	}

	// Write output
	if err := template.WriteFile(t, outputPath, format); err != nil {
		return fmt.Errorf("failed to write output: %w", err)
	}

	fmt.Printf("\033[32m✓\033[0m Converted '%s' to '%s'\n", inputPath, outputPath)

	return nil
}
