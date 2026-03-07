package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

var initCmd = &cobra.Command{
	Use:   "init [project-name]",
	Short: "Initialize a new FSL project",
	Long: `Create a new FSL project with the recommended directory structure.

This creates:
  - schemas/ directory with an example schema
  - .fluxcms.yaml configuration file
  - README.md with getting started instructions`,
	Args: cobra.MaximumNArgs(1),
	RunE: runInit,
}

func init() {
	rootCmd.AddCommand(initCmd)
}

func runInit(cmd *cobra.Command, args []string) error {
	projectDir := "."
	if len(args) > 0 {
		projectDir = args[0]
	}

	// Create project directory if it doesn't exist
	if projectDir != "." {
		if err := os.MkdirAll(projectDir, 0o755); err != nil {
			return fmt.Errorf("failed to create project directory: %w", err)
		}
	}

	// Create schemas directory
	schemasDir := filepath.Join(projectDir, "schemas")
	if err := os.MkdirAll(schemasDir, 0o755); err != nil {
		return fmt.Errorf("failed to create schemas directory: %w", err)
	}

	// Create example schema
	exampleSchema := `// Example blog post schema
// Learn more: https://github.com/infrasutra/fsl/tree/main/docs

@icon("file-text")
@description("Blog posts for your website")
type Post {
  title: String! @maxLength(200)
  slug: String! @unique @pattern("^[a-z0-9-]+$")
  excerpt: Text @maxLength(500)
  content: RichText!
  coverImage: Image
  publishedAt: DateTime
  status: "draft" | "published" | "archived" @default("draft")
  author: Author @relation
  tags: [String] @maxItems(10)
}

@icon("user")
@description("Content authors")
type Author {
  name: String!
  email: String! @unique @pattern("^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\\.[a-zA-Z]{2,}$")
  bio: Text
  avatar: Image
}

@icon("folder")
@description("Content categories")
type Category {
  name: String! @maxLength(100)
  slug: String! @unique @pattern("^[a-z0-9-]+$")
  description: Text
  parent: Category @relation
}

// Enum example
enum PostStatus {
  draft
  published
  archived
  scheduled
}
`
	examplePath := filepath.Join(schemasDir, "example.fsl")
	if err := os.WriteFile(examplePath, []byte(exampleSchema), 0o644); err != nil {
		return fmt.Errorf("failed to create example schema: %w", err)
	}

	// Create .fluxcms.yaml
	configContent := `version: "1"

# Workspace connection (optional - for remote sync)
# workspace:
#   api_url: "https://api.your-cms.com"
#   api_key: "${FSL_API_KEY}"

# Schema configuration
schemas:
  directory: "./schemas"

# SDK generation output
output:
  typescript:
    directory: "./sdk"
    client: "fetch"  # Options: fetch, axios
  go:
    directory: "./pkg/client"
`
	configPath := filepath.Join(projectDir, ".fluxcms.yaml")
	if err := os.WriteFile(configPath, []byte(configContent), 0o644); err != nil {
		return fmt.Errorf("failed to create config file: %w", err)
	}

	// Create README.md
	readmeContent := `# FSL Project

This project uses [FSL](https://github.com/infrasutra/fsl) for schema-first modeling and tooling.

## Getting Started

### Validate Schemas

` + "```bash" + `
fluxcms validate ./schemas/
` + "```" + `

### Generate TypeScript SDK

` + "```bash" + `
fluxcms generate typescript --schema=./schemas/ --output=./sdk/
` + "```" + `

### Generate Go SDK

` + "```bash" + `
fluxcms generate go --schema=./schemas/ --output=./pkg/client
` + "```" + `

### Check for Breaking Changes

` + "```bash" + `
fluxcms migrate check --schema=./schemas/
` + "```" + `

## Project Structure

` + "```" + `
.
├── schemas/           # FSL schema files
│   └── example.fsl    # Example schema
├── sdk/               # Generated TypeScript SDK (after generation)
├── pkg/client/        # Generated Go SDK (after generation)
├── .fluxcms.yaml          # Project configuration
└── README.md          # This file
` + "```" + `

## FSL Schema Language

FSL (Flux Schema Language) is a declarative schema definition language:

` + "```fsl" + `
@icon("file-text")
type Post {
  title: String! @maxLength(200)
  content: RichText!
  status: "draft" | "published"
}
` + "```" + `

Learn more in the [FSL Documentation](https://github.com/infrasutra/fsl/tree/main/docs).

## Editor Integration

Install the FSL extension for your editor:

- **VS Code**: Install ` + "`vscode-fsl`" + ` extension
- **Neovim/Vim**: Add FSL syntax and LSP configuration

Start the LSP server:

` + "```bash" + `
fluxcms lsp --stdio
` + "```" + `
`
	readmePath := filepath.Join(projectDir, "README.md")
	if err := os.WriteFile(readmePath, []byte(readmeContent), 0o644); err != nil {
		return fmt.Errorf("failed to create README: %w", err)
	}

	// Create .gitignore
	gitignoreContent := `# Generated files
sdk/
.fsl-state.json
*.log
`
	gitignorePath := filepath.Join(projectDir, ".gitignore")
	if err := os.WriteFile(gitignorePath, []byte(gitignoreContent), 0o644); err != nil {
		return fmt.Errorf("failed to create .gitignore: %w", err)
	}

	// Print success message
	fmt.Printf("\033[32m✓\033[0m Created FSL project")
	if projectDir != "." {
		fmt.Printf(" in %s", projectDir)
	}
	fmt.Println()
	fmt.Println()
	fmt.Println("Next steps:")
	if projectDir != "." {
		fmt.Printf("  cd %s\n", projectDir)
	}
	fmt.Println("  fluxcms validate ./schemas/  # Validate schemas")
	fmt.Println("  fluxcms generate typescript  # Generate SDK")
	fmt.Println()

	return nil
}
