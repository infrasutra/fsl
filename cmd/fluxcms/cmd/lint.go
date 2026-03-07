package cmd

import (
	"fmt"
	"os"

	"github.com/infrasutra/fsl/parser"
	"github.com/spf13/cobra"
)

var lintCmd = &cobra.Command{
	Use:   "lint [files...]",
	Short: "Lint FSL schemas for style and best practices",
	Long: `Lint FSL schema files for style issues and best practices.

Unlike validate, lint produces warnings and hints rather than hard errors.
Exit code is 0 even when lint findings are reported (exit 1 only on parse errors).

Examples:
  fluxcms lint
  fluxcms lint schema.fsl
  fluxcms lint ./schemas/`,
	RunE: runLint,
}

func init() {
	rootCmd.AddCommand(lintCmd)
}

func runLint(cmd *cobra.Command, args []string) error {
	paths := args
	if len(paths) == 0 {
		paths = []string{GetSchemaDirectory()}
	}

	files, err := collectFSLFiles(paths)
	if err != nil {
		return err
	}

	hasParseErrors := false
	totalFindings := 0

	fileContents := make(map[string]string, len(files))
	for _, file := range files {
		content, readErr := os.ReadFile(file)
		if readErr != nil {
			fmt.Fprintf(os.Stderr, "error reading %s: %v\n", file, readErr)
			hasParseErrors = true
			continue
		}
		fileContents[file] = string(content)
	}

	diagResults := parseFilesWithWorkspaceTypes(fileContents)

	lintCfg := buildLintConfig()

	for _, file := range files {
		diagResult, ok := diagResults[file]
		if !ok {
			continue
		}

		if !diagResult.Valid {
			hasParseErrors = true
			fmt.Printf("\033[31m✗\033[0m %s (parse errors)\n", file)
			for _, d := range diagResult.Diagnostics {
				fmt.Printf("  \033[31merror\033[0m %d:%d [%s] %s\n",
					d.StartLine, d.StartColumn, d.Source, d.Message)
			}
			continue
		}

		results := parser.Lint(diagResult.Schema, lintCfg)
		if len(results) == 0 {
			fmt.Printf("\033[32m✓\033[0m %s\n", file)
			continue
		}

		fmt.Printf("\033[33m!\033[0m %s\n", file)
		for _, r := range results {
			color, label := lintSeverityFormat(r.Rule.Severity)
			fmt.Printf("  %s%s\033[0m [%s] %s\n", color, label, r.Rule.Name, r.Message)
		}
		totalFindings += len(results)
	}

	fmt.Println()
	if hasParseErrors {
		return fmt.Errorf("parse errors encountered")
	}

	if totalFindings > 0 {
		fmt.Printf("\033[33mLint complete:\033[0m %d finding(s) across %d file(s)\n", totalFindings, len(files))
	} else {
		fmt.Printf("\033[32mLint passed:\033[0m %d file(s) clean\n", len(files))
	}

	return nil
}

func lintSeverityFormat(s parser.LintSeverity) (string, string) {
	switch s {
	case parser.LintWarning:
		return "\033[33m", "warning"
	case parser.LintHint:
		return "\033[36m", "hint   "
	default:
		return "", "info   "
	}
}

func buildLintConfig() parser.LinterConfig {
	cfg := parser.DefaultLinterConfig()
	if config == nil {
		return cfg
	}

	lc := config.Lint
	if lc.NamingConvention != nil {
		cfg.NamingConvention = *lc.NamingConvention
	}
	if lc.UnusedTypes != nil {
		cfg.UnusedTypes = *lc.UnusedTypes
	}
	if lc.RequiredFieldOrdering != nil {
		cfg.RequiredFieldOrdering = *lc.RequiredFieldOrdering
	}
	if lc.RelationCardinality != nil {
		cfg.RelationCardinality = *lc.RelationCardinality
	}
	if lc.MaxFieldCount > 0 {
		cfg.MaxFieldCount = lc.MaxFieldCount
	}

	return cfg
}
