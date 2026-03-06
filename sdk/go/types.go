package gosdk

import (
	"strings"

	"github.com/infrasutra/fsl/parser"
)

var TypeMapping = map[string]string{
	parser.TypeString:   "string",
	parser.TypeText:     "string",
	parser.TypeInt:      "int",
	parser.TypeFloat:    "float64",
	parser.TypeBoolean:  "bool",
	parser.TypeDateTime: "time.Time",
	parser.TypeDate:     "time.Time",
	parser.TypeJSON:     "map[string]any",
	parser.TypeRichText: "[]RichTextBlock",
	parser.TypeImage:    "ImageAsset",
	parser.TypeFile:     "FileAsset",
	"Enum":              "string",
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

func ToPascalCase(s string) string {
	if s == "" {
		return s
	}

	parts := strings.FieldsFunc(s, func(r rune) bool {
		return r == '_' || r == '-' || r == ' '
	})
	if len(parts) == 0 {
		parts = []string{s}
	}

	var b strings.Builder
	for _, part := range parts {
		if part == "" {
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

func ToCamelCase(s string) string {
	if s == "" {
		return s
	}
	result := []byte(s)
	result[0] = toLower(result[0])
	return string(result)
}

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
			result = append(result, ch+32)
			continue
		}
		result = append(result, ch)
	}

	return string(result)
}

func sanitizeIdentifier(s string) string {
	if s == "" {
		return "Value"
	}

	var b strings.Builder
	b.Grow(len(s))

	for i := 0; i < len(s); i++ {
		ch := s[i]
		if (ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') || (ch >= '0' && ch <= '9') {
			b.WriteByte(ch)
			continue
		}
		b.WriteByte('_')
	}

	clean := ToPascalCase(b.String())
	if clean == "" {
		return "Value"
	}
	if clean[0] >= '0' && clean[0] <= '9' {
		return "V" + clean
	}
	return clean
}
