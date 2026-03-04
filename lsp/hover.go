package lsp

import (
	"fmt"
	"sort"
	"strings"

	"github.com/infrasutra/fsl/parser"
)

// Type documentation
var typeDocumentation = map[string]string{
	"String":   "Single-line text field.\n\nSupports decorators: `@maxLength`, `@minLength`, `@pattern`, `@unique`",
	"Text":     "Multi-line text field.\n\nSupports decorators: `@maxLength`, `@minLength`",
	"Int":      "Integer number field.\n\nSupports decorators: `@min`, `@max`, `@default`",
	"Float":    "Floating-point number field.\n\nSupports decorators: `@min`, `@max`, `@precision`, `@default`",
	"Boolean":  "True/false boolean field.\n\nSupports decorators: `@default`",
	"DateTime": "ISO 8601 date and time field.\n\nFormat: `2024-01-15T10:30:00Z`\n\nSupports decorators: `@default`",
	"Date":     "Date-only field.\n\nFormat: `2024-01-15`\n\nSupports decorators: `@default`",
	"JSON":     "Arbitrary JSON value field.\n\nAccepts any valid JSON: objects, arrays, strings, numbers, booleans, null.\n\nSupports decorators: `@slices` for typed slice zones.",
	"RichText": "Formatted content with block types.\n\nSupports decorators: `@blocks` to restrict allowed block types.\n\nValid blocks: paragraph, heading, blockquote, code, list, image, video, embed, table, divider, callout, toggle",
	"Image":    "Image asset reference.\n\nSupports decorators: `@maxSize`, `@formats`\n\nValid formats: jpg, jpeg, png, gif, webp, svg, avif",
	"File":     "File asset reference.\n\nSupports decorators: `@maxSize`, `@formats`",
	"Slug":     "URL-friendly identifier.\n\nAutomatically generates URL-safe slugs from text.",
}

// Decorator documentation
var decoratorDocumentation = map[string]string{
	"required":    "Mark the field as required.\n\n**Usage:** `@required` or `!` suffix on type",
	"default":     "Set a default value for the field.\n\n**Usage:** `@default(value)`\n\nExamples:\n- `@default(\"draft\")`\n- `@default(0)`\n- `@default(true)`",
	"unique":      "Enforce uniqueness constraint.\n\n**Usage:** `@unique`",
	"maxLength":   "Maximum string length.\n\n**Usage:** `@maxLength(200)`",
	"minLength":   "Minimum string length.\n\n**Usage:** `@minLength(1)`",
	"min":         "Minimum numeric value.\n\n**Usage:** `@min(0)`",
	"max":         "Maximum numeric value.\n\n**Usage:** `@max(100)`",
	"pattern":     "Regex validation pattern.\n\n**Usage:** `@pattern(\"^[a-z]+$\")`",
	"index":       "Create a database index on this field.\n\n**Usage:** `@index`",
	"searchable":  "Include field in full-text search.\n\n**Usage:** `@searchable`",
	"relation":    "Define a relation to another type.\n\n**Usage:** `@relation` or `@relation(inverse: \"field\", onDelete: \"restrict\")`\n\nThe field type should be another type name.",
	"blocks":      "Restrict allowed RichText block types.\n\n**Usage:** `@blocks(\"paragraph\", \"heading\")`",
	"slices":      "Define typed slice-zone variants for a JSON field.\n\n**Usage:** `@slices(hero: HeroSlice, faq: FaqSlice)`\n\nEach item must be `{ type: string, data: object }`. `type` is validated against the declared keys and `data` is validated against the mapped component type fields.",
	"maxSize":     "Maximum file size in bytes.\n\n**Usage:** `@maxSize(5242880)` (5MB)",
	"formats":     "Allowed file formats.\n\n**Usage:** `@formats(\"jpg\", \"png\", \"webp\")`",
	"precision":   "Decimal precision for Float fields.\n\n**Usage:** `@precision(2)`",
	"minItems":    "Minimum array length.\n\n**Usage:** `@minItems(1)`",
	"maxItems":    "Maximum array length.\n\n**Usage:** `@maxItems(10)`",
	"hidden":      "Hide field from API responses.\n\n**Usage:** `@hidden`",
	"label":       "Custom display label for UI.\n\n**Usage:** `@label(\"Display Name\")`",
	"help":        "Help text or tooltip.\n\n**Usage:** `@help(\"Enter your full name\")`",
	"placeholder": "Input placeholder text.\n\n**Usage:** `@placeholder(\"Enter value...\")`",
	"icon":        "Lucide icon name for the type.\n\n**Usage:** `@icon(\"file-text\")`\n\nSee https://lucide.dev for icon names.",
	"description": "Description for the type.\n\n**Usage:** `@description(\"Blog posts\")`",
	"collection":  "Custom collection name.\n\n**Usage:** `@collection(\"blog_posts\")`",
	"singleton":   "Mark type as singleton (single document).\n\n**Usage:** `@singleton`\n\nSingletons are useful for settings, site config, etc.",
}

