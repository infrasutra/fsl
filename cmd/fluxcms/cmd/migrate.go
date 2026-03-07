package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
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
	migrateDiffOutput string
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
  fluxcms migrate generate --schema=./schemas/ --name="add_author_field"`,
	RunE: runMigrateGenerate,
}

var migratePreviewCmd = &cobra.Command{
	Use:   "preview",
	Short: "Preview migration changes",
	Long: `Preview what migration would be generated without creating files.

Examples:
  fluxcms migrate preview --schema=./schemas/`,
	RunE: runMigratePreview,
}

var migrateCheckCmd = &cobra.Command{
	Use:   "check",
	Short: "Check for breaking changes",
	Long: `Analyze schema changes and detect breaking changes.

Examples:
  fluxcms migrate check --schema=./schemas/`,
	RunE: runMigrateCheck,
}

var migrateDiffCmd = &cobra.Command{
	Use:   "diff",
	Short: "Diff two schemas",
	Long: `Compare two schema versions and show detailed changes.

Examples:
  fluxcms migrate diff --from=./schemas/v1 --to=./schemas/v2 --type=Post
  fluxcms migrate diff --from=./schema.fsl --to=./schema-next.fsl --format=json`,
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
	migrateDiffCmd.Flags().StringVar(&migrateDiffOutput, "output", "", "Write diff output to file instead of stdout")
	migrateDiffCmd.MarkFlagRequired("from")
	migrateDiffCmd.MarkFlagRequired("to")
}

func getSchemaPath() (string, error) {
	if migrateSchemaPath != "" {
		return migrateSchemaPath, nil
	}
	if config := GetConfig(); config != nil && config.Schemas.Directory != "" {
		return config.Schemas.Directory, nil
	}
	return "", fmt.Errorf("no schema path specified (use --schema flag or set schemas.directory in .fluxcms.yaml)")
}

// loadPreviousState reads the most recent migration file and extracts stored compiled schemas.
// Returns nil if no previous state exists (first migration).
func loadPreviousState(migrationsDir string) (map[string]*parser.CompiledSchema, error) {
	entries, err := os.ReadDir(migrationsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to read migrations directory: %w", err)
	}

	var latestFile string
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".json") {
			if entry.Name() > latestFile {
				latestFile = entry.Name()
			}
		}
	}

	if latestFile == "" {
		return nil, nil
	}

	data, err := os.ReadFile(filepath.Join(migrationsDir, latestFile))
	if err != nil {
		return nil, fmt.Errorf("failed to read migration file: %w", err)
	}

	var migration struct {
		SchemaState map[string]*parser.CompiledSchema `json:"schemaState"`
	}
	if err := json.Unmarshal(data, &migration); err != nil {
		return nil, fmt.Errorf("failed to parse migration file: %w", err)
	}

	if migration.SchemaState == nil {
		fmt.Fprintf(os.Stderr, "\033[33m⚠\033[0m  Latest migration '%s' has no schema state (created before diff support).\n", latestFile)
		fmt.Fprintf(os.Stderr, "   Run: fluxcms migrate generate --name=baseline --schema=<path> to establish state.\n")
	}

	return migration.SchemaState, nil
}

// diffCurrentVsPrevious diffs all current compiled schemas against a previous state.
// If previous is nil, all types are treated as new additions.
func diffCurrentVsPrevious(current map[string]*parser.CompiledSchema, previous map[string]*parser.CompiledSchema) []parser.SchemaChange {
	var allChanges []parser.SchemaChange

	if previous == nil {
		for typeName := range current {
			allChanges = append(allChanges, parser.SchemaChange{
				Type:     parser.ChangeTypeAdded,
				Kind:     parser.ChangeKindType,
				Path:     fmt.Sprintf("types.%s", typeName),
				Breaking: false,
				Message:  fmt.Sprintf("type '%s' was added", typeName),
			})
		}
		return allChanges
	}

	// Check for removed types
	for typeName := range previous {
		if _, exists := current[typeName]; !exists {
			allChanges = append(allChanges, parser.SchemaChange{
				Type:     parser.ChangeTypeRemoved,
				Kind:     parser.ChangeKindType,
				Path:     fmt.Sprintf("types.%s", typeName),
				Breaking: true,
				Message:  fmt.Sprintf("type '%s' was removed", typeName),
			})
		}
	}

	// Check for added and modified types
	for typeName, currentSchema := range current {
		prevSchema, exists := previous[typeName]
		if !exists {
			allChanges = append(allChanges, parser.SchemaChange{
				Type:     parser.ChangeTypeAdded,
				Kind:     parser.ChangeKindType,
				Path:     fmt.Sprintf("types.%s", typeName),
				Breaking: false,
				Message:  fmt.Sprintf("type '%s' was added", typeName),
			})
			continue
		}

		diff := parser.DiffSchemas(prevSchema, currentSchema)
		for _, change := range diff.Changes {
			change.Path = fmt.Sprintf("%s.%s", typeName, change.Path)
			allChanges = append(allChanges, change)
		}
	}

	return allChanges
}

func runMigrateGenerate(cmd *cobra.Command, args []string) error {
	path, err := getSchemaPath()
	if err != nil {
		return err
	}

	schemas, err := loadSchemas(path)
	if err != nil {
		return err
	}

	currentCompiled, _, err := compileSchemasByType(schemas)
	if err != nil {
		return fmt.Errorf("failed to compile schemas: %w", err)
	}

	timestamp := time.Now().Format("20060102150405")
	safeName := strings.ReplaceAll(strings.ToLower(migrateName), " ", "_")
	filename := fmt.Sprintf("%s_%s.json", timestamp, safeName)

	migrationsDir := filepath.Join(filepath.Dir(path), "migrations")
	if err := os.MkdirAll(migrationsDir, 0o755); err != nil {
		return fmt.Errorf("failed to create migrations directory: %w", err)
	}

	previousState, err := loadPreviousState(migrationsDir)
	if err != nil {
		return fmt.Errorf("failed to load previous schema state: %w", err)
	}

	changes := diffCurrentVsPrevious(currentCompiled, previousState)

	migration := map[string]any{
		"version":     timestamp,
		"name":        migrateName,
		"createdAt":   time.Now().Format(time.RFC3339),
		"changes":     changes,
		"schemaState": currentCompiled,
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
	if len(changes) > 0 {
		breakingCount := 0
		for _, c := range changes {
			if c.Breaking {
				breakingCount++
			}
		}
		fmt.Printf("  %d change(s) recorded", len(changes))
		if breakingCount > 0 {
			fmt.Printf(", \033[31m%d breaking\033[0m", breakingCount)
		}
		fmt.Println()
	}
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

	currentCompiled, _, err := compileSchemasByType(schemas)
	if err != nil {
		return fmt.Errorf("failed to compile schemas: %w", err)
	}

	migrationsDir := filepath.Join(filepath.Dir(path), "migrations")
	previousState, err := loadPreviousState(migrationsDir)
	if err != nil {
		return fmt.Errorf("failed to load previous schema state: %w", err)
	}

	changes := diffCurrentVsPrevious(currentCompiled, previousState)

	breakingCount := 0
	safeCount := 0
	for _, c := range changes {
		if c.Breaking {
			breakingCount++
		} else {
			safeCount++
		}
	}

	switch migrateFormat {
	case "json":
		preview := map[string]any{
			"changes":       changes,
			"breakingCount": breakingCount,
			"safeCount":     safeCount,
		}
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(preview)
	default:
		if len(changes) == 0 {
			fmt.Println("No changes detected since last migration.")
			return nil
		}
		fmt.Println("Migration Preview:")
		fmt.Println()
		for _, change := range changes {
			icon := "\033[32m+\033[0m"
			switch {
			case change.Breaking:
				icon = "\033[31m!\033[0m"
			case change.Type == parser.ChangeTypeRemoved:
				icon = "\033[31m-\033[0m"
			case change.Type == parser.ChangeTypeModified:
				icon = "\033[33m~\033[0m"
			}
			fmt.Printf("  %s [%s] %s\n", icon, change.Path, change.Message)
		}
		fmt.Println()
		fmt.Printf("Total: %d change(s)", len(changes))
		if breakingCount > 0 {
			fmt.Printf(", \033[31m%d breaking\033[0m", breakingCount)
		}
		fmt.Println()
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

	currentCompiled, _, err := compileSchemasByType(schemas)
	if err != nil {
		return fmt.Errorf("failed to compile schemas: %w", err)
	}

	migrationsDir := filepath.Join(filepath.Dir(path), "migrations")
	previousState, err := loadPreviousState(migrationsDir)
	if err != nil {
		return fmt.Errorf("failed to load previous schema state: %w", err)
	}

	allChanges := diffCurrentVsPrevious(currentCompiled, previousState)

	var breakingChanges []parser.SchemaChange
	for _, c := range allChanges {
		if c.Breaking {
			breakingChanges = append(breakingChanges, c)
		}
	}

	switch migrateFormat {
	case "json":
		result := map[string]any{
			"issues":        breakingChanges,
			"breakingCount": len(breakingChanges),
		}
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		if err := enc.Encode(result); err != nil {
			return err
		}
		if len(breakingChanges) > 0 {
			return fmt.Errorf("breaking changes detected")
		}
		return nil
	default:
		if len(breakingChanges) == 0 {
			fmt.Printf("\033[32m✓\033[0m No breaking changes detected\n")
			return nil
		}

		fmt.Printf("\033[31m✗\033[0m Found %d breaking change(s):\n\n", len(breakingChanges))
		for _, issue := range breakingChanges {
			fmt.Printf("  \033[31m✗\033[0m [%s] %s\n", issue.Path, issue.Message)
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

	var out io.Writer = os.Stdout
	if migrateDiffOutput != "" {
		f, createErr := os.Create(migrateDiffOutput)
		if createErr != nil {
			return fmt.Errorf("failed to open output file: %w", createErr)
		}
		defer f.Close()
		out = f
	}

	switch migrateFormat {
	case "json":
		enc := json.NewEncoder(out)
		enc.SetIndent("", "  ")
		return enc.Encode(schemaDiffOutput{Type: typeName, SchemaDiff: diff})
	default:
		fmt.Fprintf(out, "Schema diff for type %s\n", typeName)
		fmt.Fprintln(out, diff.Summary())
		if len(diff.Changes) == 0 {
			return nil
		}
		fmt.Fprintln(out)

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

		writeSchemaDiffGroup(out, "Added", added)
		writeSchemaDiffGroup(out, "Removed", removed)
		writeSchemaDiffGroup(out, "Modified", modified)
		return nil
	}
}

func writeSchemaDiffGroup(out io.Writer, label string, changes []parser.SchemaChange) {
	if len(changes) == 0 {
		return
	}
	fmt.Fprintf(out, "%s (%d)\n", label, len(changes))
	for _, change := range changes {
		fmt.Fprintf(out, "  - kind=%s path=%s breaking=%t message=%s\n", change.Kind, change.Path, change.Breaking, change.Message)
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
		if !slices.Contains(fromTypes, requested) {
			return "", fmt.Errorf("type '%s' not found in --from schemas; available types: %s", requested, formatTypeList(fromTypes))
		}
		if !slices.Contains(toTypes, requested) {
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

