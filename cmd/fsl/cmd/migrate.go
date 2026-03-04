package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/infrasutra/fsl/parser"
	"github.com/spf13/cobra"
)

var (
	migrateSchemaPath string
	migrateName       string
	migrateFormat     string
	migrateDiffFrom   string
	migrateDiffTo     string
	migrateDiffType   string
)

var migrateCmd = &cobra.Command{
	Use:   "migrate",
	Short: "Manage schema migrations",
	Long: `Generate and manage schema migrations for your FSL schemas.

Subcommands:
  generate  Generate a migration from schema changes
  preview   Preview migration without creating files
  check     Check for breaking changes
  diff      Diff two schemas`,
}

var migrateGenerateCmd = &cobra.Command{
	Use:   "generate",
	Short: "Generate a new migration",
	Long: `Generate a migration file based on schema changes.

Examples:
  fsl migrate generate --schema=./schemas/ --name="add_author_field"`,
	RunE: runMigrateGenerate,
}

var migratePreviewCmd = &cobra.Command{
	Use:   "preview",
	Short: "Preview migration changes",
	Long: `Preview what migration would be generated without creating files.

Examples:
  fsl migrate preview --schema=./schemas/`,
	RunE: runMigratePreview,
}

var migrateCheckCmd = &cobra.Command{
	Use:   "check",
	Short: "Check for breaking changes",
	Long: `Analyze schema changes and detect breaking changes.

Examples:
  fsl migrate check --schema=./schemas/`,
	RunE: runMigrateCheck,
}

var migrateDiffCmd = &cobra.Command{
	Use:   "diff",
	Short: "Diff two schemas",
	Long: `Compare two schema versions and show detailed changes.

Examples:
  fsl migrate diff --from=./schemas/v1 --to=./schemas/v2 --type=Post
  fsl migrate diff --from=./schema.fsl --to=./schema-next.fsl --format=json`,
	RunE: runMigrateDiff,
}

func init() {
	rootCmd.AddCommand(migrateCmd)
	migrateCmd.AddCommand(migrateGenerateCmd)
	migrateCmd.AddCommand(migratePreviewCmd)
	migrateCmd.AddCommand(migrateCheckCmd)
	migrateCmd.AddCommand(migrateDiffCmd)

	// Common flags
	migrateCmd.PersistentFlags().StringVar(&migrateSchemaPath, "schema", "", "Schema file or directory")
	migrateCmd.PersistentFlags().StringVar(&migrateFormat, "format", "pretty", "Output format: pretty, json")

	// Generate-specific flags
	migrateGenerateCmd.Flags().StringVar(&migrateName, "name", "", "Migration name (required)")
	migrateGenerateCmd.MarkFlagRequired("name")

	migrateDiffCmd.Flags().StringVar(&migrateDiffFrom, "from", "", "Schema file or directory to diff from")
	migrateDiffCmd.Flags().StringVar(&migrateDiffTo, "to", "", "Schema file or directory to diff to")
	migrateDiffCmd.Flags().StringVar(&migrateDiffType, "type", "", "Schema type name to diff")
	migrateDiffCmd.MarkFlagRequired("from")
	migrateDiffCmd.MarkFlagRequired("to")
}

type SchemaChange struct {
	Type        string `json:"type"` // "add_type", "remove_type", "add_field", "remove_field", "modify_field"
	TypeName    string `json:"typeName"`
	FieldName   string `json:"fieldName,omitempty"`
	Description string `json:"description"`
	Breaking    bool   `json:"breaking"`
}

type MigrationPreview struct {
	Changes       []SchemaChange `json:"changes"`
	BreakingCount int            `json:"breakingCount"`
	SafeCount     int            `json:"safeCount"`
}

func getSchemaPath() (string, error) {
	if migrateSchemaPath != "" {
		return migrateSchemaPath, nil
	}
	if config := GetConfig(); config != nil && config.Schemas.Directory != "" {
		return config.Schemas.Directory, nil
	}
	return "", fmt.Errorf("no schema path specified (use --schema flag or set schemas.directory in .fsl.yaml)")
}

