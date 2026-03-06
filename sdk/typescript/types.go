package typescript

import (
	"strings"

	"github.com/infrasutra/fsl/parser"
)

// TypeMapping maps FSL types to TypeScript types
var TypeMapping = map[string]string{
	parser.TypeString:   "string",
	parser.TypeText:     "string",
	parser.TypeInt:      "number",
	parser.TypeFloat:    "number",
	parser.TypeBoolean:  "boolean",
	parser.TypeDateTime: "string", // ISO 8601 string
	parser.TypeDate:     "string", // YYYY-MM-DD string
	parser.TypeJSON:     "unknown",
	parser.TypeRichText: "RichTextBlock[]",
	parser.TypeImage:    "ImageAsset",
	parser.TypeFile:     "FileAsset",
	"Enum":              "string", // Inline enums become union types
}

// MapFieldType converts an FSL field to its TypeScript type
func MapFieldType(field *parser.CompiledField) string {
	// Handle inline enums
	if field.Type == "Enum" && len(field.InlineEnum) > 0 {
		return buildUnionType(field.InlineEnum)
	}

	// Handle relations
	if field.IsRelation {
		if field.Array {
			return field.RelationTo + "[]"
		}
		return field.RelationTo
	}

	// Handle named enums (use the enum name as the type)
	if tsType, ok := TypeMapping[field.Type]; ok {
		return tsType
	}

	// Default: use the FSL type name (for custom types)
	return field.Type
}

// MapFieldTypeWithNullability adds null type if field is optional
func MapFieldTypeWithNullability(field *parser.CompiledField, strictNullChecks bool) string {
	baseType := MapFieldType(field)

	if field.Array {
		elementType := baseType
		if field.IsRelation {
			// Relations already handled in MapFieldType
			return baseType
		}

		if field.Type == "Enum" && len(field.InlineEnum) > 0 {
			elementType = buildUnionType(field.InlineEnum)
		}

		// Array type
		if field.ArrayReq {
			return elementType + "[]"
		}
		if strictNullChecks {
			return elementType + "[] | null"
		}
		return elementType + "[]"
	}

	// Non-array field
	if !field.Required && strictNullChecks {
		return baseType + " | null"
	}
	return baseType
}

// buildUnionType creates a TypeScript union type from string values
func buildUnionType(values []string) string {
	if len(values) == 0 {
		return "string"
	}

	result := ""
	for i, v := range values {
		if i > 0 {
			result += " | "
		}
		result += "\"" + v + "\""
	}
	return result
}

// BuiltInTypes returns TypeScript definitions for built-in asset types
func BuiltInTypes() string {
	return `// Built-in asset types
export interface ImageAsset {
  url: string;
  width?: number;
  height?: number;
  alt?: string;
  filename?: string;
  size?: number;
  mimeType?: string;
}

export interface FileAsset {
  url: string;
  filename?: string;
  size?: number;
  mimeType?: string;
}

export interface RichTextBlock {
  type: string;
  children?: RichTextBlock[];
  text?: string;
  [key: string]: unknown;
}

// Reference type for relations
export interface DocumentReference {
  id: string;
}
`
}

// SharedResponseTypes returns TypeScript definitions for shared API responses
func SharedResponseTypes() string {
	return `// Shared response types
export interface ApiResponse<T> {
  success: boolean;
  payload: T;
  message: string;
}

export interface ApiListResponse<T> {
  success: boolean;
  payload: T;
  total_pages: number;
  message?: string;
}

export interface SchemaRef {
  id?: string;
  api_id: string;
  name: string;
}

export interface PaginationInfo {
  total: number;
  page: number;
  limit: number;
  total_pages: number;
  has_next: boolean;
  has_previous: boolean;
}

export interface UserRef {
  id: string;
  name: string;
}

export interface DocumentResponse<T> {
  id: string;
  workspace_id: string;
  schema: SchemaRef;
  slug?: string;
  locale: string;
  data: T;
  status: string;
  current_commit: string;
  created_at: string;
  updated_at: string;
  published_at?: string;
  scheduled_at?: string;
  created_by?: UserRef;
  updated_by?: UserRef;
}

export interface DocumentListItem {
  id: string;
  slug?: string;
  locale: string;
  status: string;
  current_commit: string;
  created_at: string;
  updated_at: string;
  published_at?: string;
  scheduled_at?: string;
}

export interface DocumentListResponse {
  documents: DocumentListItem[];
  pagination: PaginationInfo;
}

export interface ContentItem<T> {
  id: string;
  slug?: string;
  locale: string;
  data: T;
  created_at: string;
  updated_at: string;
  published_at: string;
  schema: SchemaRef;
}

export interface ContentListResponse<T> {
  data: ContentItem<T>[];
  pagination: PaginationInfo;
  schema: SchemaRef;
}
`
}

// ToPascalCase converts a string to PascalCase
func ToPascalCase(s string) string {
	if len(s) == 0 {
		return s
	}
	parts := strings.FieldsFunc(s, func(r rune) bool { return r == '_' || r == '-' })
	if len(parts) == 0 {
		return s
	}
	var b strings.Builder
	for _, part := range parts {
		if len(part) == 0 {
			continue
		}
		bs := []byte(part)
		bs[0] = toUpper(bs[0])
		b.Write(bs)
	}
	return b.String()
}

// ToCamelCase converts a string to camelCase
func ToCamelCase(s string) string {
	if len(s) == 0 {
		return s
	}

	result := []byte(s)
	result[0] = toLower(result[0])
	return string(result)
}

func toUpper(b byte) byte {
	if b >= 'a' && b <= 'z' {
		return b - 32
	}
	return b
}

func toLower(b byte) byte {
	if b >= 'A' && b <= 'Z' {
		return b + 32
	}
	return b
}
