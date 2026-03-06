package python

import (
	"strings"

	"github.com/infrasutra/fsl/parser"
)

// TypeMapping maps FSL types to Python types
var TypeMapping = map[string]string{
	parser.TypeString:   "str",
	parser.TypeText:     "str",
	parser.TypeInt:      "int",
	parser.TypeFloat:    "float",
	parser.TypeBoolean:  "bool",
	parser.TypeDateTime: "datetime",
	parser.TypeDate:     "date",
	parser.TypeJSON:     "dict[str, Any]",
	parser.TypeRichText: "dict[str, Any]",
	parser.TypeImage:    "ImageAsset",
	parser.TypeFile:     "FileAsset",
	"Enum":              "str",
}

// MapFieldType converts an FSL field to its Python type
func MapFieldType(field *parser.CompiledField) string {
	// Handle inline enums — use Literal union
	if field.Type == "Enum" && len(field.InlineEnum) > 0 {
		return buildLiteralType(field.InlineEnum)
	}

	// Handle relations
	if field.IsRelation {
		if field.Array {
			return "list[" + field.RelationTo + "]"
		}
		return field.RelationTo
	}

	// Handle named enums
	if pyType, ok := TypeMapping[field.Type]; ok {
		return pyType
	}

	// Default: use the FSL type name
	return field.Type
}

// MapFieldTypeWithOptional wraps with Optional if field is not required
func MapFieldTypeWithOptional(field *parser.CompiledField) string {
	baseType := MapFieldType(field)

	if field.Array {
		elementType := baseType
		if field.IsRelation {
			return baseType
		}
		if field.Type == "Enum" && len(field.InlineEnum) > 0 {
			elementType = buildLiteralType(field.InlineEnum)
		}
		if field.ArrayReq || field.Required {
			return "list[" + elementType + "]"
		}
		return "Optional[list[" + elementType + "]]"
	}

	if !field.Required {
		return "Optional[" + baseType + "]"
	}
	return baseType
}

// buildLiteralType creates a Python Literal type from string values
func buildLiteralType(values []string) string {
	if len(values) == 0 {
		return "str"
	}

	result := "Literal["
	for i, v := range values {
		if i > 0 {
			result += ", "
		}
		result += "\"" + v + "\""
	}
	result += "]"
	return result
}

// BuiltInTypes returns Python definitions for built-in asset types
func BuiltInTypes() string {
	return `
class ImageAsset(BaseModel):
    url: str
    width: Optional[int] = None
    height: Optional[int] = None
    alt: Optional[str] = None
    filename: Optional[str] = None
    size: Optional[int] = None
    mime_type: Optional[str] = None


class FileAsset(BaseModel):
    url: str
    filename: Optional[str] = None
    size: Optional[int] = None
    mime_type: Optional[str] = None

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
		runes := []rune(part)
		if runes[0] >= 'a' && runes[0] <= 'z' {
			runes[0] -= 32
		}
		b.WriteString(string(runes))
	}
	return b.String()
}

// ToSnakeCase converts a PascalCase/camelCase string to snake_case
func ToSnakeCase(s string) string {
	if len(s) == 0 {
		return s
	}

	runes := []rune(s)
	var result []rune
	for i, ch := range runes {
		if ch >= 'A' && ch <= 'Z' {
			if i > 0 {
				prev := runes[i-1]
				nextLower := i+1 < len(runes) && runes[i+1] >= 'a' && runes[i+1] <= 'z'
				if (prev >= 'a' && prev <= 'z') || (prev >= '0' && prev <= '9') || ((prev >= 'A' && prev <= 'Z') && nextLower) {
					result = append(result, '_')
				}
			}
			result = append(result, ch+32) // toLower
		} else {
			result = append(result, ch)
		}
	}
	return string(result)
}