// GetHover returns hover information for a position
func GetHover(doc *Document, pos Position) *Hover {
	word := doc.GetWordAt(pos)
	if word == "" {
		return nil
	}

	// Check if it's a type
	if doc, ok := typeDocumentation[word]; ok {
		return &Hover{
			Contents: MarkupContent{
				Kind:  MarkupKindMarkdown,
				Value: fmt.Sprintf("## %s\n\n%s", word, doc),
			},
		}
	}

	// Check if it's a decorator (remove @ prefix if present)
	decoratorName := strings.TrimPrefix(word, "@")
	if doc, ok := decoratorDocumentation[decoratorName]; ok {
		return &Hover{
			Contents: MarkupContent{
				Kind:  MarkupKindMarkdown,
				Value: fmt.Sprintf("## @%s\n\n%s", decoratorName, doc),
			},
		}
	}

	if fieldHover := getFieldHover(doc, pos, word); fieldHover != nil {
		return fieldHover
	}

	// Check if it's a custom type
	schema := doc.GetSchema()
	if schema != nil {
		for _, t := range schema.Types {
			if t.Name == word {
				return getTypeHover(&t)
			}
		}
		for _, e := range schema.Enums {
			if e.Name == word {
				return getEnumHover(&e)
			}
		}
	}

	// Check for keywords
	switch word {
	case "type":
		return &Hover{
			Contents: MarkupContent{
				Kind:  MarkupKindMarkdown,
				Value: "## type\n\nDefines a content type.\n\n```fsl\ntype Post {\n  title: String!\n  content: RichText\n}\n```",
			},
		}
	case "enum":
		return &Hover{
			Contents: MarkupContent{
				Kind:  MarkupKindMarkdown,
				Value: "## enum\n\nDefines an enumeration.\n\n```fsl\nenum Status {\n  draft\n  published\n  archived\n}\n```",
			},
		}
	}

	return nil
}

func getFieldHover(doc *Document, pos Position, word string) *Hover {
	line := doc.GetLine(pos.Line)
	if line == "" {
		return nil
	}
	colonIndex := strings.Index(line, ":")
	if colonIndex == -1 {
		return nil
	}
	fieldName := strings.TrimSpace(line[:colonIndex])
	if fieldName == "" || fieldName != word {
		return nil
	}

	schema := doc.GetSchema()
	if schema == nil {
		return nil
	}

	typeName := findEnclosingTypeName(doc, pos.Line)
	if typeName == "" {
		return nil
	}

	for _, t := range schema.Types {
		if t.Name != typeName {
			continue
		}
		for _, f := range t.Fields {
			if f.Name == fieldName {
				return buildFieldHover(&f)
			}
		}
	}

	return nil
}

func findEnclosingTypeName(doc *Document, lineNumber int) string {
	currentType := ""
	pendingType := ""
	braceDepth := 0

	maxLine := lineNumber
	if maxLine >= len(doc.Lines) {
		maxLine = len(doc.Lines) - 1
	}

	for i := 0; i <= maxLine; i++ {
		line := doc.GetLine(i)
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "type ") {
			name := parseTypeName(trimmed)
			if name != "" {
				if strings.Contains(trimmed, "{") {
					currentType = name
					pendingType = ""
				} else {
					pendingType = name
				}
			}
		}

		if strings.Contains(line, "{") {
			if pendingType != "" && currentType == "" && braceDepth == 0 {
				currentType = pendingType
				pendingType = ""
			}
		}

		braceDepth += strings.Count(line, "{")
		braceDepth -= strings.Count(line, "}")
		if braceDepth == 0 {
			currentType = ""
			pendingType = ""
		}
	}

	if braceDepth <= 0 {
		return ""
	}
	return currentType
}

func parseTypeName(line string) string {
	trimmed := strings.TrimSpace(line)
	trimmed = strings.TrimPrefix(trimmed, "type ")
	if trimmed == "" {
		return ""
	}
	fields := strings.Fields(trimmed)
	if len(fields) == 0 {
		return ""
	}
	name := strings.TrimSuffix(fields[0], "{")
	return strings.TrimSpace(name)
}

