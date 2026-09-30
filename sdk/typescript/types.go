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
	parser.TypeDateTime: "string",
	parser.TypeDate:     "string",
	parser.TypeJSON:     "unknown",
	parser.TypeRichText: "RichTextBlock[]",
	parser.TypeImage:    "ImageAsset",
	parser.TypeFile:     "FileAsset",
	"Enum":              "string",
}

// MapFieldType converts an FSL field to its TypeScript type
func MapFieldType(field *parser.CompiledField) string {

	if field.Type == "Enum" && len(field.InlineEnum) > 0 {
		return buildUnionType(field.InlineEnum)
	}

	if field.IsRelation {
		if field.Array {
			return "RelationValue[]"
		}
		return "RelationValue"
	}

	if tsType, ok := TypeMapping[field.Type]; ok {
		return tsType
	}

	return field.Type
}

// MapFieldTypeWithNullability adds null type if field is optional
func MapFieldTypeWithNullability(field *parser.CompiledField, strictNullChecks bool) string {
	baseType := MapFieldType(field)

	if field.Array {
		elementType := baseType
		if field.IsRelation {

			return baseType
		}

		if field.Type == "Enum" && len(field.InlineEnum) > 0 {
			elementType = "(" + buildUnionType(field.InlineEnum) + ")"
		}

		if field.ArrayReq {
			return elementType + "[]"
		}
		if strictNullChecks {
			return elementType + "[] | null"
		}
		return elementType + "[]"
	}

	if !field.Required && strictNullChecks {
		return baseType + " | null"
	}
	return baseType
}

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

export interface RichTextMark {
  type: string;
  attrs?: Record<string, unknown>;
}

export interface RichTextBlock {
  type: string;
  content?: RichTextBlock[];
  text?: string;
  attrs?: Record<string, unknown>;
  marks?: RichTextMark[];
}

export interface DocumentReference {
  id: string;
}

export type RelationValue = string | DocumentReference;
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
  published_at?: string;
  included?: Record<string, unknown>;
}

export interface ContentListResponse<T> {
  payload: ContentItem<T>[];
  pagination: PaginationInfo;
  schema: SchemaRef;
}

export interface ContentListOptions {
  page?: number;
  limit?: number;
  locale?: string;
  sort?: string;
  filter?: string[];
  fields?: string[];
  include?: string[];
}
`
}

// ToPascalCase converts a string to PascalCase
func ToPascalCase(s string) string {
	if len(s) == 0 {
		return s
	}
	parts := strings.FieldsFunc(s, func(r rune) bool {
		return !(r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9')
	})
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
