package lsp

import (
	"strings"

	"github.com/infrasutra/fsl/parser"
)

// Built-in completions for FSL

var typeKeywords = []CompletionItem{
	{
		Label:            "type",
		Kind:             CompletionKindKeyword,
		Detail:           "Define a content type",
		InsertText:       "type ${1:Name} {\n\t$0\n}",
		InsertTextFormat: InsertTextFormatSnippet,
	},
	{
		Label:            "enum",
		Kind:             CompletionKindKeyword,
		Detail:           "Define an enumeration",
		InsertText:       "enum ${1:Name} {\n\t$0\n}",
		InsertTextFormat: InsertTextFormatSnippet,
	},
}

var fieldTypes = []CompletionItem{
	{Label: "String", Kind: CompletionKindClass, Detail: "Single-line text"},
	{Label: "Text", Kind: CompletionKindClass, Detail: "Multi-line text"},
	{Label: "Int", Kind: CompletionKindClass, Detail: "Integer number"},
	{Label: "Float", Kind: CompletionKindClass, Detail: "Floating-point number"},
	{Label: "Boolean", Kind: CompletionKindClass, Detail: "True/false value"},
	{Label: "DateTime", Kind: CompletionKindClass, Detail: "ISO 8601 date and time"},
	{Label: "Date", Kind: CompletionKindClass, Detail: "Date only (YYYY-MM-DD)"},
	{Label: "JSON", Kind: CompletionKindClass, Detail: "Any JSON value"},
	{Label: "RichText", Kind: CompletionKindClass, Detail: "Formatted content blocks"},
	{Label: "Image", Kind: CompletionKindClass, Detail: "Image asset reference"},
	{Label: "File", Kind: CompletionKindClass, Detail: "File asset reference"},
}