func buildFieldHover(field *parser.FieldDef) *Hover {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("## %s\n\n", field.Name))
	sb.WriteString(fmt.Sprintf("Type: `%s`\n", field.Type))
	if field.Required {
		sb.WriteString("Required: yes\n")
	} else {
		sb.WriteString("Required: no\n")
	}
	if field.Array {
		if field.ArrayReq {
			sb.WriteString("Array: yes (required)\n")
		} else {
			sb.WriteString("Array: yes\n")
		}
	} else {
		sb.WriteString("Array: no\n")
	}
	if field.IsRelation {
		sb.WriteString("Relation: yes\n")
	}

	if len(field.Decorators) > 0 {
		sb.WriteString("\nDecorators:\n")
		keys := make([]string, 0, len(field.Decorators))
		for name := range field.Decorators {
			keys = append(keys, name)
		}
		sort.Strings(keys)
		for _, name := range keys {
			value := field.Decorators[name]
			sb.WriteString(fmt.Sprintf("- %s\n", formatDecorator(name, value)))
		}
	}

	return &Hover{
		Contents: MarkupContent{
			Kind:  MarkupKindMarkdown,
			Value: sb.String(),
		},
	}
}

func formatDecorator(name string, value any) string {
	if value == nil || value == true {
		return fmt.Sprintf("@%s", name)
	}
	switch v := value.(type) {
	case string:
		return fmt.Sprintf("@%s(\"%s\")", name, v)
	case int, int64, float64, bool:
		return fmt.Sprintf("@%s(%v)", name, v)
	case []string:
		return fmt.Sprintf("@%s(%s)", name, formatStringSlice(v))
	case []any:
		return fmt.Sprintf("@%s(%s)", name, formatAnySlice(v))
	case map[string]any:
		return fmt.Sprintf("@%s(%s)", name, formatNamedArgs(v))
	default:
		return fmt.Sprintf("@%s(%v)", name, v)
	}
}

func formatStringSlice(values []string) string {
	if len(values) == 0 {
		return "[]"
	}
	parts := make([]string, 0, len(values))
	for _, value := range values {
		parts = append(parts, fmt.Sprintf("\"%s\"", value))
	}
	return "[" + strings.Join(parts, ", ") + "]"
}

func formatAnySlice(values []any) string {
	if len(values) == 0 {
		return "[]"
	}
	parts := make([]string, 0, len(values))
	for _, value := range values {
		parts = append(parts, formatDecoratorArg(value))
	}
	return "[" + strings.Join(parts, ", ") + "]"
}

func formatNamedArgs(values map[string]any) string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	parts := make([]string, 0, len(keys))
	for _, key := range keys {
		parts = append(parts, fmt.Sprintf("%s: %s", key, formatDecoratorArg(values[key])))
	}
	return strings.Join(parts, ", ")
}

func formatDecoratorArg(value any) string {
	switch v := value.(type) {
	case string:
		return fmt.Sprintf("\"%s\"", v)
	case []string:
		return formatStringSlice(v)
	case []any:
		return formatAnySlice(v)
	case map[string]any:
		return "{" + formatNamedArgs(v) + "}"
	default:
		return fmt.Sprintf("%v", v)
	}
}

func getTypeHover(t *parser.TypeDef) *Hover {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("## %s\n\n", t.Name))

	// Add description if present
	for _, dec := range t.Decorators {
		if dec.Name == "description" && len(dec.Args) > 0 {
			if desc, ok := dec.Args[0].(string); ok {
				sb.WriteString(fmt.Sprintf("%s\n\n", desc))
			}
		}
	}

	// List fields
	sb.WriteString("**Fields:**\n")
	for _, f := range t.Fields {
		required := ""
		if f.Required {
			required = "!"
		}
		array := ""
		if f.Array {
			array = "[]"
		}
		sb.WriteString(fmt.Sprintf("- `%s`: %s%s%s\n", f.Name, array, f.Type, required))
	}

	return &Hover{
		Contents: MarkupContent{
			Kind:  MarkupKindMarkdown,
			Value: sb.String(),
		},
	}
}

func getEnumHover(e *parser.EnumDef) *Hover {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("## %s (enum)\n\n", e.Name))
	sb.WriteString("**Values:**\n")
	for _, v := range e.Values {
		sb.WriteString(fmt.Sprintf("- `%s`\n", v))
	}

	return &Hover{
		Contents: MarkupContent{
			Kind:  MarkupKindMarkdown,
			Value: sb.String(),
		},
	}
}
