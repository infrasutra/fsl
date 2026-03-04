package lsp

import (
	"fmt"
	"strings"

	"github.com/infrasutra/fsl/parser"
)

// GetDiagnostics returns LSP diagnostics for a document
func GetDiagnostics(doc *Document) []Diagnostic {
	result := doc.GetParseResult()
	if result == nil {
		return []Diagnostic{}
	}

	diagnostics := make([]Diagnostic, 0, len(result.Diagnostics))
	for _, d := range result.Diagnostics {
		diagnostics = append(diagnostics, ConvertDiagnostic(d))
	}

	schema := doc.GetSchema()
	if schema == nil {
		return diagnostics
	}

	enumNames := make(map[string]struct{}, len(schema.Enums))
	for _, e := range schema.Enums {
		enumNames[e.Name] = struct{}{}
	}

	usedEnums := make(map[string]struct{})
	for _, t := range schema.Types {
		for _, f := range t.Fields {
			if _, ok := enumNames[f.Type]; ok {
				usedEnums[f.Type] = struct{}{}
			}
		}
	}

	for _, e := range schema.Enums {
		if _, ok := usedEnums[e.Name]; ok {
			continue
		}
		rng := findEnumNameRange(doc, e.Name)
		diagnostics = append(diagnostics, Diagnostic{
			Range:    rng,
			Severity: SeverityWarning,
			Source:   "fsl",
			Message:  fmt.Sprintf("enum '%s' is not referenced by any field", e.Name),
		})
	}

	// Run lint rules and append as Warning/Hint diagnostics
	lintResults := parser.Lint(schema, parser.DefaultLinterConfig())
	for _, lr := range lintResults {
		sev := mapLintSeverity(lr.Rule.Severity)
		rng := findLintNameRange(doc, lr)
		diagnostics = append(diagnostics, Diagnostic{
			Range:    rng,
			Severity: sev,
			Source:   "fsl-lint",
			Message:  lr.Message,
			Code:     lr.Rule.Name,
		})
	}

	return diagnostics
}

func mapLintSeverity(s parser.LintSeverity) DiagnosticSeverity {
	switch s {
	case parser.LintWarning:
		return SeverityWarning
	case parser.LintHint:
		return SeverityHint
	default:
		return SeverityWarning
	}
}

func findLintNameRange(doc *Document, lr parser.LintResult) Range {
	// If FieldName is set, search for "<FieldName>:" after the line containing "type <TypeName>"
	if lr.FieldName != "" && lr.TypeName != "" {
		inType := false
		for lineIndex, line := range doc.Lines {
			trimmed := strings.TrimSpace(line)
			if !inType {
				if strings.HasPrefix(trimmed, "type ") && strings.Contains(trimmed, lr.TypeName) {
					inType = true
				}
				continue
			}
			// Check if we've left the type block
			if trimmed == "}" {
				inType = false
				continue
			}
			idx := strings.Index(line, lr.FieldName+":")
			if idx >= 0 {
				return Range{
					Start: Position{Line: lineIndex, Character: idx},
					End:   Position{Line: lineIndex, Character: idx + len(lr.FieldName)},
				}
			}
		}
	}

	// If only TypeName is set, search for "type <TypeName>" or "enum <TypeName>"
	if lr.TypeName != "" && lr.FieldName == "" {
		for lineIndex, line := range doc.Lines {
			trimmed := strings.TrimSpace(line)
			if strings.HasPrefix(trimmed, "type ") || strings.HasPrefix(trimmed, "enum ") {
				idx := strings.Index(line, lr.TypeName)
				if idx >= 0 {
					return Range{
						Start: Position{Line: lineIndex, Character: idx},
						End:   Position{Line: lineIndex, Character: idx + len(lr.TypeName)},
					}
				}
			}
		}
	}

	// Fallback: find first occurrence of the quoted name
	name := extractFirstQuoted(lr.Message)
	if name == "" {
		return Range{}
	}
	for lineIndex, line := range doc.Lines {
		idx := strings.Index(line, name)
		if idx >= 0 {
			return Range{
				Start: Position{Line: lineIndex, Character: idx},
				End:   Position{Line: lineIndex, Character: idx + len(name)},
			}
		}
	}
	return Range{}
}

func extractFirstQuoted(s string) string {
	start := strings.Index(s, "'")
	if start < 0 {
		return ""
	}
	end := strings.Index(s[start+1:], "'")
	if end < 0 {
		return ""
	}
	return s[start+1 : start+1+end]
}

func findEnumNameRange(doc *Document, enumName string) Range {
	for lineIndex, line := range doc.Lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "enum ") {
			parts := strings.Fields(trimmed)
			if len(parts) >= 2 && parts[1] == enumName {
				start := strings.Index(line, enumName)
				if start < 0 {
					start = 0
				}
				return Range{
					Start: Position{Line: lineIndex, Character: start},
					End:   Position{Line: lineIndex, Character: start + len(enumName)},
				}
			}
		}
	}

	return Range{
		Start: Position{Line: 0, Character: 0},
		End:   Position{Line: 0, Character: 0},
	}
}

// ConvertDiagnostic converts an FSL diagnostic to an LSP diagnostic
func ConvertDiagnostic(d parser.Diagnostic) Diagnostic {
	return Diagnostic{
		Range: Range{
			Start: Position{
				Line:      d.StartLine - 1,   // Convert to 0-indexed
				Character: d.StartColumn - 1, // Convert to 0-indexed
			},
			End: Position{
				Line:      d.EndLine - 1,
				Character: d.EndColumn - 1,
			},
		},
		Severity: mapSeverity(d.Severity),
		Source:   "fsl",
		Message:  d.Message,
	}
}

func mapSeverity(s parser.DiagnosticSeverity) DiagnosticSeverity {
	switch s {
	case parser.SeverityError:
		return SeverityError
	case parser.SeverityWarning:
		return SeverityWarning
	case parser.SeverityInfo:
		return SeverityInformation
	case parser.SeverityHint:
		return SeverityHint
	default:
		return SeverityError
	}
}