var fieldDecorators = []CompletionItem{
	{
		Label:         "@required",
		Kind:          CompletionKindProperty,
		Detail:        "Mark field as required",
		InsertText:    "@required",
		Documentation: "Equivalent to using ! after the type",
	},
	{
		Label:            "@default",
		Kind:             CompletionKindProperty,
		Detail:           "Set default value",
		InsertText:       "@default(${1:value})",
		InsertTextFormat: InsertTextFormatSnippet,
	},
	{
		Label:      "@unique",
		Kind:       CompletionKindProperty,
		Detail:     "Enforce uniqueness",
		InsertText: "@unique",
	},
	{
		Label:            "@maxLength",
		Kind:             CompletionKindProperty,
		Detail:           "Maximum string length",
		InsertText:       "@maxLength(${1:100})",
		InsertTextFormat: InsertTextFormatSnippet,
	},
	{
		Label:            "@minLength",
		Kind:             CompletionKindProperty,
		Detail:           "Minimum string length",
		InsertText:       "@minLength(${1:1})",
		InsertTextFormat: InsertTextFormatSnippet,
	},
	{
		Label:            "@min",
		Kind:             CompletionKindProperty,
		Detail:           "Minimum numeric value",
		InsertText:       "@min(${1:0})",
		InsertTextFormat: InsertTextFormatSnippet,
	},
	{
		Label:            "@max",
		Kind:             CompletionKindProperty,
		Detail:           "Maximum numeric value",
		InsertText:       "@max(${1:100})",
		InsertTextFormat: InsertTextFormatSnippet,
	},
	{
		Label:            "@pattern",
		Kind:             CompletionKindProperty,
		Detail:           "Regex validation pattern",
		InsertText:       "@pattern(\"${1:^[a-z]+$}\")",
		InsertTextFormat: InsertTextFormatSnippet,
	},
	{
		Label:      "@index",
		Kind:       CompletionKindProperty,
		Detail:     "Create database index",
		InsertText: "@index",
	},
	{
		Label:      "@searchable",
		Kind:       CompletionKindProperty,
		Detail:     "Include in full-text search",
		InsertText: "@searchable",
	},
	{
		Label:      "@relation",
		Kind:       CompletionKindProperty,
		Detail:     "Define relation to another type (supports inverse/onDelete args)",
		InsertText: "@relation",
	},
	{
		Label:            "@slices",
		Kind:             CompletionKindProperty,
		Detail:           "Typed slice-zone mappings for JSON",
		InsertText:       "@slices(${1:hero}: ${2:HeroSlice})",
		InsertTextFormat: InsertTextFormatSnippet,
	},
	{
		Label:            "@maxSize",
		Kind:             CompletionKindProperty,
		Detail:           "Maximum file size in bytes",
		InsertText:       "@maxSize(${1:5242880})",
		InsertTextFormat: InsertTextFormatSnippet,
	},
	{
		Label:            "@formats",
		Kind:             CompletionKindProperty,
		Detail:           "Allowed file formats",
		InsertText:       "@formats(\"${1:jpg}\", \"${2:png}\")",
		InsertTextFormat: InsertTextFormatSnippet,
	},
	{
		Label:            "@precision",
		Kind:             CompletionKindProperty,
		Detail:           "Decimal precision for Float",
		InsertText:       "@precision(${1:2})",
		InsertTextFormat: InsertTextFormatSnippet,
	},
	{
		Label:            "@minItems",
		Kind:             CompletionKindProperty,
		Detail:           "Minimum array length",
		InsertText:       "@minItems(${1:1})",
		InsertTextFormat: InsertTextFormatSnippet,
	},
	{
		Label:            "@maxItems",
		Kind:             CompletionKindProperty,
		Detail:           "Maximum array length",
		InsertText:       "@maxItems(${1:10})",
		InsertTextFormat: InsertTextFormatSnippet,
	},
	{
		Label:      "@hidden",
		Kind:       CompletionKindProperty,
		Detail:     "Hide field from API",
		InsertText: "@hidden",
	},
	{
		Label:            "@label",
		Kind:             CompletionKindProperty,
		Detail:           "Custom display label",
		InsertText:       "@label(\"${1:Label}\")",
		InsertTextFormat: InsertTextFormatSnippet,
	},
	{
		Label:            "@help",
		Kind:             CompletionKindProperty,
		Detail:           "Help text/tooltip",
		InsertText:       "@help(\"${1:Help text}\")",
		InsertTextFormat: InsertTextFormatSnippet,
	},
	{
		Label:            "@placeholder",
		Kind:             CompletionKindProperty,
		Detail:           "Input placeholder text",
		InsertText:       "@placeholder(\"${1:Enter value...}\")",
		InsertTextFormat: InsertTextFormatSnippet,
	},
}

var relationArgumentCompletions = []CompletionItem{
	{
		Label:            "inverse",
		Kind:             CompletionKindProperty,
		Detail:           "Inverse relation field",
		InsertText:       "inverse: \"${1:field}\"",
		InsertTextFormat: InsertTextFormatSnippet,
	},
	{
		Label:            "onDelete",
		Kind:             CompletionKindProperty,
		Detail:           "Delete behavior (cascade|restrict|setNull)",
		InsertText:       "onDelete: \"${1:cascade}\"",
		InsertTextFormat: InsertTextFormatSnippet,
	},
}

var typeDecorators = []CompletionItem{
	{
		Label:            "@icon",
		Kind:             CompletionKindProperty,
		Detail:           "Lucide icon name",
		InsertText:       "@icon(\"${1:file-text}\")",
		InsertTextFormat: InsertTextFormatSnippet,
	},
	{
		Label:            "@description",
		Kind:             CompletionKindProperty,
		Detail:           "Type description",
		InsertText:       "@description(\"${1:Description}\")",
		InsertTextFormat: InsertTextFormatSnippet,
	},
	{
		Label:            "@collection",
		Kind:             CompletionKindProperty,
		Detail:           "Custom collection name",
		InsertText:       "@collection(\"${1:name}\")",
		InsertTextFormat: InsertTextFormatSnippet,
	},
	{
		Label:      "@singleton",
		Kind:       CompletionKindProperty,
		Detail:     "Single-document type",
		InsertText: "@singleton",
	},
}