func runMigrateGenerate(cmd *cobra.Command, args []string) error {
	path, err := getSchemaPath()
	if err != nil {
		return err
	}

	// Load and validate schemas
	_, err = loadSchemas(path)
	if err != nil {
		return err
	}

	// Generate migration file
	timestamp := time.Now().Format("20060102150405")
	safeName := strings.ReplaceAll(strings.ToLower(migrateName), " ", "_")
	filename := fmt.Sprintf("%s_%s.json", timestamp, safeName)

	migrationsDir := filepath.Join(filepath.Dir(path), "migrations")
	if err := os.MkdirAll(migrationsDir, 0o755); err != nil {
		return fmt.Errorf("failed to create migrations directory: %w", err)
	}

	migration := map[string]any{
		"version":   timestamp,
		"name":      migrateName,
		"createdAt": time.Now().Format(time.RFC3339),
		"changes":   []any{}, // Would be populated by comparing with previous schema state
	}

	migrationPath := filepath.Join(migrationsDir, filename)
	data, err := json.MarshalIndent(migration, "", "  ")
	if err != nil {
		return err
	}

	if err := os.WriteFile(migrationPath, data, 0o644); err != nil {
		return fmt.Errorf("failed to write migration: %w", err)
	}

	fmt.Printf("\033[32m✓\033[0m Created migration: %s\n", migrationPath)
	return nil
}

func runMigratePreview(cmd *cobra.Command, args []string) error {
	path, err := getSchemaPath()
	if err != nil {
		return err
	}

	schemas, err := loadSchemas(path)
	if err != nil {
		return err
	}

	// For now, just show schema structure as preview
	// In a full implementation, this would compare with stored schema state
	preview := MigrationPreview{
		Changes: []SchemaChange{},
	}

	for _, schema := range schemas {
		for _, t := range schema.Types {
			preview.Changes = append(preview.Changes, SchemaChange{
				Type:        "type_detected",
				TypeName:    t.Name,
				Description: fmt.Sprintf("Type '%s' with %d field(s)", t.Name, len(t.Fields)),
				Breaking:    false,
			})
			preview.SafeCount++
		}
		for _, e := range schema.Enums {
			preview.Changes = append(preview.Changes, SchemaChange{
				Type:        "enum_detected",
				TypeName:    e.Name,
				Description: fmt.Sprintf("Enum '%s' with %d value(s)", e.Name, len(e.Values)),
				Breaking:    false,
			})
			preview.SafeCount++
		}
	}

	switch migrateFormat {
	case "json":
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(preview)
	default:
		fmt.Println("Schema Preview:")
		fmt.Println()
		for _, change := range preview.Changes {
			icon := "\033[32m+\033[0m"
			if change.Breaking {
				icon = "\033[31m!\033[0m"
			}
			fmt.Printf("  %s %s: %s\n", icon, change.TypeName, change.Description)
		}
		fmt.Println()
		fmt.Printf("Total: %d type(s)/enum(s) detected\n", len(preview.Changes))
		return nil
	}
}

func runMigrateCheck(cmd *cobra.Command, args []string) error {
	path, err := getSchemaPath()
	if err != nil {
		return err
	}

	schemas, err := loadSchemas(path)
	if err != nil {
		return err
	}

	// Check for potential breaking changes
	var issues []SchemaChange

	for _, schema := range schemas {
		for _, t := range schema.Types {
			// Check for required fields without defaults (potential breaking change)
			for _, f := range t.Fields {
				if f.Required {
					if _, hasDefault := f.Decorators["default"]; !hasDefault {
						issues = append(issues, SchemaChange{
							Type:        "required_no_default",
							TypeName:    t.Name,
							FieldName:   f.Name,
							Description: fmt.Sprintf("Required field '%s.%s' has no default value", t.Name, f.Name),
							Breaking:    true,
						})
					}
				}
			}
		}
	}

	switch migrateFormat {
	case "json":
		result := map[string]any{
			"issues":        issues,
			"breakingCount": len(issues),
		}
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		if err := enc.Encode(result); err != nil {
			return err
		}
		if len(issues) > 0 {
			return fmt.Errorf("breaking changes detected")
		}
		return nil
	default:
		if len(issues) == 0 {
			fmt.Printf("\033[32m✓\033[0m No breaking changes detected\n")
			return nil
		}

		fmt.Printf("\033[33m!\033[0m Found %d potential issue(s):\n\n", len(issues))
		for _, issue := range issues {
			icon := "\033[33m⚠\033[0m"
			if issue.Breaking {
				icon = "\033[31m✗\033[0m"
			}
			fmt.Printf("  %s %s\n", icon, issue.Description)
		}
		fmt.Println()
		return fmt.Errorf("breaking changes detected")
	}
}

type schemaDiffOutput struct {
	Type string `json:"type"`
	*parser.SchemaDiff
}

