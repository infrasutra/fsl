package parser

import (
	"fmt"
	"strings"
	"unicode"
)

// LintSeverity represents lint rule severity
type LintSeverity int

const (
	LintWarning LintSeverity = iota
	LintHint
)

// LintRule represents a single lint rule
type LintRule struct {
	Name     string
	Message  string
	Severity LintSeverity
}

// LintResult represents a lint finding
type LintResult struct {
	Rule      LintRule
	Message   string
	Line      int
	Column    int
	TypeName  string // parent type name for context-aware range finding
	FieldName string // field name for context-aware range finding
}

// LinterConfig controls which rules are enabled
type LinterConfig struct {
	NamingConvention      bool // types PascalCase, fields camelCase
	UnusedTypes           bool // warn on types not referenced by other types
	RequiredFieldOrdering bool // required fields before optional
	RelationCardinality   bool // warn on relations without explicit array marker
	MaxFieldCount         int  // warn if a type has too many fields (0 = disabled)
}

// DefaultLinterConfig returns a LinterConfig with all rules enabled and sensible defaults
func DefaultLinterConfig() LinterConfig {
	return LinterConfig{
		NamingConvention:      true,
		UnusedTypes:           true,
		RequiredFieldOrdering: true,
		RelationCardinality:   true,
		MaxFieldCount:         30,
	}
}

// Lint runs all enabled lint rules on a parsed schema.
func Lint(schema *Schema, config LinterConfig) []LintResult {
	var results []LintResult

	if config.NamingConvention {
		results = append(results, lintNamingConvention(schema)...)
	}
	if config.UnusedTypes {
		results = append(results, lintUnusedTypes(schema)...)
	}
	if config.RequiredFieldOrdering {
		results = append(results, lintRequiredFieldOrdering(schema)...)
	}
	if config.RelationCardinality {
		results = append(results, lintRelationCardinality(schema)...)
	}
	if config.MaxFieldCount > 0 {
		results = append(results, lintMaxFieldCount(schema, config.MaxFieldCount)...)
	}

	return results
}

func lintNamingConvention(schema *Schema) []LintResult {
	var results []LintResult
	rule := LintRule{Name: "naming-convention", Severity: LintWarning}

	for _, typeDef := range schema.Types {
		if !isPascalCase(typeDef.Name) {
			r := rule
			r.Message = fmt.Sprintf("type name '%s' should be PascalCase (first letter uppercase)", typeDef.Name)
			results = append(results, LintResult{Rule: r, Message: r.Message, TypeName: typeDef.Name})
		}
		for _, field := range typeDef.Fields {
			if !isCamelCase(field.Name) {
				r := rule
				r.Message = fmt.Sprintf("field name '%s' on type '%s' should be camelCase (first letter lowercase)", field.Name, typeDef.Name)
				results = append(results, LintResult{Rule: r, Message: r.Message, TypeName: typeDef.Name, FieldName: field.Name})
			}
		}
	}
	for _, enumDef := range schema.Enums {
		if !isPascalCase(enumDef.Name) {
			r := rule
			r.Message = fmt.Sprintf("enum name '%s' should be PascalCase (first letter uppercase)", enumDef.Name)
			results = append(results, LintResult{Rule: r, Message: r.Message, TypeName: enumDef.Name})
		}
	}
	return results
}

func lintUnusedTypes(schema *Schema) []LintResult {
	if len(schema.Types) <= 1 {
		return nil
	}
	referenced := make(map[string]bool)
	for _, typeDef := range schema.Types {
		for _, field := range typeDef.Fields {
			if field.IsRelation {
				referenced[field.Type] = true
			}
		}
	}
	rule := LintRule{Name: "unused-types", Severity: LintWarning}
	var results []LintResult
	for _, typeDef := range schema.Types {
		if !referenced[typeDef.Name] {
			r := rule
			r.Message = fmt.Sprintf("type '%s' is never referenced as a relation target by any other type", typeDef.Name)
			results = append(results, LintResult{Rule: r, Message: r.Message, TypeName: typeDef.Name})
		}
	}
	return results
}

func lintRequiredFieldOrdering(schema *Schema) []LintResult {
	rule := LintRule{Name: "required-field-ordering", Severity: LintWarning}
	var results []LintResult
	for _, typeDef := range schema.Types {
		seenOptional := false
		for _, field := range typeDef.Fields {
			isRequired := field.Required || (field.Array && field.ArrayReq)
			if !isRequired {
				seenOptional = true
			} else if seenOptional {
				r := rule
				r.Message = fmt.Sprintf("required field '%s' on type '%s' appears after optional fields; consider moving required fields first", field.Name, typeDef.Name)
				results = append(results, LintResult{Rule: r, Message: r.Message, TypeName: typeDef.Name, FieldName: field.Name})
			}
		}
	}
	return results
}

func lintRelationCardinality(schema *Schema) []LintResult {
	rule := LintRule{Name: "relation-cardinality", Severity: LintHint}
	var results []LintResult
	for _, typeDef := range schema.Types {
		for _, field := range typeDef.Fields {
			if field.IsRelation && !field.Array {
				r := rule
				r.Message = fmt.Sprintf("relation field '%s' on type '%s' is a singular relation; consider whether a one-to-many ([%s]) relation is intended", field.Name, typeDef.Name, field.Type)
				results = append(results, LintResult{Rule: r, Message: r.Message, TypeName: typeDef.Name, FieldName: field.Name})
			}
		}
	}
	return results
}

func lintMaxFieldCount(schema *Schema, maxCount int) []LintResult {
	rule := LintRule{Name: "max-field-count", Severity: LintWarning}
	var results []LintResult
	for _, typeDef := range schema.Types {
		if len(typeDef.Fields) > maxCount {
			r := rule
			r.Message = fmt.Sprintf("type '%s' has %d fields, which exceeds the recommended maximum of %d; consider splitting it into smaller types", typeDef.Name, len(typeDef.Fields), maxCount)
			results = append(results, LintResult{Rule: r, Message: r.Message, TypeName: typeDef.Name})
		}
	}
	return results
}

func isPascalCase(s string) bool {
	if s == "" {
		return false
	}
	return unicode.IsUpper([]rune(s)[0])
}

func isCamelCase(s string) bool {
	if s == "" {
		return false
	}
	return unicode.IsLower([]rune(s)[0])
}

// LintResultsToString formats lint results as human-readable lines.
func LintResultsToString(results []LintResult) string {
	if len(results) == 0 {
		return ""
	}
	var sb strings.Builder
	for _, r := range results {
		severity := "warning"
		if r.Rule.Severity == LintHint {
			severity = "hint"
		}
		if r.Line > 0 {
			fmt.Fprintf(&sb, "%s:%d:%d [%s] %s\n", severity, r.Line, r.Column, r.Rule.Name, r.Message)
		} else {
			fmt.Fprintf(&sb, "%s [%s] %s\n", severity, r.Rule.Name, r.Message)
		}
	}
	return sb.String()
}