// GetCompletions returns completion items for a position
func GetCompletions(doc *Document, pos Position) *CompletionList {
	line := doc.GetLine(pos.Line)
	if line == "" {
		return &CompletionList{Items: typeKeywords}
	}

	// Get context before cursor
	prefix := ""
	if pos.Character <= len(line) {
		prefix = strings.TrimSpace(line[:pos.Character])
	}

	items := []CompletionItem{}

	if isInsideRelationDecorator(line, pos) {
		items = append(items, relationArgumentCompletions...)
		return &CompletionList{Items: items, IsIncomplete: false}
	}

	// Check context
	switch {
	case strings.HasPrefix(prefix, "@"):
		// Decorator context - determine if type-level or field-level
		if isAtTypeLevel(doc, pos) {
			items = append(items, typeDecorators...)
		} else {
			items = append(items, fieldDecorators...)
		}

	case strings.HasSuffix(prefix, ":"):
		// After colon - suggest types
		items = append(items, fieldTypes...)
		// Also suggest custom types from schema
		items = append(items, getCustomTypeCompletions(doc)...)

	case prefix == "" || prefix == "//" || strings.HasPrefix(prefix, "//"):
		// Empty line or comment - suggest type/enum keywords
		items = append(items, typeKeywords...)
		items = append(items, typeDecorators...)

	case strings.Contains(prefix, "{"):
		// Inside a type body - could be field definition
		items = append(items, fieldDecorators...)

	default:
		// Default - provide all completions
		items = append(items, typeKeywords...)
		items = append(items, fieldTypes...)
		items = append(items, fieldDecorators...)
	}

	return &CompletionList{
		IsIncomplete: false,
		Items:        items,
	}
}

func isInsideRelationDecorator(line string, pos Position) bool {
	if pos.Character > len(line) {
		return false
	}
	prefix := line[:pos.Character]
	idx := strings.LastIndex(prefix, "@relation(")
	if idx == -1 {
		return false
	}
	if strings.LastIndex(prefix, ")") > idx {
		return false
	}
	return true
}

// isAtTypeLevel determines if position is at type level (outside field definitions)
func isAtTypeLevel(doc *Document, pos Position) bool {
	// Simple heuristic: count braces before position
	braceCount := 0
	for i := 0; i < pos.Line; i++ {
		line := doc.GetLine(i)
		braceCount += strings.Count(line, "{") - strings.Count(line, "}")
	}

	// Also check current line up to cursor
	line := doc.GetLine(pos.Line)
	if pos.Character <= len(line) {
		partial := line[:pos.Character]
		braceCount += strings.Count(partial, "{") - strings.Count(partial, "}")
	}

	return braceCount == 0
}

// getCustomTypeCompletions returns completions for custom types defined in the schema
func getCustomTypeCompletions(doc *Document) []CompletionItem {
	schema := doc.GetSchema()
	if schema == nil {
		return nil
	}

	items := []CompletionItem{}

	// Add types
	for _, t := range schema.Types {
		items = append(items, CompletionItem{
			Label:  t.Name,
			Kind:   CompletionKindClass,
			Detail: "Content type (auto-relation)",
		})
	}

	// Add enums
	for _, e := range schema.Enums {
		items = append(items, CompletionItem{
			Label:  e.Name,
			Kind:   CompletionKindEnum,
			Detail: "Enum type",
		})
	}

	return items
}

// GetBuiltinTypes returns the list of built-in FSL types
func GetBuiltinTypes() []string {
	types := make([]string, 0, len(parser.BuiltinTypes))
	for t := range parser.BuiltinTypes {
		types = append(types, t)
	}
	return types
}
