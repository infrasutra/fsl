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
	templateCmd.AddCommand(templateConvertCmd)

	// Flags for list
	templateListCmd.Flags().StringVar(&templatePath, "path", "./templates", "Path to templates directory")
	templateListCmd.Flags().StringVar(&templateCategory, "category", "", "Filter by category (content, commerce, marketing, system, custom)")

}

func runTemplateList(cmd *cobra.Command, args []string) error {
	// Check if directory exists
	info, err := os.Stat(templatePath)
	if os.IsNotExist(err) {
		fmt.Printf("Templates directory '%s' does not exist.\n", templatePath)
		fmt.Println("\nTo create templates, use: fluxcms template create <file>")
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