func runMigrateDiff(cmd *cobra.Command, args []string) error {
	fromSchemas, err := loadSchemas(migrateDiffFrom)
	if err != nil {
		return err
	}

	toSchemas, err := loadSchemas(migrateDiffTo)
	if err != nil {
		return err
	}

	fromCompiled, fromTypes, err := compileSchemasByType(fromSchemas)
	if err != nil {
		return fmt.Errorf("failed to compile --from schemas: %w", err)
	}

	toCompiled, toTypes, err := compileSchemasByType(toSchemas)
	if err != nil {
		return fmt.Errorf("failed to compile --to schemas: %w", err)
	}

	typeName, err := resolveDiffType(migrateDiffType, fromTypes, toTypes)
	if err != nil {
		return err
	}

	fromSchema, ok := fromCompiled[typeName]
	if !ok {
		return fmt.Errorf("type '%s' not found in --from schemas; available types: %s", typeName, formatTypeList(fromTypes))
	}

	toSchema, ok := toCompiled[typeName]
	if !ok {
		return fmt.Errorf("type '%s' not found in --to schemas; available types: %s", typeName, formatTypeList(toTypes))
	}

	diff := parser.DiffSchemas(fromSchema, toSchema)

	switch migrateFormat {
	case "json":
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(schemaDiffOutput{Type: typeName, SchemaDiff: diff})
	default:
		fmt.Printf("Schema diff for type %s\n", typeName)
		fmt.Println(diff.Summary())
		if len(diff.Changes) == 0 {
			return nil
		}
		fmt.Println()

		added := []parser.SchemaChange{}
		removed := []parser.SchemaChange{}
		modified := []parser.SchemaChange{}
		for _, change := range diff.Changes {
			switch change.Type {
			case parser.ChangeTypeAdded:
				added = append(added, change)
			case parser.ChangeTypeRemoved:
				removed = append(removed, change)
			case parser.ChangeTypeModified:
				modified = append(modified, change)
			}
		}

		sort.Slice(added, func(i, j int) bool {
			return added[i].Path < added[j].Path
		})
		sort.Slice(removed, func(i, j int) bool {
			return removed[i].Path < removed[j].Path
		})
		sort.Slice(modified, func(i, j int) bool {
			return modified[i].Path < modified[j].Path
		})

		printSchemaDiffGroup("Added", added)
		printSchemaDiffGroup("Removed", removed)
		printSchemaDiffGroup("Modified", modified)
		return nil
	}
}

func printSchemaDiffGroup(label string, changes []parser.SchemaChange) {
	if len(changes) == 0 {
		return
	}
	fmt.Printf("%s (%d)\n", label, len(changes))
	for _, change := range changes {
		fmt.Printf("  - kind=%s path=%s breaking=%t message=%s\n", change.Kind, change.Path, change.Breaking, change.Message)
	}
}

func compileSchemasByType(schemas []*parser.Schema) (map[string]*parser.CompiledSchema, []string, error) {
	compiled := make(map[string]*parser.CompiledSchema)
	for _, schema := range schemas {
		for _, schemaType := range schema.Types {
			if _, exists := compiled[schemaType.Name]; exists {
				return nil, nil, fmt.Errorf("type '%s' defined multiple times", schemaType.Name)
			}
			result, err := parser.Compile(schema, schemaType.Name, schemaType.Name, false)
			if err != nil {
				return nil, nil, fmt.Errorf("failed to compile type '%s': %w", schemaType.Name, err)
			}
			compiled[schemaType.Name] = result
		}
	}

	if len(compiled) == 0 {
		return nil, nil, fmt.Errorf("no types found in schemas")
	}

	names := make([]string, 0, len(compiled))
	for name := range compiled {
		names = append(names, name)
	}
	sort.Strings(names)
	return compiled, names, nil
}

func resolveDiffType(requested string, fromTypes, toTypes []string) (string, error) {
	if requested != "" {
		if !containsString(fromTypes, requested) {
			return "", fmt.Errorf("type '%s' not found in --from schemas; available types: %s", requested, formatTypeList(fromTypes))
		}
		if !containsString(toTypes, requested) {
			return "", fmt.Errorf("type '%s' not found in --to schemas; available types: %s", requested, formatTypeList(toTypes))
		}
		return requested, nil
	}

	if len(fromTypes) == 1 && len(toTypes) == 1 && fromTypes[0] == toTypes[0] {
		return fromTypes[0], nil
	}

	return "", fmt.Errorf("multiple types detected; use --type. --from types: %s; --to types: %s", formatTypeList(fromTypes), formatTypeList(toTypes))
}

func formatTypeList(types []string) string {
	if len(types) == 0 {
		return "none"
	}
	return strings.Join(types, ", ")
}

func containsString(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}
