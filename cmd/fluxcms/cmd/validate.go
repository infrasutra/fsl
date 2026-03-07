package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/infrasutra/fsl/parser"
	"github.com/spf13/cobra"
)

var validateFormat string
var validateLint bool

var validateCmd = &cobra.Command{
	Use:   "validate [path]",
	Short: "Validate FSL schema files",
	Long: `Validate one or more FSL schema files and report errors with line numbers.

Examples:
  # Validate a single file
  fluxcms validate schema.fsl

  # Validate all .fsl files in a directory
  fluxcms validate ./schemas/

  # Output in JSON format
  fluxcms validate schema.fsl --format=json`,
	Args: cobra.MinimumNArgs(1),
	RunE: runValidate,
}

func init() {
	rootCmd.AddCommand(validateCmd)
	validateCmd.Flags().StringVar(&validateFormat, "format", "pretty", "Output format: pretty, json")
	validateCmd.Flags().BoolVar(&validateLint, "lint", false, "Run lint rules and report warnings")
}

type ValidationResult struct {
	File        string              `json:"file"`
	Valid       bool                `json:"valid"`
	Diagnostics []parser.Diagnostic `json:"diagnostics"`
	Lines       []string            `json:"-"`
}

type ValidationReport struct {
	Results     []ValidationResult `json:"results"`
	TotalFiles  int                `json:"totalFiles"`
	ValidFiles  int                `json:"validFiles"`
	TotalErrors int                `json:"totalErrors"`
}

func runValidate(cmd *cobra.Command, args []string) error {
	files, err := collectFSLFiles(args)
	if err != nil {
		return err
	}

	fileContents := make(map[string]string, len(files))
	resultsByFile := make(map[string]ValidationResult, len(files))

	for _, file := range files {
		result := ValidationResult{
			File:        file,
			Valid:       true,
			Diagnostics: []parser.Diagnostic{},
		}

		content, readErr := os.ReadFile(file)
		if readErr != nil {
			result.Valid = false
			result.Diagnostics = append(result.Diagnostics, parser.Diagnostic{
				Severity:    parser.SeverityError,
				Message:     fmt.Sprintf("cannot read file: %v", readErr),
				StartLine:   1,
				StartColumn: 1,
				EndLine:     1,
				EndColumn:   1,
				Source:      "io",
			})
			resultsByFile[file] = result
			continue
		}

		result.Lines = strings.Split(string(content), "\n")
		resultsByFile[file] = result
		fileContents[file] = string(content)
	}

	diagResultsByFile := parseFilesWithWorkspaceTypes(fileContents)
	for file, diagResult := range diagResultsByFile {
		result := resultsByFile[file]
		result.Valid = diagResult.Valid
		result.Diagnostics = diagResult.Diagnostics

		if validateLint && diagResult.Valid && diagResult.Schema != nil {
			lintResults := parser.Lint(diagResult.Schema, parser.DefaultLinterConfig())
			for _, lr := range lintResults {
				severity := parser.SeverityWarning
				if lr.Rule.Severity == parser.LintHint {
					severity = parser.SeverityHint
				}
				result.Diagnostics = append(result.Diagnostics, parser.Diagnostic{
					Severity:    severity,
					Message:     lr.Message,
					StartLine:   lr.Line,
					StartColumn: lr.Column,
					EndLine:     lr.Line,
					EndColumn:   lr.Column,
					Source:      "lint",
				})
			}
		}

		resultsByFile[file] = result
	}

	report := ValidationReport{
		Results:    make([]ValidationResult, 0, len(files)),
		TotalFiles: len(files),
	}

	for _, file := range files {
		result := resultsByFile[file]
		report.Results = append(report.Results, result)
		if result.Valid {
			report.ValidFiles++
		}
		report.TotalErrors += len(result.Diagnostics)
	}

	switch validateFormat {
	case "json":
		return outputJSON(report)
	default:
		return outputPretty(report)
	}
}

func outputJSON(report ValidationReport) error {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(report)
}

func outputPretty(report ValidationReport) error {
	hasErrors := false

	for _, result := range report.Results {
		if result.Valid {
			if len(result.Diagnostics) > 0 {
				fmt.Printf("\033[33m⚠\033[0m %s\n", result.File)
			} else {
				fmt.Printf("\033[32m✓\033[0m %s\n", result.File)
			}
		} else {
			hasErrors = true
			fmt.Printf("\033[31m✗\033[0m %s\n", result.File)
		}
		for _, diag := range result.Diagnostics {
			if result.Valid && diag.Severity == parser.SeverityError {
				continue
			}
			severityColor := getSeverityColor(diag.Severity)
			severityLabel := getSeverityLabel(diag.Severity)
			source := diag.Source
			if source == "" {
				source = "fsl"
			}
			fmt.Printf("  %s%s\033[0m %d:%d [%s] - %s\n",
				severityColor,
				severityLabel,
				diag.StartLine,
				diag.StartColumn,
				source,
				diag.Message,
			)
			lineText, caretText := formatDiagnosticLine(result.Lines, diag.StartLine, diag.StartColumn)
			if lineText != "" {
				fmt.Printf("  > %s\n", lineText)
				if caretText != "" {
					fmt.Printf("  > %s\n", caretText)
				}
			}
		}
	}

	fmt.Println()
	if hasErrors {
		fmt.Printf("\033[31mValidation failed:\033[0m %d/%d files valid, %d error(s)\n",
			report.ValidFiles, report.TotalFiles, report.TotalErrors)
		return fmt.Errorf("validation failed")
	}

	fmt.Printf("\033[32mValidation passed:\033[0m %d file(s) valid\n", report.TotalFiles)
	return nil
}

func getSeverityColor(severity parser.DiagnosticSeverity) string {
	switch severity {
	case parser.SeverityError:
		return "\033[31m" // Red
	case parser.SeverityWarning:
		return "\033[33m" // Yellow
	case parser.SeverityInfo:
		return "\033[34m" // Blue
	case parser.SeverityHint:
		return "\033[36m" // Cyan
	default:
		return ""
	}
}

func getSeverityLabel(severity parser.DiagnosticSeverity) string {
	switch severity {
	case parser.SeverityError:
		return "error"
	case parser.SeverityWarning:
		return "warning"
	case parser.SeverityInfo:
		return "info"
	case parser.SeverityHint:
		return "hint"
	default:
		return "unknown"
	}
}

func formatDiagnosticLine(lines []string, lineNumber, columnNumber int) (string, string) {
	if lineNumber <= 0 || lineNumber > len(lines) {
		return "", ""
	}
	line := lines[lineNumber-1]
	if line == "" {
		return "", ""
	}
	if columnNumber <= 0 {
		columnNumber = 1
	}
	maxColumn := len(line) + 1
	if columnNumber > maxColumn {
		columnNumber = maxColumn
	}
	caret := strings.Repeat(" ", columnNumber-1) + "^"
	return line, caret
}
